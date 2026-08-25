package requesttrust

import (
	"net/http"
	"net/http/httptest"
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
}
