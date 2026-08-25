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
		"3 STOPS",
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

func TestAddToItineraryButtonRendersFormAndAddedState(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})

	var output bytes.Buffer
	if err := AddToItineraryButton("aw0000000000001", "compact").Render(ctx, &output); err != nil {
		t.Fatalf("render add button: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`action="/itineraries/draft/add"`,
		`method="post"`,
		`hx-post="/itineraries/draft/add"`,
		`hx-target="#itinerary-tray"`,
		`hx-swap="outerHTML"`,
		`name="artwork_id" value="aw0000000000001"`,
		`name="_csrf" value="csrf-token"`,
		`hx-select="unset"`,
		"ADD TO ITINERARY",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("add button does not contain %q", expected)
		}
	}
	if strings.Contains(rendered, `hx-disinherit="hx-select"`) {
		t.Error("add button must not use hx-disinherit; the unset sentinel preserves the inherited tray/OOB swap")
	}

	addedCtx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{"aw0000000000001": true})
	var added bytes.Buffer
	if err := AddToItineraryButton("aw0000000000001", "compact").Render(addedCtx, &added); err != nil {
		t.Fatalf("render added button: %v", err)
	}
	if !strings.Contains(added.String(), "ADDED") {
		t.Error("added button must show the ADDED state")
	}
	if !strings.Contains(added.String(), "disabled") {
		t.Error("added button must be disabled")
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
