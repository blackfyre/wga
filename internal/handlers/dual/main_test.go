package dual

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestGetDualLookupResultsRequiresTwoRunes(t *testing.T) {
	content, err := getDualLookupResults(nil, "artwork", " é ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content.Kind != "artwork" || content.Query != "é" || !content.QueryTooShort {
		t.Fatalf("unexpected short lookup result: %#v", content)
	}
}

func TestGetArtistLookupResultsIsBounded(t *testing.T) {
	app := newDualLookupTestApp(t)

	for index := range dualLookupLimit + 1 {
		saveLookupArtist(t, app, fmt.Sprintf("Artist Lookup %02d", index), true)
	}
	saveLookupArtist(t, app, "Artist Lookup hidden", false)

	content, err := getDualLookupResults(app, "artist", "lookup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content.Results) != dualLookupLimit {
		t.Fatalf("expected %d results, got %d", dualLookupLimit, len(content.Results))
	}

	for index, result := range content.Results {
		wantLabel := fmt.Sprintf("Artist Lookup %02d", index)
		if result.Label != wantLabel {
			t.Fatalf("expected result %d to be %q, got %q", index, wantLabel, result.Label)
		}
		if !strings.HasPrefix(result.Url, "/artists/") {
			t.Fatalf("expected canonical artist URL, got %q", result.Url)
		}
	}
}

func TestGetArtworkLookupResultsIsBounded(t *testing.T) {
	app := newDualLookupTestApp(t)
	artist := saveLookupArtist(t, app, "Lookup Artist", true)

	for index := range dualLookupLimit + 1 {
		saveLookupArtwork(t, app, artist.Id, fmt.Sprintf("Artwork Lookup %02d", index), true)
	}
	saveLookupArtwork(t, app, artist.Id, "Artwork Lookup hidden", false)

	content, err := getDualLookupResults(app, "artwork", "lookup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content.Results) != dualLookupLimit {
		t.Fatalf("expected %d results, got %d", dualLookupLimit, len(content.Results))
	}

	for index, result := range content.Results {
		wantLabel := fmt.Sprintf("Artwork Lookup %02d", index)
		if result.Label != wantLabel {
			t.Fatalf("expected result %d to be %q, got %q", index, wantLabel, result.Label)
		}
		if result.Context != "Lookup Artist" {
			t.Fatalf("expected artist context, got %q", result.Context)
		}
		if !strings.HasPrefix(result.Url, "/artists/") {
			t.Fatalf("expected canonical artwork URL, got %q", result.Url)
		}
	}
}

func TestGetArtworkLookupResultsIncludesAuthorContext(t *testing.T) {
	app := newDualLookupTestApp(t)
	firstArtist := saveLookupArtist(t, app, "First Lookup Artist", true)
	secondArtist := saveLookupArtist(t, app, "Second Lookup Artist", true)
	saveLookupArtworkWithAuthors(
		t,
		app,
		[]string{firstArtist.Id, secondArtist.Id},
		"Multiple Lookup Authors",
		true,
	)

	content, err := getDualLookupResults(app, "artwork", "multiple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(content.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(content.Results))
	}

	if content.Results[0].Context != "First Lookup Artist" && content.Results[0].Context != "Second Lookup Artist" {
		t.Fatalf("expected associated author context, got %q", content.Results[0].Context)
	}
}

func TestDualLookupResultsRenderAccessibleStates(t *testing.T) {
	tests := []struct {
		name    string
		content dto.DualLookupResultsDto
		want    string
	}{
		{
			name:    "empty query",
			content: dto.DualLookupResultsDto{Kind: "artist"},
			want:    "Start typing to search artists.",
		},
		{
			name: "short query",
			content: dto.DualLookupResultsDto{
				Kind:          "artwork",
				Query:         "a",
				QueryTooShort: true,
			},
			want: "Enter at least two characters to search.",
		},
		{
			name:    "no results",
			content: dto.DualLookupResultsDto{Kind: "artist", Query: "Missing"},
			want:    "No artists match",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			if err := pages.DualLookupResults(test.content).Render(context.Background(), &buff); err != nil {
				t.Fatalf("unexpected render error: %v", err)
			}

			if !strings.Contains(buff.String(), test.want) {
				t.Fatalf("expected %q in %q", test.want, buff.String())
			}
		})
	}
}

func newDualLookupTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection(constants.CollectionArtists)
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("failed to create artists collection: %v", err)
	}

	artworks := core.NewBaseCollection(constants.CollectionArtworks)
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.BoolField{Name: "published"},
		&core.RelationField{
			Name:         "author",
			CollectionId: artists.Id,
			MaxSelect:    2,
		},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("failed to create artworks collection: %v", err)
	}

	return app
}

func saveLookupArtist(t *testing.T, app core.App, name string, published bool) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtists)
	if err != nil {
		t.Fatalf("failed to find artists collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save artist: %v", err)
	}

	return record
}

func saveLookupArtwork(t *testing.T, app core.App, artistID string, title string, published bool) *core.Record {
	return saveLookupArtworkWithAuthors(t, app, []string{artistID}, title, published)
}

func saveLookupArtworkWithAuthors(t *testing.T, app core.App, artistIDs []string, title string, published bool) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatalf("failed to find artworks collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("author", artistIDs)
	record.Set("published", published)
	record.Set("title", title)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save artwork: %v", err)
	}

	return record
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
