package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// BrowserTunnelPath is the first-party path the browser SDK posts envelopes
// to. It avoids tracking keywords so content blockers do not match it.
const BrowserTunnelPath = "/diagnostics/browser"

const (
	browserTunnelTimeout     = 5 * time.Second
	envelopeContentType      = "application/x-sentry-envelope"
	upstreamResponseDrainCap = 64 << 10
)

var (
	// ErrEnvelopeHeader reports a missing or malformed envelope header DSN.
	ErrEnvelopeHeader = errors.New("envelope header is missing or malformed")
	// ErrEnvelopeDestination reports an envelope addressed to a project other
	// than the configured browser project.
	ErrEnvelopeDestination = errors.New("envelope targets an unexpected project")
)

// TunnelOutcome classifies a forwarding attempt for logs.
type TunnelOutcome string

const (
	TunnelForwarded           TunnelOutcome = "forwarded"
	TunnelUpstreamUnavailable TunnelOutcome = "upstream_unavailable"
	TunnelUpstreamTimeout     TunnelOutcome = "upstream_timeout"
)

// TunnelResult is the part of Sentry's response relayed back to the browser.
type TunnelResult struct {
	Status     int
	RateLimits string
	RetryAfter string
	Outcome    TunnelOutcome
}

// BrowserTunnel forwards browser Sentry envelopes to the configured browser
// project. Its destination is fixed at construction from configuration, so no
// request data can redirect it.
type BrowserTunnel struct {
	browser  *sentry.Dsn
	server   *sentry.Dsn
	endpoint string
	client   *http.Client
}

// NewBrowserTunnel builds a tunnel for the configured browser DSN. It returns
// nil without error when browser monitoring is disabled. A nil transport uses
// a dedicated, uninstrumented default transport.
func NewBrowserTunnel(browserDSN string, serverDSN string, transport http.RoundTripper) (*BrowserTunnel, error) {
	if browserDSN == "" {
		return nil, nil
	}

	browser, err := sentry.NewDsn(browserDSN)
	if err != nil {
		return nil, fmt.Errorf("browser tunnel: invalid browser DSN")
	}

	var server *sentry.Dsn
	if serverDSN != "" {
		server, err = sentry.NewDsn(serverDSN)
		if err != nil {
			return nil, fmt.Errorf("browser tunnel: invalid server DSN")
		}
	}

	if transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}

	return &BrowserTunnel{
		browser:  browser,
		server:   server,
		endpoint: browser.GetAPIURL().String(),
		client: &http.Client{
			Transport: transport,
			Timeout:   browserTunnelTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Forward validates the envelope header and relays the unchanged body. It
// returns ErrEnvelopeHeader or ErrEnvelopeDestination without contacting
// Sentry when the envelope is not addressed to the browser project. Upstream
// transport failures are reported through the result, not as errors.
func (t *BrowserTunnel) Forward(ctx context.Context, body []byte) (TunnelResult, error) {
	if err := t.validate(body); err != nil {
		return TunnelResult{}, err
	}

	// Envelopes sent while a page unloads must still be delivered after the
	// browser drops the connection; the timeout keeps the work bounded.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), browserTunnelTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return TunnelResult{Status: http.StatusBadGateway, Outcome: TunnelUpstreamUnavailable}, nil
	}
	request.Header.Set("Content-Type", envelopeContentType)

	response, err := t.client.Do(request)
	if err != nil {
		if isTimeout(err) {
			return TunnelResult{Status: http.StatusGatewayTimeout, Outcome: TunnelUpstreamTimeout}, nil
		}
		return TunnelResult{Status: http.StatusBadGateway, Outcome: TunnelUpstreamUnavailable}, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, upstreamResponseDrainCap))
	_ = response.Body.Close()

	return TunnelResult{
		Status:     response.StatusCode,
		RateLimits: response.Header.Get("X-Sentry-Rate-Limits"),
		RetryAfter: response.Header.Get("Retry-After"),
		Outcome:    TunnelForwarded,
	}, nil
}

func (t *BrowserTunnel) validate(body []byte) error {
	line := body
	if end := bytes.IndexByte(body, '\n'); end >= 0 {
		line = body[:end]
	}

	dsn, err := envelopeHeaderDSN(line)
	if err != nil {
		return ErrEnvelopeHeader
	}

	target, err := sentry.NewDsn(dsn)
	if err != nil {
		return ErrEnvelopeHeader
	}
	if !sameProject(target, t.browser) || (t.server != nil && sameProject(target, t.server)) {
		return ErrEnvelopeDestination
	}

	return nil
}

// envelopeHeaderDSN reads the header's "dsn" member strictly. encoding/json
// struct decoding matches keys case-insensitively (including Unicode folds)
// and keeps the last duplicate, whereas Sentry reads the exact key, so a
// header with case-variant or repeated "dsn" keys could be validated against
// one DSN and ingested under another. Such headers are rejected.
func envelopeHeaderDSN(line []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(line))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return "", ErrEnvelopeHeader
	}

	dsn, found := "", false
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return "", ErrEnvelopeHeader
		}
		key, ok := token.(string)
		if !ok {
			return "", ErrEnvelopeHeader
		}
		if key != "dsn" {
			if strings.EqualFold(key, "dsn") {
				return "", ErrEnvelopeHeader
			}
			var skipped json.RawMessage
			if err := decoder.Decode(&skipped); err != nil {
				return "", ErrEnvelopeHeader
			}
			continue
		}
		if found {
			return "", ErrEnvelopeHeader
		}
		if err := decoder.Decode(&dsn); err != nil || dsn == "" {
			return "", ErrEnvelopeHeader
		}
		found = true
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') || !found {
		return "", ErrEnvelopeHeader
	}
	if decoder.More() {
		return "", ErrEnvelopeHeader
	}

	return dsn, nil
}

func sameProject(a *sentry.Dsn, b *sentry.Dsn) bool {
	return a.GetScheme() == b.GetScheme() &&
		a.GetHost() == b.GetHost() &&
		a.GetPort() == b.GetPort() &&
		a.GetPath() == b.GetPath() &&
		a.GetPublicKey() == b.GetPublicKey() &&
		a.GetProjectID() == b.GetProjectID()
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
