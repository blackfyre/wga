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
		Work:    DualWorkRecord{ArtworkID: "aw0000000000001", Title: "Work", Sizes: []DualLink{}},
		SelfSel: "#dual-left", TargetSel: "#dual-left",
	}
	var output bytes.Buffer
	if err := dualWorkPane(window).Render(ctx, &output); err != nil {
		t.Fatalf("render dual work pane: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{`h-[46px]`, `px-[22px]`, "ADD TO AN ITINERARY +", `hx-select="unset"`} {
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
	for _, expected := range []string{`h-[50px]`, "ADD TO AN ITINERARY +", `hx-select="unset"`} {
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
