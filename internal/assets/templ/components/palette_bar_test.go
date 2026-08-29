package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestPaletteBarRendersSourceNamesAndWeightedBands(t *testing.T) {
	var output bytes.Buffer
	bands := []dto.ColourSwatch{
		{Name: "Prussian Blue", Hex: "#1a2b3c", Weight: 5000},
		{Name: "Slate Blue", Hex: "#4d5e6f", Weight: 3000},
	}
	if err := PaletteBar(bands, true, "SOURCE NOTE").Render(context.Background(), &output); err != nil {
		t.Fatalf("render palette bar: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		"Prussian Blue, #1a2b3c, 63% of the surface",
		"Slate Blue, #4d5e6f, 38% of the surface",
		"background:#1a2b3c;flex-grow:5000;flex-basis:0",
		"w-[220px]",
		"flex-wrap",
		"SOURCE NOTE",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("palette bar missing %q", expected)
		}
	}
	if strings.Contains(rendered, "max-w-full") {
		t.Error("palette tooltip must not be constrained by its swatch width")
	}
}

func TestPaletteBarUsesEqualBandsForComparison(t *testing.T) {
	var output bytes.Buffer
	bands := []dto.ColourSwatch{
		{Name: "Prussian Blue", Hex: "#1a2b3c", Weight: 5000},
		{Name: "Slate Blue", Hex: "#4d5e6f", Weight: 3000},
	}
	if err := PaletteBar(bands, false, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render palette bar: %v", err)
	}

	if strings.Contains(output.String(), "flex-grow:") || !strings.Contains(output.String(), "background:#1a2b3c;flex:1") {
		t.Error("comparison palette must use equal bands")
	}
}

func TestPaletteBarOmitsUnavailableColourName(t *testing.T) {
	var output bytes.Buffer
	bands := []dto.ColourSwatch{{Hex: "#1a2b3c", Weight: 5000}}
	if err := PaletteBar(bands, true, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render palette bar: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `aria-label="#1a2b3c, 100% of the surface"`) {
		t.Errorf("unnamed palette bar has incorrect label: %s", rendered)
	}
	if strings.Contains(rendered, `<span></span>`) || strings.Contains(rendered, ", #1a2b3c") {
		t.Errorf("unnamed palette bar must omit the unavailable name: %s", rendered)
	}
}
