package diagnostics

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/observability"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const (
	testPublicURL  = "https://wga.example.org"
	testBrowserDSN = "https://browserkey@o1.ingest.de.sentry.io/42"
	testEnvelope   = `{"event_id":"abc","dsn":"` + testBrowserDSN + `"}` + "\n" + `{"type":"event"}` + "\n{}"
)

type fakeForwarder struct {
	mu     sync.Mutex
	calls  int
	result observability.TunnelResult
	err    error
}

func (f *fakeForwarder) Forward(context.Context, []byte) (observability.TunnelResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.result, f.err
}

func trustedIdentity(r *http.Request) (string, bool) { return r.RemoteAddr, true }

func newTestApp(t *testing.T) core.App {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ClearBootstrap(); err != nil {
			t.Errorf("reset bootstrap state: %v", err)
		}
	})
	return app
}

func newRelay(t *testing.T, app core.App, tunnel forwarder) *relay {
	t.Helper()
	origin, err := canonicalOrigin(testPublicURL)
	if err != nil {
		t.Fatal(err)
	}
	limiter, err := newRateLimiter(time.Now)
	if err != nil {
		t.Fatal(err)
	}
	capacity, err := requestprotection.NewCapacityGate(relayCapacity)
	if err != nil {
		t.Fatal(err)
	}
	h := &relay{app: app, tunnel: tunnel, origin: origin, identity: trustedIdentity, limiter: limiter, capacity: capacity}
	h.logger = func(*core.RequestEvent) *slog.Logger { return slog.New(slog.DiscardHandler) }
	return h
}

func relayRequest(body string, mutate func(*http.Request)) *http.Request {
	request := httptest.NewRequest(http.MethodPost, observability.BrowserTunnelPath, strings.NewReader(body))
	request.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	request.Header.Set("Origin", testPublicURL)
	request.RemoteAddr = "203.0.113.7:1234"
	if mutate != nil {
		mutate(request)
	}
	return request
}

func serve(t *testing.T, h *relay, request *http.Request) (*httptest.ResponseRecorder, *core.RequestEvent) {
	t.Helper()
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: h.app, Event: router.Event{Request: request, Response: recorder}}
	if err := h.serve(event); err != nil {
		t.Fatalf("serve() error = %v", err)
	}
	return recorder, event
}

func TestRelayForwardsAndPassesBackSentryRateLimits(t *testing.T) {
	tunnel := &fakeForwarder{result: observability.TunnelResult{Status: http.StatusTooManyRequests, RateLimits: "60:error", RetryAfter: "60", Outcome: observability.TunnelForwarded}}
	h := newRelay(t, newTestApp(t), tunnel)

	recorder, _ := serve(t, h, relayRequest(testEnvelope, nil))

	if tunnel.calls != 1 || recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("calls = %d, status = %d", tunnel.calls, recorder.Code)
	}
	if recorder.Header().Get("X-Sentry-Rate-Limits") != "60:error" || recorder.Header().Get("Retry-After") != "60" {
		t.Errorf("rate-limit headers not relayed: %v", recorder.Header())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Body.Len() != 0 {
		t.Errorf("response must be uncached and empty: %v %q", recorder.Header(), recorder.Body.String())
	}
}

func TestRelayRejectsWithoutForwarding(t *testing.T) {
	large := strings.Repeat("x", maxEnvelopeBytes+1)
	tests := []struct {
		name     string
		body     string
		mutate   func(*http.Request)
		identity func(*http.Request) (string, bool)
		err      error
		status   int
	}{
		{name: "cross origin", body: testEnvelope, mutate: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, status: http.StatusForbidden},
		{name: "cross origin referer", body: testEnvelope, mutate: func(r *http.Request) { r.Header.Del("Origin"); r.Header.Set("Referer", "https://evil.example/page") }, status: http.StatusForbidden},
		{name: "other port", body: testEnvelope, mutate: func(r *http.Request) { r.Header.Set("Origin", "https://wga.example.org:8443") }, status: http.StatusForbidden},
		{name: "untrusted identity", body: testEnvelope, identity: func(*http.Request) (string, bool) { return "", false }, status: http.StatusForbidden},
		{name: "encoded body", body: testEnvelope, mutate: func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, status: http.StatusUnsupportedMediaType},
		{name: "declared too large", body: large, status: http.StatusRequestEntityTooLarge},
		{name: "streamed too large", body: large, mutate: func(r *http.Request) { r.ContentLength = -1 }, status: http.StatusRequestEntityTooLarge},
		{name: "invalid header", body: testEnvelope, err: observability.ErrEnvelopeHeader, status: http.StatusBadRequest},
		{name: "unexpected destination", body: testEnvelope, err: observability.ErrEnvelopeDestination, status: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tunnel := &fakeForwarder{err: test.err}
			h := newRelay(t, newTestApp(t), tunnel)
			if test.identity != nil {
				h.identity = test.identity
			}

			recorder, _ := serve(t, h, relayRequest(test.body, test.mutate))

			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			if forwarded := tunnel.calls > 0 && test.err == nil; forwarded {
				t.Fatalf("request was forwarded %d times", tunnel.calls)
			}
		})
	}
}

