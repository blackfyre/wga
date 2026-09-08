package requesttrust

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDirectParsesAndCanonicalisesRemoteAddr(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		realIP     string
		railway    string
		want       string
		wantOK     bool
	}{
		{name: "host and port", remoteAddr: "198.51.100.7:54321", want: "198.51.100.7", wantOK: true},
		{name: "bare ipv4", remoteAddr: "198.51.100.7", want: "198.51.100.7", wantOK: true},
		{name: "ipv4-mapped ipv6 canonicalises", remoteAddr: "::ffff:198.51.100.7", want: "198.51.100.7", wantOK: true},
		{name: "ipv6 with port", remoteAddr: "[2001:db8::1]:8080", want: "2001:db8::1", wantOK: true},
		{name: "ignores forwarded headers", remoteAddr: "198.51.100.7:1", forwarded: "203.0.113.9, 10.0.0.1", realIP: "203.0.113.9", railway: "edge-pop", want: "198.51.100.7", wantOK: true},
		{name: "missing address", wantOK: false},
		{name: "invalid host", remoteAddr: "not-an-ip:80", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("X-Forwarded-For", test.forwarded)
			request.Header.Set("X-Real-IP", test.realIP)
			request.Header.Set("X-Railway-Edge", test.railway)

			got, ok := New(SourceDirect)(request)
			if ok != test.wantOK || got != test.want {
				t.Fatalf("resolve = %q, %t; want %q, %t", got, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestRailwayRequiresEdgeMarkerAndSingleRealIP(t *testing.T) {
	tests := []struct {
		name      string
		edge      string
		realIP    string
		forwarded string
		want      string
		wantOK    bool
	}{
		{name: "valid edge and single ip", edge: "us-west-1", realIP: "198.51.100.7", want: "198.51.100.7", wantOK: true},
		{name: "ignores forwarded", edge: "us-west-1", realIP: "198.51.100.7", forwarded: "203.0.113.9", want: "198.51.100.7", wantOK: true},
		{name: "canonicalises mapped ip", edge: "edge", realIP: "::ffff:198.51.100.7", want: "198.51.100.7", wantOK: true},
		{name: "missing edge", realIP: "198.51.100.7", wantOK: false},
		{name: "empty edge", edge: "  ", realIP: "198.51.100.7", wantOK: false},
		{name: "edge with comma", edge: "edge-a,edge-b", realIP: "198.51.100.7", wantOK: false},
		{name: "edge with whitespace", edge: "edge a", realIP: "198.51.100.7", wantOK: false},
		{name: "missing real ip", edge: "us-west-1", wantOK: false},
		{name: "invalid real ip", edge: "us-west-1", realIP: "not-an-ip", wantOK: false},
		{name: "real ip with port", edge: "us-west-1", realIP: "198.51.100.7:8080", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.Header.Set("X-Railway-Edge", test.edge)
			request.Header.Set("X-Real-IP", test.realIP)
			request.Header.Set("X-Forwarded-For", test.forwarded)

			got, ok := New(SourceRailway)(request)
			if ok != test.wantOK || got != test.want {
				t.Fatalf("resolve = %q, %t; want %q, %t", got, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestRailwayRejectsDuplicateHeaders(t *testing.T) {
	t.Run("duplicate edge markers", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Add("X-Railway-Edge", "edge-a")
		request.Header.Add("X-Railway-Edge", "edge-b")
		request.Header.Set("X-Real-IP", "198.51.100.7")

		if _, ok := New(SourceRailway)(request); ok {
			t.Fatal("duplicate edge markers must fail closed")
		}
	})

	t.Run("duplicate real ips", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-Railway-Edge", "edge-a")
		request.Header.Add("X-Real-IP", "198.51.100.7")
		request.Header.Add("X-Real-IP", "198.51.100.8")

		if _, ok := New(SourceRailway)(request); ok {
			t.Fatal("duplicate real-ip headers must fail closed")
		}
	})
}

func TestCloudflareRailwayAuthenticatesCurrentAndNextSecrets(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		clientIP string
		want     string
	}{
		{name: "current secret with ipv4", secret: "current-origin-secret", clientIP: "198.51.100.7", want: "198.51.100.7"},
		{name: "current secret with ipv6", secret: "current-origin-secret", clientIP: "2001:db8::1", want: "2001:db8::1"},
		{name: "next secret during rotation", secret: "next-origin-secret", clientIP: "::ffff:198.51.100.8", want: "198.51.100.8"},
	}

	resolver := New(SourceCloudflareRailway, NewCloudflareOriginSecrets("current-origin-secret", "next-origin-secret"))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("X-Railway-Edge", "edge-pop")
			request.Header.Set("X-WGA-Edge-Secret", test.secret)
			request.Header.Set("CF-Connecting-IP", test.clientIP)

			got, ok := resolver(request)
			if !ok || got != test.want {
				t.Fatalf("resolve = %q, %t; want %q, true", got, ok, test.want)
			}
		})
	}
}

func TestCloudflareRailwayRequiresConfiguredOriginAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Railway-Edge", "edge-pop")
	request.Header.Set("X-WGA-Edge-Secret", "visitor-supplied")
	request.Header.Set("CF-Connecting-IP", "198.51.100.7")

	if _, ok := New(SourceCloudflareRailway)(request); ok {
		t.Fatal("Cloudflare source without configured secrets must fail closed")
	}
	if _, ok := New(SourceCloudflareRailway, NewCloudflareOriginSecrets("configured-secret", ""))(request); ok {
		t.Fatal("unrecognised Cloudflare origin secret must fail closed")
	}
}

func TestCloudflareRailwayRejectsAdversarialProxyHeaders(t *testing.T) {
	tests := []struct {
		name       string
		edge       []string
		secrets    []string
		clientIPs  []string
		host       string
		realIP     string
		forwarded  string
		remoteAddr string
	}{
		{name: "missing edge", secrets: []string{"configured-secret"}, clientIPs: []string{"198.51.100.7"}},
		{name: "invalid edge", edge: []string{"edge a"}, secrets: []string{"configured-secret"}, clientIPs: []string{"198.51.100.7"}},
		{name: "duplicate edge", edge: []string{"edge-a", "edge-b"}, secrets: []string{"configured-secret"}, clientIPs: []string{"198.51.100.7"}},
		{name: "missing secret", edge: []string{"edge-a"}, clientIPs: []string{"198.51.100.7"}},
		{name: "invalid secret", edge: []string{"edge-a"}, secrets: []string{"visitor-secret"}, clientIPs: []string{"198.51.100.7"}},
		{name: "duplicate secret", edge: []string{"edge-a"}, secrets: []string{"configured-secret", "configured-secret"}, clientIPs: []string{"198.51.100.7"}},
		{name: "missing Cloudflare IP", edge: []string{"edge-a"}, secrets: []string{"configured-secret"}},
		{name: "duplicate Cloudflare IP", edge: []string{"edge-a"}, secrets: []string{"configured-secret"}, clientIPs: []string{"198.51.100.7", "198.51.100.8"}},
		{name: "malformed Cloudflare IP", edge: []string{"edge-a"}, secrets: []string{"configured-secret"}, clientIPs: []string{"not-an-ip"}},
		{name: "Cloudflare IP with port", edge: []string{"edge-a"}, secrets: []string{"configured-secret"}, clientIPs: []string{"198.51.100.7:443"}},
		{name: "comma-separated Cloudflare IP", edge: []string{"edge-a"}, secrets: []string{"configured-secret"}, clientIPs: []string{"198.51.100.7, 198.51.100.8"}},
		{name: "direct Railway request", edge: []string{"edge-a"}, realIP: "198.51.100.7", remoteAddr: "198.51.100.8:443"},
		{name: "spoofed public headers", edge: []string{"edge-a"}, host: "beta.wga.hu", realIP: "198.51.100.7", forwarded: "198.51.100.7", remoteAddr: "198.51.100.8:443"},
	}

	resolver := New(SourceCloudflareRailway, NewCloudflareOriginSecrets("configured-secret", "next-secret"))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://origin.example/", nil)
			request.Host = test.host
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("X-Real-IP", test.realIP)
			request.Header.Set("X-Forwarded-For", test.forwarded)
			for _, value := range test.edge {
				request.Header.Add("X-Railway-Edge", value)
			}
			for _, value := range test.secrets {
				request.Header.Add("X-WGA-Edge-Secret", value)
			}
			for _, value := range test.clientIPs {
				request.Header.Add("CF-Connecting-IP", value)
			}

			if identity, ok := resolver(request); ok {
				t.Fatalf("adversarial request resolved trusted identity %q", identity)
			}
		})
	}
}

