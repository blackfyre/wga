package utils_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestContextFromRequestBionicReading(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
		want   bool
	}{
		{name: "absent cookie"},
		{name: "on", cookie: "on", want: true},
		{name: "off", cookie: "off"},
		{name: "unrecognised value", cookie: "enabled"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "wga_bionic", Value: test.cookie})
			}

			if got := templutils.GetBionicReading(templutils.ContextFromRequest(request)); got != test.want {
				t.Fatalf("expected bionic reading %t, got %t", test.want, got)
			}
		})
	}
}

func TestContextFromRequestNilRequest(t *testing.T) {
	if templutils.GetBionicReading(templutils.ContextFromRequest(nil)) {
		t.Fatal("expected bionic reading to be disabled")
	}
}

func TestContextFromRequestProjectsPreferences(t *testing.T) {
	tests := []struct {
		name    string
		cookies []*http.Cookie
		want    dto.Preferences
	}{
		{name: "defaults", want: dto.Preferences{Palette: dto.DefaultPaletteKey}},
		{
			name:    "palette only",
			cookies: []*http.Cookie{{Name: "wga_palette", Value: "verdigris"}},
			want:    dto.Preferences{Palette: "verdigris"},
		},
		{
			name:    "invalid palette falls back",
			cookies: []*http.Cookie{{Name: "wga_palette", Value: "neon"}},
			want:    dto.Preferences{Palette: dto.DefaultPaletteKey},
		},
		{
			name:    "scheme dark",
			cookies: []*http.Cookie{{Name: "wga_theme", Value: "dark"}},
			want:    dto.Preferences{Palette: dto.DefaultPaletteKey, Scheme: "dark"},
		},
		{
			name:    "legacy scheme value",
			cookies: []*http.Cookie{{Name: "wga_theme", Value: "wga_dark"}},
			want:    dto.Preferences{Palette: dto.DefaultPaletteKey, Scheme: "dark"},
		},
		{
			name:    "invalid scheme stays unset",
			cookies: []*http.Cookie{{Name: "wga_theme", Value: "sepia"}},
			want:    dto.Preferences{Palette: dto.DefaultPaletteKey},
		},
		{
			name:    "bionic on",
			cookies: []*http.Cookie{{Name: "wga_bionic", Value: "on"}},
			want:    dto.Preferences{Palette: dto.DefaultPaletteKey, Bionic: true},
		},
		{
			name: "combined",
			cookies: []*http.Cookie{
				{Name: "wga_palette", Value: "baroque"},
				{Name: "wga_theme", Value: "light"},
				{Name: "wga_bionic", Value: "on"},
			},
			want: dto.Preferences{Palette: "baroque", Scheme: "light", Bionic: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, cookie := range test.cookies {
				request.AddCookie(cookie)
			}

			got := templutils.GetPreferences(templutils.ContextFromRequest(request))
			if got != test.want {
				t.Fatalf("preferences = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestContextFromRequestPreservesRequestContext(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), key, "value"))
	request.AddCookie(&http.Cookie{Name: "wga_bionic", Value: "on"})

	ctx := templutils.ContextFromRequest(request)
	if got := ctx.Value(key); got != "value" {
		t.Fatalf("expected preserved value %q, got %q", "value", got)
	}
	if !templutils.GetBionicReading(ctx) {
		t.Fatal("expected bionic reading to be enabled")
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
