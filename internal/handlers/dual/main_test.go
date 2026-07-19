package dual

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/pocketbase/pocketbase/core"
)

func TestFormatArtistNameList(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  []dto.ArtistNameListEntry
	}{
		{
			name:  "empty list",
			input: map[string]string{},
			want:  []dto.ArtistNameListEntry{},
		},
		{
			name: "sorts by label then URL",
			input: map[string]string{
				"/artists/two":     "Artist Two",
				"/artists/one":     "Artist One",
				"/artists/one-alt": "Artist One",
			},
			want: []dto.ArtistNameListEntry{
				{Url: "/artists/one", Label: "Artist One"},
				{Url: "/artists/one-alt", Label: "Artist One"},
				{Url: "/artists/two", Label: "Artist Two"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := formatArtistNameList(test.input)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %#v, got %#v", test.want, got)
			}
		})
	}
}

func TestReverseSide(t *testing.T) {
	tests := []struct {
		side string
		want string
	}{
		{side: "left", want: "right"},
		{side: "right", want: "left"},
		{side: "centre", want: ""},
	}

	for _, test := range tests {
		if got := reverseSide(test.side); got != test.want {
			t.Errorf("reverseSide(%q) = %q, want %q", test.side, got, test.want)
		}
	}
}

func TestResolvePaneTarget(t *testing.T) {
	tests := []struct {
		name      string
		side      string
		requested string
		want      string
	}{
		{name: "left targets itself", side: "left", requested: "left", want: "left"},
		{name: "left targets other pane", side: "left", requested: "right", want: "right"},
		{name: "right targets itself", side: "right", requested: "right", want: "right"},
		{name: "right targets other pane", side: "right", requested: "left", want: "left"},
		{name: "empty defaults to other pane", side: "left", requested: "", want: "right"},
		{name: "whitespace defaults to other pane", side: "right", requested: "  ", want: "left"},
		{name: "invalid target defaults to other pane", side: "left", requested: "centre", want: "right"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolvePaneTarget(test.side, test.requested); got != test.want {
				t.Fatalf("resolvePaneTarget(%q, %q) = %q, want %q", test.side, test.requested, got, test.want)
			}
		})
	}
}

func TestParsePanePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    panePathDto
		wantErr bool
	}{
		{
			name:  "default pane",
			input: "default",
			want:  panePathDto{Kind: "default", RelPath: "default"},
		},
		{
			name:  "artist canonical path",
			input: "/artists/example-123",
			want:  panePathDto{Kind: "artist", Id: "123", RelPath: "/artists/example-123"},
		},
		{
			name:  "artist artwork canonical path",
			input: "/artists/example-123/artwork-777",
			want:  panePathDto{Kind: "artwork", Id: "777", RelPath: "/artists/example-123/artwork-777"},
		},
		{
			name:  "legacy artwork path",
			input: "/artists/example-123/artworks/artwork-777",
			want:  panePathDto{Kind: "artwork", Id: "777", RelPath: "/artists/example-123/artworks/artwork-777"},
		},
		{
			name:  "artworks route path",
			input: "/artworks/artwork-777",
			want:  panePathDto{Kind: "artwork", Id: "777", RelPath: "/artworks/artwork-777"},
		},
		{
			name:    "unsupported path",
			input:   "/pages/privacy-policy",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePanePath(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", test.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", test.input, err)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %+v, got %+v", test.want, got)
			}
		})
	}
}

func TestBuildDualModePushURL(t *testing.T) {
	left := renderPaneDto{
		Side:     "left",
		RelPath:  "/artists/example-123",
		RenderTo: "left",
	}
	right := renderPaneDto{
		Side:     "right",
		RelPath:  "/artists/example-123/artwork-777",
		RenderTo: "right",
	}

	parsed := parseDualModeURL(t, buildDualModePushURL(left, right))
	assertDualModeQuery(t, parsed, map[string]string{
		"left":            left.RelPath,
		"right":           right.RelPath,
		"left_render_to":  "left",
		"right_render_to": "right",
	})
}

func TestBuildDualModeURLDefaultsEmptyValues(t *testing.T) {
	parsed := parseDualModeURL(t, buildDualModeURL("", "", "", "invalid"))
	assertDualModeQuery(t, parsed, map[string]string{
		"left":            "default",
		"right":           "default",
		"left_render_to":  "right",
		"right_render_to": "left",
	})
}

