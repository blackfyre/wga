package observability

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

const (
	testBrowserDSN = "https://browserkey@o1.ingest.de.sentry.io/42"
	testServerDSN  = "https://serverkey@o1.ingest.de.sentry.io/7"
)

type recordingTransport struct {
	mu       sync.Mutex
	requests []*http.Request
	bodies   []string
	respond  func(*http.Request) (*http.Response, error)
}

func (r *recordingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	r.mu.Lock()
	body, _ := io.ReadAll(request.Body)
	r.requests = append(r.requests, request)
	r.bodies = append(r.bodies, string(body))
	r.mu.Unlock()
	if r.respond != nil {
		return r.respond(request)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}")), Request: request}, nil
}

func (r *recordingTransport) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

func envelope(dsn string) []byte {
	return []byte(`{"event_id":"abc","dsn":"` + dsn + `"}` + "\n" + `{"type":"event"}` + "\n" + `{"message":"boom"}`)
}

func newTestTunnel(t *testing.T, transport http.RoundTripper) *BrowserTunnel {
	t.Helper()
	tunnel, err := NewBrowserTunnel(testBrowserDSN, testServerDSN, transport)
	if err != nil || tunnel == nil {
		t.Fatalf("NewBrowserTunnel() = %v, %v", tunnel, err)
	}
	return tunnel
}

func TestBrowserTunnelIsDisabledWithoutBrowserDSN(t *testing.T) {
	tunnel, err := NewBrowserTunnel("", testServerDSN, nil)
	if tunnel != nil || err != nil {
		t.Fatalf("NewBrowserTunnel(\"\") = %v, %v, want nil, nil", tunnel, err)
	}
}

func TestBrowserTunnelForwardsMatchingEnvelopeToFixedEndpoint(t *testing.T) {
	transport := &recordingTransport{respond: func(request *http.Request) (*http.Response, error) {
		header := http.Header{}
		header.Set("X-Sentry-Rate-Limits", "60:error:organization")
		header.Set("Retry-After", "60")
		header.Set("Set-Cookie", "upstream=1")
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: header, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	}}
	tunnel := newTestTunnel(t, transport)
	body := envelope(testBrowserDSN)

	result, err := tunnel.Forward(context.Background(), body)
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}
	if transport.calls() != 1 {
		t.Fatalf("upstream calls = %d, want 1", transport.calls())
	}
	request := transport.requests[0]
	if got := request.URL.String(); got != "https://o1.ingest.de.sentry.io/api/42/envelope/" {
		t.Errorf("upstream URL = %q", got)
	}
	if request.Method != http.MethodPost || transport.bodies[0] != string(body) {
		t.Errorf("upstream request = %s with body %q, want unchanged POST body", request.Method, transport.bodies[0])
	}
	for name := range request.Header {
		if name != "Content-Type" {
			t.Errorf("upstream request carries unexpected header %q", name)
		}
	}
	if got := request.Header.Get("Content-Type"); got != envelopeContentType {
		t.Errorf("Content-Type = %q", got)
	}
	if result.Status != http.StatusTooManyRequests || result.RateLimits != "60:error:organization" || result.RetryAfter != "60" || result.Outcome != TunnelForwarded {
		t.Errorf("result = %+v", result)
	}
}

