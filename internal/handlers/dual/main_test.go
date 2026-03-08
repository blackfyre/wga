package dual

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestFormatArtistNameList(t *testing.T) {
	// Test case 1: Empty artist name list
	artistNameList := map[string]string{}
	expected := []dto.ArtistNameListEntry{}
	result := formatArtistNameList(artistNameList)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 2: Single entry in artist name list
	artistNameList = map[string]string{
		"/artist/1": "Artist One",
	}
	expected = []dto.ArtistNameListEntry{
		{Url: "/artist/1", Label: "Artist One"},
	}
	result = formatArtistNameList(artistNameList)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 3: Multiple entries in artist name list
	artistNameList = map[string]string{
		"/artist/1": "Artist One",
		"/artist/2": "Artist Two",
	}
	expected = []dto.ArtistNameListEntry{
		{Url: "/artist/1", Label: "Artist One"},
		{Url: "/artist/2", Label: "Artist Two"},
	}
	result = formatArtistNameList(artistNameList)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestReverseSide(t *testing.T) {
	// Test case 1: Input is "left"
	input := "left"
	expected := "right"
	result := reverseSide(input)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 2: Input is "right"
	input = "right"
	expected = "left"
	result = reverseSide(input)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 3: Input is neither "left" nor "right"
	input = "center"
	expected = ""
	result = reverseSide(input)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
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
			name:  "legacy broken artwork path",
			input: "/artists/example-123/artworks/artwork-777",
			want:  panePathDto{Kind: "artwork", Id: "777", RelPath: "/artists/example-123/artworks/artwork-777"},
		},
		{
			name:  "artworks route path",
			input: "/artworks/artwork-777",
			want:  panePathDto{Kind: "artwork", Id: "777", RelPath: "/artworks/artwork-777"},
		},
		{
			name:    "invalid path",
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
	left := renderPaneDto{Side: "left", RelPath: "/artists/example-123"}
	right := renderPaneDto{Side: "right", RelPath: "/artists/example-123/artwork-777"}

	pushURL := buildDualModePushURL(left, right)

	parsed, err := url.Parse(pushURL)
	if err != nil {
		t.Fatalf("expected valid push url, got error: %v", err)
	}

	if parsed.Path != "/dual-mode" {
		t.Fatalf("expected /dual-mode path, got %s", parsed.Path)
	}

	if got := parsed.Query().Get("left"); got != left.RelPath {
		t.Fatalf("expected left=%s, got %s", left.RelPath, got)
	}

	if got := parsed.Query().Get("right"); got != right.RelPath {
		t.Fatalf("expected right=%s, got %s", right.RelPath, got)
	}

	if got := parsed.Query().Get("left_render_to"); got != right.Side {
		t.Fatalf("expected left_render_to=%s, got %s", right.Side, got)
	}

	if got := parsed.Query().Get("right_render_to"); got != left.Side {
		t.Fatalf("expected right_render_to=%s, got %s", left.Side, got)
	}
}

func TestDefaultPaneContentLeft(t *testing.T) {
	content, err := defaultPaneContent("left")
	if err != nil {
		t.Fatalf("unexpected error while rendering default left pane content: %v", err)
	}

	if content == "" {
		t.Fatalf("expected non-empty default left pane content")
	}

	if !strings.Contains(content, "Choose content for comparison") {
		t.Fatalf("expected default left pane UI, got %q", content)
	}
}

func TestDefaultPaneContentUnsupported(t *testing.T) {
	if _, err := defaultPaneContent("center"); err == nil {
		t.Fatalf("expected unsupported pane type error")
	}
}
