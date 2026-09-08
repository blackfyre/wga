package artists

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestPrefersMarkdown(t *testing.T) {
	tests := []struct {
		accept string
		want   bool
	}{
		{accept: "text/markdown", want: true},
		{accept: "Text/Markdown; charset=utf-8", want: true},
		{accept: "text/markdown, text/html", want: true},
		{accept: "text/markdown;q=0.5, text/html;q=0.9", want: false},
		{accept: "text/markdown;q=0, text/html", want: false},
		{accept: "text/html,application/xhtml+xml,*/*", want: false},
		{accept: "*/*", want: false},
		{accept: "", want: false},
		{accept: "text/markdown;q=invalid", want: false},
	}
	for _, test := range tests {
		if got := prefersMarkdown(test.accept); got != test.want {
			t.Errorf("prefersMarkdown(%q) = %t, want %t", test.accept, got, test.want)
		}
	}
}

func TestMarkdownNegotiationPrecedesCatalogueWork(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pathKeys map[string]string
		invoke   func(*core.RequestEvent) error
		location string
	}{
		{
			name: "artist", path: "/artists/synthetic-artist-artistone000001",
			pathKeys: map[string]string{"name": "synthetic-artist-artistone000001"},
			invoke:   func(event *core.RequestEvent) error { return processArtist(event, nil) },
			location: "/agents/artists/artistone000001.md",
		},
		{
			name: "artwork", path: "/artists/synthetic-artist-artistone000001/blue-study-artworkone00001",
			pathKeys: map[string]string{"name": "synthetic-artist-artistone000001", "awid": "blue-study-artworkone00001"},
			invoke:   func(event *core.RequestEvent) error { return processArtwork(event, nil, config.EnvironmentTest) },
			location: "/agents/artworks/artworkone00001.md",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Accept", "text/markdown")
			for key, value := range test.pathKeys {
				request.SetPathValue(key, value)
			}
			recorder := httptest.NewRecorder()
			event := &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}
			if err := test.invoke(event); err != nil {
				t.Fatalf("negotiate Markdown: %v", err)
			}
			if recorder.Code != http.StatusTemporaryRedirect {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
			}
			if got := recorder.Header().Get("Location"); got != test.location {
				t.Errorf("Location = %q, want %q", got, test.location)
			}
			if !strings.Contains(recorder.Header().Get("Vary"), "Accept") {
				t.Errorf("Vary = %q, want Accept", recorder.Header().Get("Vary"))
			}
		})
	}
}