func TestBuildDualModeActionURL(t *testing.T) {
	left := renderPaneDto{RelPath: "/artists/left-123", RenderTo: "right"}
	right := renderPaneDto{RelPath: "/artworks/right-456", RenderTo: "left"}

	tests := []struct {
		action    string
		wantLeft  string
		wantRight string
	}{
		{action: "copy-left-to-right", wantLeft: left.RelPath, wantRight: left.RelPath},
		{action: "copy-right-to-left", wantLeft: right.RelPath, wantRight: right.RelPath},
		{action: "reverse", wantLeft: right.RelPath, wantRight: left.RelPath},
		{action: "clear-left", wantLeft: "default", wantRight: right.RelPath},
		{action: "clear-right", wantLeft: left.RelPath, wantRight: "default"},
	}

	for _, test := range tests {
		t.Run(test.action, func(t *testing.T) {
			parsed := parseDualModeURL(t, buildDualModeActionURL(left, right, test.action))
			assertDualModeQuery(t, parsed, map[string]string{
				"left":            test.wantLeft,
				"right":           test.wantRight,
				"left_render_to":  "right",
				"right_render_to": "left",
			})
		})
	}
}

func TestBuildDualPaneLoadForms(t *testing.T) {
	left := renderPaneDto{RelPath: "/artists/left-123", RenderTo: "right"}
	right := renderPaneDto{RelPath: "/artworks/right-456", RenderTo: "left"}

	got := buildDualPaneLoadForms(left, right)
	want := dto.DualPaneLoadFormsDto{
		Left: dto.DualPaneLoadFormDto{
			Path:          left.RelPath,
			OtherPath:     right.RelPath,
			LeftRenderTo:  "right",
			RightRenderTo: "left",
		},
		Right: dto.DualPaneLoadFormDto{
			Path:          right.RelPath,
			OtherPath:     left.RelPath,
			LeftRenderTo:  "right",
			RightRenderTo: "left",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildDualPaneLoadFormsBlankDefaultPath(t *testing.T) {
	left := renderPaneDto{RelPath: "default", RenderTo: "right"}
	right := renderPaneDto{RelPath: "/artworks/right-456", RenderTo: "left"}

	forms := buildDualPaneLoadForms(left, right)

	if forms.Left.Path != "" || forms.Right.OtherPath != "default" {
		t.Fatalf("expected blank default input and preserved path, got %#v", forms)
	}
}

func TestBuildDualPaneTargetURLs(t *testing.T) {
	left := renderPaneDto{RelPath: "/artists/left-123", RenderTo: "right"}
	right := renderPaneDto{RelPath: "/artworks/right-456", RenderTo: "left"}
	targets := buildDualPaneTargetURLs(left, right)

	tests := []struct {
		name string
		url  string
		want map[string]string
	}{
		{
			name: "left same pane",
			url:  targets.LeftSamePaneUrl,
			want: map[string]string{
				"left":            left.RelPath,
				"right":           right.RelPath,
				"left_render_to":  "left",
				"right_render_to": "left",
			},
		},
		{
			name: "left other pane",
			url:  targets.LeftOtherPaneUrl,
			want: map[string]string{
				"left":            left.RelPath,
				"right":           right.RelPath,
				"left_render_to":  "right",
				"right_render_to": "left",
			},
		},
		{
			name: "right same pane",
			url:  targets.RightSamePaneUrl,
			want: map[string]string{
				"left":            left.RelPath,
				"right":           right.RelPath,
				"left_render_to":  "right",
				"right_render_to": "right",
			},
		},
		{
			name: "right other pane",
			url:  targets.RightOtherPaneUrl,
			want: map[string]string{
				"left":            left.RelPath,
				"right":           right.RelPath,
				"left_render_to":  "right",
				"right_render_to": "left",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertDualModeQuery(t, parseDualModeURL(t, test.url), test.want)
		})
	}
}

func TestBuildDualModePaneURL(t *testing.T) {
	artistPath := "/artists/aagaard-carl-frederik-f2540d7a3fe99f9"
	artworkPath := "/artists/aagaard-carl-frederik-f2540d7a3fe99f9/deer-beside-a-lake-a6aab5e26c30056"

	tests := []struct {
		name        string
		side        string
		currentPath string
		destination string
		query       map[string][]string
		want        map[string]string
	}{
		{
			name:        "left source opens in other pane",
			side:        "left",
			currentPath: artistPath,
			destination: artworkPath,
			query: map[string][]string{
				"left":            {artistPath},
				"right":           {"default"},
				"left_render_to":  {"right"},
				"right_render_to": {"left"},
			},
			want: map[string]string{
				"left":            artistPath,
				"right":           artworkPath,
				"left_render_to":  "right",
				"right_render_to": "left",
			},
		},
		{
			name:        "left source opens in same pane",
			side:        "left",
			currentPath: artistPath,
			destination: artworkPath,
			query: map[string][]string{
				"left":            {artistPath},
				"right":           {"default"},
				"left_render_to":  {"left"},
				"right_render_to": {"left"},
			},
			want: map[string]string{
				"left":            artworkPath,
				"right":           "default",
				"left_render_to":  "left",
				"right_render_to": "left",
			},
		},
		{
			name:        "right source opens in other pane",
			side:        "right",
			currentPath: artworkPath,
			destination: artistPath,
			query: map[string][]string{
				"left":            {"default"},
				"right":           {artworkPath},
				"left_render_to":  {"right"},
				"right_render_to": {"left"},
			},
			want: map[string]string{
				"left":            artistPath,
				"right":           artworkPath,
				"left_render_to":  "right",
				"right_render_to": "left",
			},
		},
		{
			name:        "right source opens in same pane",
			side:        "right",
			currentPath: artworkPath,
			destination: artistPath,
			query: map[string][]string{
				"left":            {"default"},
				"right":           {artworkPath},
				"left_render_to":  {"right"},
				"right_render_to": {"right"},
			},
			want: map[string]string{
				"left":            "default",
				"right":           artistPath,
				"left_render_to":  "right",
				"right_render_to": "right",
			},
		},
		{
			name:        "invalid target defaults to other pane",
			side:        "left",
			currentPath: artistPath,
			destination: artworkPath,
			query: map[string][]string{
				"left":            {artistPath},
				"right":           {"default"},
				"left_render_to":  {"centre"},
				"right_render_to": {"right"},
			},
			want: map[string]string{
				"left":            artistPath,
				"right":           artworkPath,
				"left_render_to":  "right",
				"right_render_to": "right",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseDualModeURL(t, buildDualModePaneURL(test.side, test.currentPath, test.destination, test.query))
			assertDualModeQuery(t, got, test.want)
		})
	}
}

func TestResolvePaneRelPath(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		rendered  string
		want      string
	}{
		{
			name:      "uses rendered canonical path",
			requested: "/artists/requested-123",
			rendered:  "/artists/canonical-123",
			want:      "/artists/canonical-123",
		},
		{
			name:      "uses request when rendered path is empty",
			requested: "/artists/requested-123",
			want:      "/artists/requested-123",
		},
		{
			name:      "rejects nested dual mode URL",
			requested: "/artists/requested-123",
			rendered:  "/dual-mode?left=%2Fartists%2Frequested-123",
			want:      "/artists/requested-123",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolvePaneRelPath(test.requested, test.rendered); got != test.want {
				t.Fatalf("resolvePaneRelPath(%q, %q) = %q, want %q", test.requested, test.rendered, got, test.want)
			}
		})
	}
}

func TestRenderDefaultPane(t *testing.T) {
	pane, err := renderDefaultPane("left", "right")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pane.Side != "left" || pane.RelPath != "default" || pane.RenderTo != "right" {
		t.Fatalf("unexpected default pane metadata: %+v", pane)
	}

	if !strings.Contains(pane.Content, "Choose content for comparison") {
		t.Fatalf("expected default pane UI, got %q", pane.Content)
	}
}

func TestRenderPaneFallsBackForUnsupportedPath(t *testing.T) {
	event := &core.RequestEvent{}
	event.Request = httptest.NewRequest("GET", "/dual-mode?left=/pages/privacy-policy&left_render_to=centre", nil)

	pane, err := renderPane("left", nil, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pane.RelPath != "default" || pane.RenderTo != "right" {
		t.Fatalf("unexpected fallback pane metadata: %+v", pane)
	}

	if !strings.Contains(pane.Content, "Choose content for comparison") {
		t.Fatalf("expected default pane UI, got %q", pane.Content)
	}
}

func TestDefaultPaneContentUnsupported(t *testing.T) {
	if _, err := defaultPaneContent("centre"); err == nil {
		t.Fatal("expected unsupported pane type error")
	}
}

func parseDualModeURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("expected valid URL, got %q: %v", rawURL, err)
	}

	if parsed.Path != "/dual-mode" {
		t.Fatalf("expected /dual-mode path, got %q", parsed.Path)
	}

	return parsed
}

func assertDualModeQuery(t *testing.T, parsed *url.URL, want map[string]string) {
	t.Helper()

	for key, wantValue := range want {
		if got := parsed.Query().Get(key); got != wantValue {
			t.Errorf("expected %s=%q, got %q", key, wantValue, got)
		}
	}
}
