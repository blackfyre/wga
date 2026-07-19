package artworks

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/pocketbase/pocketbase/core"
)

func TestGetDualModeSearchContext(t *testing.T) {
	event := &core.RequestEvent{}
	event.Request = httptest.NewRequest(
		"GET",
		"/artworks?dual_left=/artists/left-123&dual_right=/artworks/right-456&dual_left_render_to=left&dual_right_render_to=right&dual_target=right",
		nil,
	)

	got := getDualModeSearchContext(event)
	if got == nil {
		t.Fatal("expected Dual Mode search context")
	}

	want := &dto.ArtworkSearchDualModeDto{
		LeftPath:      "/artists/left-123",
		RightPath:     "/artworks/right-456",
		LeftRenderTo:  "left",
		RightRenderTo: "right",
		Target:        "right",
	}

	if *got != *want {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestGetDualModeSearchContextRequiresTarget(t *testing.T) {
	event := &core.RequestEvent{}
	event.Request = httptest.NewRequest("GET", "/artworks?dual_left=/artists/left-123", nil)

	if got := getDualModeSearchContext(event); got != nil {
		t.Fatalf("expected no Dual Mode search context, got %#v", got)
	}
}

func TestBuildArtworkSearchPathPreservesDualModeContext(t *testing.T) {
	dualModeContext := &dto.ArtworkSearchDualModeDto{
		LeftPath:      "/artists/left-123",
		RightPath:     "/artworks/right-456",
		LeftRenderTo:  "left",
		RightRenderTo: "right",
		Target:        "right",
	}

	searchURL, err := url.Parse(buildArtworkSearchPath("/artworks/results", &filters{
		Title: "A Couple",
		Page:  "2",
	}, dualModeContext))
	if err != nil {
		t.Fatalf("expected valid search URL: %v", err)
	}

	if searchURL.Path != "/artworks/results" {
		t.Fatalf("expected /artworks/results path, got %q", searchURL.Path)
	}

	for key, want := range map[string]string{
		"title":                "A Couple",
		"page":                 "2",
		"dual_left":            dualModeContext.LeftPath,
		"dual_right":           dualModeContext.RightPath,
		"dual_left_render_to":  dualModeContext.LeftRenderTo,
		"dual_right_render_to": dualModeContext.RightRenderTo,
		"dual_target":          dualModeContext.Target,
	} {
		if got := searchURL.Query().Get(key); got != want {
			t.Errorf("expected %s=%q, got %q", key, want, got)
		}
	}
}

func TestBuildDualModeArtworkURL(t *testing.T) {
	dualModeContext := &dto.ArtworkSearchDualModeDto{
		LeftPath:      "/artists/left-123",
		RightPath:     "/artworks/right-456",
		LeftRenderTo:  "left",
		RightRenderTo: "right",
		Target:        "right",
	}

	dualModeURL, err := url.Parse(buildDualModeArtworkURL("/artworks/selected-789", dualModeContext))
	if err != nil {
		t.Fatalf("expected valid Dual Mode URL: %v", err)
	}

	if dualModeURL.Path != "/dual-mode" {
		t.Fatalf("expected /dual-mode path, got %q", dualModeURL.Path)
	}

	for key, want := range map[string]string{
		"left":            dualModeContext.LeftPath,
		"right":           "/artworks/selected-789",
		"left_render_to":  dualModeContext.LeftRenderTo,
		"right_render_to": dualModeContext.RightRenderTo,
	} {
		if got := dualModeURL.Query().Get(key); got != want {
			t.Errorf("expected %s=%q, got %q", key, want, got)
		}
	}
}