func TestRelayAllowsCanonicalOriginVariants(t *testing.T) {
	for _, origin := range []string{"https://WGA.example.org", "https://wga.example.org:443"} {
		tunnel := &fakeForwarder{result: observability.TunnelResult{Status: http.StatusOK, Outcome: observability.TunnelForwarded}}
		h := newRelay(t, newTestApp(t), tunnel)
		recorder, _ := serve(t, h, relayRequest(testEnvelope, func(r *http.Request) { r.Header.Set("Origin", origin) }))
		if recorder.Code != http.StatusOK || tunnel.calls != 1 {
			t.Errorf("origin %q: status = %d, calls = %d", origin, recorder.Code, tunnel.calls)
		}
	}
}

func TestRelayLimitsEachClient(t *testing.T) {
	tunnel := &fakeForwarder{result: observability.TunnelResult{Status: http.StatusOK, Outcome: observability.TunnelForwarded}}
	h := newRelay(t, newTestApp(t), tunnel)

	for range rateLimitPerWindow {
		if recorder, _ := serve(t, h, relayRequest(testEnvelope, nil)); recorder.Code != http.StatusOK {
			t.Fatalf("request within budget rejected with %d", recorder.Code)
		}
	}
	recorder, _ := serve(t, h, relayRequest(testEnvelope, nil))
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "60" {
		t.Fatalf("over-budget request: status = %d, Retry-After = %q", recorder.Code, recorder.Header().Get("Retry-After"))
	}
	if tunnel.calls != rateLimitPerWindow {
		t.Fatalf("calls = %d, want %d", tunnel.calls, rateLimitPerWindow)
	}

	other, _ := serve(t, h, relayRequest(testEnvelope, func(r *http.Request) { r.RemoteAddr = "198.51.100.2:1" }))
	if other.Code != http.StatusOK {
		t.Fatalf("another client was limited with %d", other.Code)
	}
}

func TestRelayRejectsWhenCapacityIsExhausted(t *testing.T) {
	tunnel := &fakeForwarder{result: observability.TunnelResult{Status: http.StatusOK, Outcome: observability.TunnelForwarded}}
	h := newRelay(t, newTestApp(t), tunnel)
	for range relayCapacity {
		lease, ok := h.capacity.TryAcquire(context.Background())
		if !ok {
			t.Fatal("could not fill capacity")
		}
		t.Cleanup(lease.Release)
	}

	recorder, event := serve(t, h, relayRequest(testEnvelope, nil))

	if recorder.Code != http.StatusServiceUnavailable || recorder.Header().Get("Retry-After") != "1" || tunnel.calls != 0 {
		t.Fatalf("status = %d, Retry-After = %q, calls = %d", recorder.Code, recorder.Header().Get("Retry-After"), tunnel.calls)
	}
	if !requestfailure.IsExpectedResponse(event) {
		t.Fatal("capacity rejection must not be reported as a server fault")
	}
}

func TestRelayMarksUpstreamFailuresAsExpected(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusGatewayTimeout, http.StatusInternalServerError} {
		tunnel := &fakeForwarder{result: observability.TunnelResult{Status: status, Outcome: observability.TunnelUpstreamUnavailable}}
		h := newRelay(t, newTestApp(t), tunnel)

		recorder, event := serve(t, h, relayRequest(testEnvelope, nil))

		if recorder.Code != status || !requestfailure.IsExpectedResponse(event) {
			t.Errorf("status %d: code = %d, expected = %v", status, recorder.Code, requestfailure.IsExpectedResponse(event))
		}
	}
}