func TestBrowserTunnelRejectsOtherDestinationsWithoutUpstreamCall(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want error
	}{
		{name: "server project", body: envelope(testServerDSN), want: ErrEnvelopeDestination},
		{name: "other host", body: envelope("https://browserkey@evil.example/42"), want: ErrEnvelopeDestination},
		{name: "other port", body: envelope("https://browserkey@o1.ingest.de.sentry.io:8443/42"), want: ErrEnvelopeDestination},
		{name: "other scheme", body: envelope("http://browserkey@o1.ingest.de.sentry.io/42"), want: ErrEnvelopeDestination},
		{name: "other path", body: envelope("https://browserkey@o1.ingest.de.sentry.io/prefix/42"), want: ErrEnvelopeDestination},
		{name: "other key", body: envelope("https://otherkey@o1.ingest.de.sentry.io/42"), want: ErrEnvelopeDestination},
		{name: "other project", body: envelope("https://browserkey@o1.ingest.de.sentry.io/43"), want: ErrEnvelopeDestination},
		{name: "missing dsn", body: []byte(`{"event_id":"abc"}` + "\n{}"), want: ErrEnvelopeHeader},
		{name: "malformed dsn", body: envelope("not a dsn"), want: ErrEnvelopeHeader},
		{name: "malformed header", body: []byte("not json\n{}"), want: ErrEnvelopeHeader},
		{name: "empty body", body: nil, want: ErrEnvelopeHeader},
		{name: "case-variant key", body: []byte(`{"DSN":"` + testBrowserDSN + `"}` + "\n{}"), want: ErrEnvelopeHeader},
		{name: "unicode-folded key", body: []byte(`{"dſn":"` + testBrowserDSN + `"}` + "\n{}"), want: ErrEnvelopeHeader},
		{name: "server dsn shadowed by case variant", body: []byte(`{"dsn":"` + testServerDSN + `","DSN":"` + testBrowserDSN + `"}` + "\n{}"), want: ErrEnvelopeHeader},
		{name: "duplicate dsn key", body: []byte(`{"dsn":"` + testServerDSN + `","dsn":"` + testBrowserDSN + `"}` + "\n{}"), want: ErrEnvelopeHeader},
		{name: "non-string dsn", body: []byte(`{"dsn":42}` + "\n{}"), want: ErrEnvelopeHeader},
		{name: "trailing data after header", body: []byte(`{"dsn":"` + testBrowserDSN + `"} {}` + "\n{}"), want: ErrEnvelopeHeader},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &recordingTransport{}
			tunnel := newTestTunnel(t, transport)

			if _, err := tunnel.Forward(context.Background(), test.body); !errors.Is(err, test.want) {
				t.Fatalf("Forward() error = %v, want %v", err, test.want)
			}
			if transport.calls() != 0 {
				t.Fatalf("upstream calls = %d, want 0", transport.calls())
			}
		})
	}
}

func TestBrowserTunnelDoesNotFollowRedirects(t *testing.T) {
	transport := &recordingTransport{respond: func(request *http.Request) (*http.Response, error) {
		header := http.Header{}
		header.Set("Location", "https://evil.example/")
		return &http.Response{StatusCode: http.StatusFound, Header: header, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	}}
	tunnel := newTestTunnel(t, transport)

	result, err := tunnel.Forward(context.Background(), envelope(testBrowserDSN))
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}
	if transport.calls() != 1 || result.Status != http.StatusFound {
		t.Fatalf("calls = %d, status = %d; want one call and the redirect status relayed", transport.calls(), result.Status)
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestBrowserTunnelMapsUpstreamFailures(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		outcome TunnelOutcome
	}{
		{name: "timeout", err: timeoutError{}, status: http.StatusGatewayTimeout, outcome: TunnelUpstreamTimeout},
		{name: "transport failure", err: errors.New("connection refused"), status: http.StatusBadGateway, outcome: TunnelUpstreamUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &recordingTransport{respond: func(*http.Request) (*http.Response, error) { return nil, test.err }}
			tunnel := newTestTunnel(t, transport)

			result, err := tunnel.Forward(context.Background(), envelope(testBrowserDSN))
			if err != nil {
				t.Fatalf("Forward() error = %v", err)
			}
			if result.Status != test.status || result.Outcome != test.outcome {
				t.Fatalf("result = %+v, want status %d outcome %s", result, test.status, test.outcome)
			}
		})
	}
}

func TestBrowserTunnelCompletesAfterClientCancellation(t *testing.T) {
	transport := &recordingTransport{respond: func(request *http.Request) (*http.Response, error) {
		if err := request.Context().Err(); err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	}}
	tunnel := newTestTunnel(t, transport)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := tunnel.Forward(ctx, envelope(testBrowserDSN))
	if err != nil || result.Status != http.StatusOK {
		t.Fatalf("Forward() = %+v, %v; want delivery despite client cancellation", result, err)
	}
}
