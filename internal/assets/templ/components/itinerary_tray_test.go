package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestItineraryTrayRendersMountAndCount(t *testing.T) {
	tray := dto.ItineraryTrayView{
		Count:      3,
		BuilderURL: "/itineraries/new",
		Thumbs:     []string{"/api/files/artworks/aw0000000000001/img.jpg?thumb=120x0"},
	}

	var output bytes.Buffer
	if err := ItineraryTray(tray, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render tray: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`id="itinerary-tray"`,
		"ITINERARY DRAFT · 3 OF 15",
		`href="/itineraries/new"`,
		`?thumb=120x0`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("tray does not contain %q", expected)
		}
	}
	if strings.Contains(rendered, "hx-swap-oob") {
		t.Error("non-OOB tray must not carry hx-swap-oob")
	}
}

func TestItineraryTrayHidesBarWhenEmpty(t *testing.T) {
	var output bytes.Buffer
	if err := ItineraryTray(dto.ItineraryTrayView{}, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render tray: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `id="itinerary-tray"`) {
		t.Error("empty tray must still mount the target element")
	}
	if strings.Contains(rendered, "STOPS") {
		t.Error("empty tray must not render the bar content")
	}
}

func TestItineraryTrayOutOfBand(t *testing.T) {
	var output bytes.Buffer
	if err := ItineraryTray(dto.ItineraryTrayView{Count: 1, BuilderURL: "/itineraries/new"}, true).Render(context.Background(), &output); err != nil {
		t.Fatalf("render tray: %v", err)
	}

	if !strings.Contains(output.String(), `hx-swap-oob="true"`) {
		t.Error("OOB tray must carry hx-swap-oob")
	}
}

func TestItineraryTraySyncsDynamicReservationTargets(t *testing.T) {
	var output bytes.Buffer
	if err := ItineraryTray(dto.ItineraryTrayView{Count: 1, BuilderURL: "/itineraries/new"}, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render tray: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`const hasContent = this.children.length > 0`,
		`document.getElementById('mc-area')`,
		`mcArea.classList.toggle('pb-28', hasContent)`,
		`mcArea.classList.toggle('md:pb-20', hasContent)`,
		`document.getElementById('toast-container')`,
		`toastContainer.classList.toggle('bottom-28', hasContent)`,
		`toastContainer.classList.toggle('md:bottom-20', hasContent)`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("after-settle handler missing dynamic reservation sync %q", expected)
		}
	}
}

func TestAddToItineraryButtonRendersFormAndAddedState(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})
	var output bytes.Buffer
	if err := AddToItineraryButton("aw0000000000001", "compact").Render(ctx, &output); err != nil {
		t.Fatalf("render add button: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{
		`action="/itineraries/draft/add"`, `method="post"`, `hx-post="/itineraries/draft/add"`,
		`hx-target="#itinerary-tray"`, `hx-swap="outerHTML"`, `name="artwork_id" value="aw0000000000001"`,
		`name="_csrf" value="csrf-token"`, `hx-select="unset"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("add button does not contain %q", expected)
		}
	}
	if strings.Contains(rendered, `hx-disinherit="hx-select"`) {
		t.Error("add button must not use hx-disinherit")
	}

	addedCtx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{"aw0000000000001": true})
	var added bytes.Buffer
	if err := AddToItineraryButton("aw0000000000001", "compact").Render(addedCtx, &added); err != nil {
		t.Fatalf("render added button: %v", err)
	}
	if !strings.Contains(added.String(), "ADDED") || !strings.Contains(added.String(), "disabled") {
		t.Error("added button must show disabled ADDED state")
	}
}

func TestAddToItineraryButtonRendersNothingWithoutSession(t *testing.T) {
	var output bytes.Buffer
	if err := AddToItineraryButton("aw0000000000001", "compact").Render(context.Background(), &output); err != nil {
		t.Fatalf("render add button: %v", err)
	}
	if output.String() != "" {
		t.Error("add button must render nothing without a projected session")
	}
}

func TestItineraryTrayExactPresentationAndThumbnailOrder(t *testing.T) {
	tray := dto.ItineraryTrayView{
		Count: 2, BuilderURL: "/itineraries/new",
		Thumbs: []string{
			"/api/files/artworks/first/image.jpg?thumb=120x0",
			"/api/files/artworks/second/image.jpg?thumb=120x0",
		},
	}
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", tray, nil)
	var output bytes.Buffer
	if err := ItineraryTray(tray, false).Render(ctx, &output); err != nil {
		t.Fatalf("render tray: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{
		`class="fixed inset-x-0 bottom-0 z-[45] border-t border-neutral-content/20 bg-neutral animate-[wga-rise_240ms_cubic-bezier(0.22,0.61,0.36,1)]"`,
		`role="region" aria-label="Itinerary draft"`, `ITINERARY DRAFT · 2 OF 15`,
		`class="h-7 w-7 border border-neutral-content/25 bg-neutral-content/10 object-cover"`,
		`>CLEAR</button>`, `href="/itineraries/new"`, `>ARRANGE &amp; NARRATE →</a>`,
		`method="post" action="/itineraries/draft/clear"`,
		`hx-post="/itineraries/draft/clear" hx-target="#itinerary-tray" hx-swap="outerHTML" hx-select="unset"`,
		`name="_csrf" value="csrf-token"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("tray missing exact presentation %q", expected)
		}
	}
	first := strings.Index(rendered, `src="/api/files/artworks/first/image.jpg?thumb=120x0"`)
	second := strings.Index(rendered, `src="/api/files/artworks/second/image.jpg?thumb=120x0"`)
	if first < 0 || second < 0 || first > second {
		t.Fatalf("tray thumbnails are missing or out of order: first=%d second=%d", first, second)
	}
}

func TestItineraryTrayMapsReferenceShell(t *testing.T) {
	tray := dto.ItineraryTrayView{Count: 1, BuilderURL: "/itineraries/new"}
	var output bytes.Buffer
	if err := ItineraryTray(tray, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render tray: %v", err)
	}

	rendered := output.String()
	shell := `class="fixed inset-x-0 bottom-0 z-[45] border-t border-neutral-content/20 bg-neutral animate-[wga-rise_240ms_cubic-bezier(0.22,0.61,0.36,1)]"`
	if !strings.Contains(rendered, shell) {
		t.Errorf("tray shell missing reference mapping %q", shell)
	}
	if strings.Contains(rendered, "wga-enter") {
		t.Error("tray must not apply the 280ms wga-enter animation")
	}
	container := `class="container mx-auto flex items-center gap-4 px-5 py-3 md:gap-7 md:px-10 md:py-3.5"`
	if !strings.Contains(rendered, container) {
		t.Errorf("tray container missing reference padding %q", container)
	}
}