func TestRelayForwardsRealEnvelopeWithoutClientData(t *testing.T) {
	var upstream *http.Request
	var upstreamBody string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		upstream = r
		body, _ := io.ReadAll(r.Body)
		upstreamBody = string(body)
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	tunnel, err := observability.NewBrowserTunnel(testBrowserDSN, "", transport)
	if err != nil {
		t.Fatal(err)
	}
	h := newRelay(t, newTestApp(t), tunnel)

	recorder, _ := serve(t, h, relayRequest(testEnvelope, func(r *http.Request) {
		r.Header.Set("Cookie", "session=secret")
		r.Header.Set("X-Forwarded-For", "203.0.113.7")
		r.Header.Set("CF-Connecting-IP", "203.0.113.7")
		r.Header.Set("Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	}))

	if recorder.Code != http.StatusOK || upstream == nil {
		t.Fatalf("status = %d, upstream = %v", recorder.Code, upstream)
	}
	if upstreamBody != testEnvelope {
		t.Errorf("upstream body changed: %q", upstreamBody)
	}
	for name := range upstream.Header {
		if name != "Content-Type" {
			t.Errorf("client data leaked upstream in header %q", name)
		}
	}
}

func TestRelayIsRegisteredOnlyWithBrowserTunnel(t *testing.T) {
	for _, withTunnel := range []bool{false, true} {
		app := newTestApp(t)
		var tunnel *observability.BrowserTunnel
		if withTunnel {
			var err error
			tunnel, err = observability.NewBrowserTunnel(testBrowserDSN, "", roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := RegisterHandlers(app, tunnel, testPublicURL, trustedIdentity); err != nil {
			t.Fatalf("RegisterHandlers() error = %v", err)
		}

		pbRouter, err := apis.NewRouter(app)
		if err != nil {
			t.Fatalf("create router: %v", err)
		}
		serveEvent := &core.ServeEvent{App: app, Router: pbRouter}
		if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
			mux, err := event.Router.BuildMux()
			if err != nil {
				return err
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, relayRequest(testEnvelope, nil))
			want := http.StatusNotFound
			if withTunnel {
				want = http.StatusOK
			}
			if recorder.Code != want {
				t.Errorf("tunnel configured = %v: status = %d, want %d", withTunnel, recorder.Code, want)
			}
			return nil
		}); err != nil {
			t.Fatalf("trigger serve event: %v", err)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRelayLogsOutcomeWithoutEnvelopeContent(t *testing.T) {
	for _, test := range []struct {
		name   string
		tunnel *fakeForwarder
		origin string
	}{
		{name: "forwarded", tunnel: &fakeForwarder{result: observability.TunnelResult{Status: http.StatusOK, Outcome: observability.TunnelForwarded}}, origin: testPublicURL},
		{name: "upstream failure", tunnel: &fakeForwarder{result: observability.TunnelResult{Status: http.StatusBadGateway, Outcome: observability.TunnelUpstreamUnavailable}}, origin: testPublicURL},
		{name: "rejected", tunnel: &fakeForwarder{err: observability.ErrEnvelopeDestination}, origin: testPublicURL},
		{name: "cross origin", tunnel: &fakeForwarder{}, origin: "https://evil.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			h := newRelay(t, newTestApp(t), test.tunnel)
			h.logger = func(*core.RequestEvent) *slog.Logger {
				return slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			}

			serve(t, h, relayRequest(testEnvelope, func(r *http.Request) { r.Header.Set("Origin", test.origin) }))

			if logs.Len() == 0 {
				t.Fatal("relay wrote no log entry")
			}
			for _, secret := range []string{"browserkey", "event_id", "abc", testBrowserDSN, "203.0.113.7"} {
				if strings.Contains(logs.String(), secret) {
					t.Errorf("log contains %q: %s", secret, logs.String())
				}
			}
		})
	}
}

type capacityCheckingReader struct {
	t        *testing.T
	capacity *requestprotection.CapacityGate
	reader   io.Reader
}

func (r capacityCheckingReader) Read(p []byte) (int, error) {
	if inUse := r.capacity.InUse(); inUse != 0 {
		r.t.Errorf("capacity in use while reading the body: %d", inUse)
	}
	return r.reader.Read(p)
}

func TestRelayReadsBodyBeforeTakingCapacity(t *testing.T) {
	tunnel := &fakeForwarder{result: observability.TunnelResult{Status: http.StatusOK, Outcome: observability.TunnelForwarded}}
	h := newRelay(t, newTestApp(t), tunnel)
	request := relayRequest("", nil)
	request.Body = io.NopCloser(capacityCheckingReader{t: t, capacity: h.capacity, reader: strings.NewReader(testEnvelope)})

	recorder, _ := serve(t, h, request)

	if recorder.Code != http.StatusOK || tunnel.calls != 1 {
		t.Fatalf("status = %d, calls = %d", recorder.Code, tunnel.calls)
	}
}
