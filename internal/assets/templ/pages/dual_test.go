package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestDualWorkUsesTypedFullLabelRowAddControl(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, nil)
	window := DualWindow{
		Work:    DualWorkRecord{ArtworkID: "aw0000000000001", Title: "Work"},
		SelfSel: "#dual-left", TargetSel: "#dual-left",
	}
	var output bytes.Buffer
	if err := dualWorkPane(window).Render(ctx, &output); err != nil {
		t.Fatalf("render dual work pane: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{`h-12`, `px-[22px]`, "ADD TO AN ITINERARY +", `hx-select="unset"`, `data-study-board-add="aw0000000000001"`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("dual work pane missing typed row contract %q", expected)
		}
	}
}

func TestDualCardGridUsesTypedBlockAddControl(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, nil)
	var output bytes.Buffer
	if err := dualCardGrid([]DualCard{{ArtworkID: "aw0000000000001", Title: "Work", Href: "/work"}}, "#dual-left").Render(ctx, &output); err != nil {
		t.Fatalf("render dual card grid: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{`h-12`, "ADD TO AN ITINERARY +", `hx-select="unset"`, `data-study-board-add="aw0000000000001"`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("dual card grid missing typed block contract %q", expected)
		}
	}
}

func TestDualWorkPaneRendersEqualPaletteBands(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, nil)
	window := DualWindow{
		Work: DualWorkRecord{
			ArtworkID: "aw0000000000001",
			Title:     "Work",
			Palette: []dto.ColourSwatch{
				{Name: "Prussian Blue", Hex: "#1a2b3c", Weight: 5000},
				{Name: "Slate Blue", Hex: "#4d5e6f", Weight: 3000},
			},
		},
		SelfSel: "#dual-left", TargetSel: "#dual-left",
	}
	var output bytes.Buffer
	if err := dualWorkPane(window).Render(ctx, &output); err != nil {
		t.Fatalf("render dual work pane: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{"PALETTE", "Prussian Blue", "background:#1a2b3c;flex:1"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("dual palette missing %q", expected)
		}
	}
	if strings.Contains(rendered, "flex-grow:") {
		t.Error("dual palette bands must remain equal regardless of sampled weight")
	}
}

func TestDualWorkPaneShowsFixedPlateEvidence(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, nil)
	window := DualWindow{
		Work: DualWorkRecord{
			Title: "Work", Image: "/plate?thumb=1100x0", Zoom: "/plate?thumb=2000x0",
			SourceURL: "/plate.jpg", CurrentLocation: "Mauritshuis, The Hague",
		},
		SelfSel: "#dual-left", TargetSel: "#dual-left",
	}
	var output bytes.Buffer
	if err := dualWorkPane(window).Render(ctx, &output); err != nil {
		t.Fatalf("render dual work pane: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{"CLICK TO ZOOM", "THIS REPRODUCTION", "DOWNLOAD THE FULL FILE", "CURRENT LOCATION · Mauritshuis, The Hague", "ACCESSED"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("dual work pane missing %q", expected)
		}
	}
	for _, retired := range []string{"IMAGE SIZE", "r_size=", "l_size="} {
		if strings.Contains(rendered, retired) {
			t.Errorf("dual work pane retained %q", retired)
		}
	}
}

func TestDualSelectionLinksStayInTheirPane(t *testing.T) {
	window := DualWindow{
		Key: "left", SelfSel: "#dual-left", TargetSel: "#dual-right",
		Selection: DualSelectionRecord{
			ArtistFilingName: "Rijn, Rembrandt van", ArtistShortName: "Rembrandt",
			ArtistHref: "/dual-mode?left=artist", DisplayTitle: "Paintings", Context: "Selection lede.",
			Commentary: "<p>Complete commentary.</p>", HasCommentary: true, WorkCount: 1,
			HoldingNote: "1 selected from 10 catalogued works by Rembrandt.",
			Works:       []DualCard{{Title: "Work", Href: "/dual-mode?right=work"}},
			Siblings:    []DualLink{{Label: "Drawings", Href: "/dual-mode?left=sibling"}},
		},
	}
	var output bytes.Buffer
	if err := dualSelectionPane(window).Render(context.Background(), &output); err != nil {
		t.Fatalf("render dual selection pane: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{"21 — SELECTION", "Selection lede.", "Complete commentary.", "SELECTED WORKS", "OTHER SELECTIONS — Rembrandt", `href="/dual-mode?left=sibling" hx-get="/dual-mode?left=sibling" hx-target="#dual-left"`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("dual selection pane missing %q", expected)
		}
	}
}
