// Package diagnostics exposes the first-party browser event relay. It is a
// thin adapter: it admits the request and delegates envelope validation and
// forwarding to the observability browser tunnel.
package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/observability"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/pocketbase/pocketbase/core"
)

const (
	maxEnvelopeBytes   = 256 << 10
	bodyReadTimeout    = 5 * time.Second
	relayCapacity      = 16
	capacityRetryAfter = time.Second
)

type forwarder interface {
	Forward(ctx context.Context, body []byte) (observability.TunnelResult, error)
}

type relay struct {
	app      core.App
	tunnel   forwarder
	origin   *url.URL
	identity requesttrust.Resolver
	limiter  *rateLimiter
	capacity *requestprotection.CapacityGate
	logger   func(*core.RequestEvent) *slog.Logger
}

// RegisterHandlers registers the browser event relay when browser monitoring
// is configured. A nil tunnel registers nothing.
func RegisterHandlers(app core.App, tunnel *observability.BrowserTunnel, publicURL string, identity requesttrust.Resolver) error {
	if tunnel == nil {
		return nil
	}
	return register(app, tunnel, publicURL, identity)
}

func register(app core.App, tunnel forwarder, publicURL string, identity requesttrust.Resolver) error {
	if identity == nil {
		return fmt.Errorf("browser event relay requires a trusted client identity resolver")
	}
	origin, err := canonicalOrigin(publicURL)
	if err != nil {
		return err
	}
	limiter, err := newRateLimiter(time.Now)
	if err != nil {
		return err
	}
	capacity, err := requestprotection.NewCapacityGate(relayCapacity)
	if err != nil {
		return err
	}

	handler := &relay{app: app, tunnel: tunnel, origin: origin, identity: identity, limiter: limiter, capacity: capacity}
	handler.logger = func(e *core.RequestEvent) *slog.Logger { return logging.RequestLogger(app, e) }
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST(observability.BrowserTunnelPath, handler.serve)
		return se.Next()
	})

	return nil
}

func (h *relay) serve(e *core.RequestEvent) error {
	e.Response.Header().Set("Cache-Control", "no-store")

	if encoding := strings.TrimSpace(e.Request.Header.Get("Content-Encoding")); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return h.reject(e, http.StatusUnsupportedMediaType, "unsupported_encoding", 0)
	}
	if e.Request.ContentLength > maxEnvelopeBytes {
		return h.reject(e, http.StatusRequestEntityTooLarge, "too_large", 0)
	}
	if !h.sameOrigin(e.Request) {
		return h.reject(e, http.StatusForbidden, "cross_origin", 0)
	}
	clientID, ok := h.identity(e.Request)
	if !ok || clientID == "" {
		return h.reject(e, http.StatusForbidden, "missing_trusted_identity", 0)
	}
	if !h.limiter.allow(clientID) {
		return h.reject(e, http.StatusTooManyRequests, "rate_limited", rateLimitWindow)
	}
	// Read the bounded body under a short deadline before taking a capacity
	// slot, so slow uploads cannot hold forwarding capacity.
	_ = http.NewResponseController(e.Response).SetReadDeadline(time.Now().Add(bodyReadTimeout))
	body, err := io.ReadAll(http.MaxBytesReader(e.Response, e.Request.Body, maxEnvelopeBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return h.reject(e, http.StatusRequestEntityTooLarge, "too_large", 0)
		}
		return h.reject(e, http.StatusBadRequest, "unreadable_body", 0)
	}

	lease, ok := h.capacity.TryAcquire(e.Request.Context())
	if !ok {
		return h.reject(e, http.StatusServiceUnavailable, "capacity", capacityRetryAfter)
	}
	defer lease.Release()

	result, err := h.tunnel.Forward(e.Request.Context(), body)
	switch {
	case errors.Is(err, observability.ErrEnvelopeHeader):
		return h.reject(e, http.StatusBadRequest, "invalid_header", 0)
	case errors.Is(err, observability.ErrEnvelopeDestination):
		return h.reject(e, http.StatusForbidden, "unexpected_destination", 0)
	case err != nil:
		return h.reject(e, http.StatusBadGateway, "forward_failed", 0)
	}

	if result.RateLimits != "" {
		e.Response.Header().Set("X-Sentry-Rate-Limits", result.RateLimits)
	}
	if result.RetryAfter != "" {
		e.Response.Header().Set("Retry-After", result.RetryAfter)
	}
	if result.Status >= http.StatusInternalServerError {
		requestfailure.MarkExpectedResponse(e)
	}

	logger := h.logger(e)
	fields := []any{"event", "observability.browser_relay.completed", "outcome", string(result.Outcome), "upstream_status", result.Status, "size_bucket", sizeBucket(len(body))}
	if result.Outcome == observability.TunnelForwarded {
		logger.Debug("Browser event relayed", fields...)
	} else {
		logger.Warn("Browser event relay failed", fields...)
	}

	return e.NoContent(result.Status)
}

func (h *relay) reject(e *core.RequestEvent, status int, outcome string, retryAfter time.Duration) error {
	if retryAfter > 0 {
		e.Response.Header().Set("Retry-After", strconv.Itoa(int(retryAfter/time.Second)))
	}
	if status >= http.StatusInternalServerError {
		requestfailure.MarkExpectedResponse(e)
	}
	h.logger(e).Info("Browser event relay rejected request",
		"event", "observability.browser_relay.rejected",
		"outcome", outcome,
		"status", status,
	)
	return e.NoContent(status)
}

// sameOrigin compares Origin, or Referer when Origin is absent, with the
// configured public origin. Browsers always send Origin on fetch POSTs, so a
// request carrying neither header is left to identity and DSN validation.
func (h *relay) sameOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return h.matchesOrigin(origin)
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return h.matchesOrigin(referer)
	}
	return true
}

func (h *relay) matchesOrigin(raw string) bool {
	candidate, err := url.Parse(raw)
	if err != nil {
		return false
	}
	normalised, ok := normaliseOrigin(candidate)
	return ok && normalised.String() == h.origin.String()
}

func canonicalOrigin(publicURL string) (*url.URL, error) {
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("browser event relay: public URL must be a bare origin")
	}
	origin, ok := normaliseOrigin(parsed)
	if !ok {
		return nil, fmt.Errorf("browser event relay: public URL must be an http or https origin")
	}
	return origin, nil
}

func normaliseOrigin(u *url.URL) (*url.URL, bool) {
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Hostname() == "" {
		return nil, false
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		host = host + ":" + port
	}
	return &url.URL{Scheme: scheme, Host: host}, true
}

func sizeBucket(size int) string {
	switch {
	case size < 4<<10:
		return "<4KiB"
	case size < 32<<10:
		return "<32KiB"
	default:
		return ">=32KiB"
	}
}