func TestCloudflareRailwayIgnoresNonAuthoritativeIdentityHeaders(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://origin.example/", nil)
	request.Host = "spoofed.example"
	request.RemoteAddr = "192.0.2.10:443"
	request.Header.Set("X-Railway-Edge", "edge-a")
	request.Header.Set("X-WGA-Edge-Secret", "configured-secret")
	request.Header.Set("CF-Connecting-IP", "198.51.100.7")
	request.Header.Set("X-Real-IP", "203.0.113.8")
	request.Header.Set("X-Forwarded-For", "203.0.113.9")

	identity, ok := New(SourceCloudflareRailway, NewCloudflareOriginSecrets("configured-secret", ""))(request)
	if !ok || identity != "198.51.100.7" {
		t.Fatalf("resolve = %q, %t; want authoritative Cloudflare identity", identity, ok)
	}
}

func TestCloudflareOriginSecretsNeverFormatValues(t *testing.T) {
	const current = "current-secret-marker"
	const next = "next-secret-marker"
	secrets := NewCloudflareOriginSecrets(current, next)

	for _, formatted := range []string{fmt.Sprint(secrets), fmt.Sprintf("%+v", secrets), fmt.Sprintf("%#v", secrets)} {
		if strings.Contains(formatted, current) || strings.Contains(formatted, next) {
			t.Fatalf("formatted secrets exposed configured values: %q", formatted)
		}
	}
}

func TestUnknownSourceFallsBackToDirect(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.RemoteAddr = "198.51.100.7:1234"

	got, ok := New(Source("unknown"))(request)
	if !ok || got != "198.51.100.7" {
		t.Fatalf("unknown source = %q, %t; want direct resolution", got, ok)
	}
}

func TestNilRequestFailsClosed(t *testing.T) {
	if _, ok := New(SourceDirect)(nil); ok {
		t.Fatal("nil request must fail closed for direct")
	}
	if _, ok := New(SourceRailway)(nil); ok {
		t.Fatal("nil request must fail closed for railway")
	}
	if _, ok := New(SourceCloudflareRailway, NewCloudflareOriginSecrets("configured-secret", ""))(nil); ok {
		t.Fatal("nil request must fail closed for Cloudflare via Railway")
	}
}
