package utils_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestContextFromRequestNilRequest(t *testing.T) {
	if got := templutils.RequestPath(templutils.ContextFromRequest(nil)); got != "" {
		t.Fatalf("expected empty request path, got %q", got)
	}
}

func TestContextFromRequestPreservesRequestContext(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), key, "value"))

	ctx := templutils.ContextFromRequest(request)
	if got := ctx.Value(key); got != "value" {
		t.Fatalf("expected preserved value %q, got %q", "value", got)
	}
	cancelled, cancel := context.WithCancel(request.Context())
	cancel()
	cancelledRequest := request.WithContext(cancelled)
	if err := templutils.ContextFromRequest(cancelledRequest).Err(); err != context.Canceled {
		t.Fatalf("expected cancelled context, got %v", err)
	}
}

func TestIsPathActiveChoosesTheLongestOwningDestination(t *testing.T) {
	destinations := []string{"/", "/artworks", "/artworks/search", "/itineraries", "/itineraries/new"}
	tests := []struct {
		name      string
		request   string
		candidate string
		want      bool
	}{
		{name: "root owns only root", request: "/", candidate: "/", want: true},
		{name: "root does not own another route", request: "/artworks", candidate: "/"},
		{name: "nested path selects longest destination", request: "/itineraries/new", candidate: "/itineraries/new", want: true},
		{name: "nested path does not select parent", request: "/itineraries/new", candidate: "/itineraries"},
		{name: "path boundary avoids false prefixes", request: "/artworks-search", candidate: "/artworks"},
		{name: "trailing slash remains owned", request: "/artworks/", candidate: "/artworks", want: true},
		{name: "artist record remains artists", request: "/artists/albrecht-durer-a1", candidate: "/artists", want: true},
		{name: "singular artist record remains artists", request: "/artist/albrecht-durer-a1", candidate: "/artists", want: true},
		{name: "artist artwork record belongs to artworks", request: "/artists/albrecht-durer-a1/melencolia-work1", candidate: "/artworks", want: true},
		{name: "singular artist artwork record belongs to artworks", request: "/artist/albrecht-durer-a1/melencolia-work1", candidate: "/artworks", want: true},
		{name: "artist artwork record does not select artists", request: "/artists/albrecht-durer-a1/melencolia-work1", candidate: "/artists"},
		{name: "artist selection remains artists", request: "/artists/albrecht-durer-a1/selections/selection1", candidate: "/artists", want: true},
		{name: "singular artist selection remains artists", request: "/artist/albrecht-durer-a1/selections/selection1", candidate: "/artists", want: true},
		{name: "artwork results remain artworks", request: "/artworks/results", candidate: "/artworks", want: true},
		{name: "singular artwork route remains outside artist aliases", request: "/artwork/melencolia-work1", candidate: "/artworks"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.request, nil)
			got := templutils.IsPathActive(templutils.ContextFromRequest(request), test.candidate, destinations)
			if got != test.want {
				t.Fatalf("IsPathActive(%q, %q) = %t, want %t", test.request, test.candidate, got, test.want)
			}
		})
	}
}
