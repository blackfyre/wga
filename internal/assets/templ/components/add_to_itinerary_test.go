package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
)

const addTestArtworkID = "aw0000000000001"

func renderAddAction(t *testing.T, ctx context.Context, variant AddToItineraryVariant) string {
	t.Helper()

	var output bytes.Buffer
	if err := AddToItineraryAction(addTestArtworkID, variant).Render(ctx, &output); err != nil {
		t.Fatalf("render typed add action: %v", err)
	}

	return output.String()
}

func renderAddButton(t *testing.T, ctx context.Context, variant string) string {
	t.Helper()

	var output bytes.Buffer
	if err := AddToItineraryButton(addTestArtworkID, variant).Render(ctx, &output); err != nil {
		t.Fatalf("render legacy add button: %v", err)
	}

	return output.String()
}

// TestAddToItineraryActionVariants table-tests the accepted reference
// presentation for every typed variant across the available, added, and full
// states: exact wrapper/button classes, the 46px row and 50px block
// dimensions, the compact short labels versus row/block full labels, and the
// shared disabled semantics.
func TestAddToItineraryActionVariants(t *testing.T) {
	base := "font-mono tracking-[1px] border transition-colors"

	cases := []struct {
		name      string
		variant   AddToItineraryVariant
		added     bool
		full      bool
		wrapper   string
		button    string
		label     string
		dimension string
	}{
		{
			name:    "compact available",
			variant: AddToItineraryCompact,
			wrapper: "shrink-0 self-center",
			button:  base + " text-(length:--t-10) px-2.5 py-1.5 border-primary text-primary hover:opacity-70",
			label:   "ADD",
		},
		{
			name:    "compact added",
			variant: AddToItineraryCompact,
			added:   true,
			wrapper: "shrink-0 self-center",
			button:  base + " text-(length:--t-10) px-2.5 py-1.5 border-base-content/20 bg-base-content/6 text-faint-2",
			label:   "ADDED ✓",
		},
		{
			name:    "compact full",
			variant: AddToItineraryCompact,
			full:    true,
			wrapper: "shrink-0 self-center",
			button:  base + " text-(length:--t-10) px-2.5 py-1.5 border-base-content/25 text-faint-2",
			label:   "ITINERARY IS FULL",
		},
		{
			name:      "row available",
			variant:   AddToItineraryRow,
			wrapper:   "shrink-0",
			button:    base + " text-(length:--t-11) tracking-[1.5px] px-[22px] h-[46px] border-control hover:border-primary hover:text-primary",
			label:     "ADD TO AN ITINERARY +",
			dimension: "h-[46px]",
		},
		{
			name:      "row added",
			variant:   AddToItineraryRow,
			added:     true,
			wrapper:   "shrink-0",
			button:    base + " text-(length:--t-11) tracking-[1.5px] px-[22px] h-[46px] border-base-content/20 bg-base-content/6 text-faint-2",
			label:     "IN YOUR ITINERARY ✓",
			dimension: "h-[46px]",
		},
		{
			name:      "row full",
			variant:   AddToItineraryRow,
			full:      true,
			wrapper:   "shrink-0",
			button:    base + " text-(length:--t-11) tracking-[1.5px] px-[22px] h-[46px] border-base-content/25 text-faint-2",
			label:     "ITINERARY IS FULL",
			dimension: "h-[46px]",
		},
		{
			name:      "block available",
			variant:   AddToItineraryBlock,
			wrapper:   "w-full",
			button:    base + " w-full text-xs tracking-[1.5px] px-6 h-[50px] border-control hover:border-primary hover:text-primary",
			label:     "ADD TO AN ITINERARY +",
			dimension: "h-[50px]",
		},
		{
			name:      "block added",
			variant:   AddToItineraryBlock,
			added:     true,
			wrapper:   "w-full",
			button:    base + " w-full text-xs tracking-[1.5px] px-6 h-[50px] border-base-content/20 bg-base-content/6 text-faint-2",
			label:     "IN YOUR ITINERARY ✓",
			dimension: "h-[50px]",
		},
		{
			name:      "block full",
			variant:   AddToItineraryBlock,
			full:      true,
			wrapper:   "w-full",
			button:    base + " w-full text-xs tracking-[1.5px] px-6 h-[50px] border-base-content/25 text-faint-2",
			label:     "ITINERARY IS FULL",
			dimension: "h-[50px]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			added := map[string]bool{}
			tray := dto.ItineraryTrayView{}
			if tc.added {
				added[addTestArtworkID] = true
			}
			if tc.full {
				tray.Count = itineraryworkflow.MaxStops
			}
			ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", tray, added)

			rendered := renderAddAction(t, ctx, tc.variant)

			for _, expected := range []string{
				`class="` + tc.wrapper + `"`,
				`class="` + tc.button + `"`,
				">" + tc.label + "</button>",
			} {
				if !strings.Contains(rendered, expected) {
					t.Errorf("variant %s does not contain %q\nrendered: %s", tc.variant, expected, rendered)
				}
			}

			if tc.added || tc.full {
				if !strings.Contains(rendered, "disabled") {
					t.Errorf("variant %s in added/full state must be disabled", tc.variant)
				}
			} else if strings.Contains(rendered, "disabled") {
				t.Errorf("available variant %s must not be disabled", tc.variant)
			}

			if tc.dimension == "" {
				if strings.Contains(rendered, "h-[46px]") || strings.Contains(rendered, "h-[50px]") {
					t.Errorf("compact variant %s must not carry fixed row/block heights", tc.variant)
				}
			} else if !strings.Contains(rendered, tc.dimension) {
				t.Errorf("variant %s missing dimension %q", tc.variant, tc.dimension)
			}
		})
	}
}

// TestAddToItineraryActionFormContract proves every typed variant renders the
// same ordinary POST fallback and HTMX contract with the unset sentinel and
// never opts out of inherited swapping.
func TestAddToItineraryActionFormContract(t *testing.T) {
	for _, variant := range []AddToItineraryVariant{AddToItineraryCompact, AddToItineraryRow, AddToItineraryBlock} {
		t.Run(string(variant), func(t *testing.T) {
			ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})
			rendered := renderAddAction(t, ctx, variant)

			for _, expected := range []string{
				`method="post"`,
				`action="/itineraries/draft/add"`,
				`hx-post="/itineraries/draft/add"`,
				`hx-target="#itinerary-tray"`,
				`hx-swap="outerHTML"`,
				`hx-select="unset"`,
				`name="artwork_id" value="` + addTestArtworkID + `"`,
				`name="_csrf" value="csrf-token"`,
			} {
				if !strings.Contains(rendered, expected) {
					t.Errorf("variant %s does not contain %q", variant, expected)
				}
			}

			if strings.Contains(rendered, "hx-disinherit") {
				t.Errorf("variant %s must not use hx-disinherit", variant)
			}
		})
	}
}

// TestAddToItineraryActionAnonymousRendersNothing proves a cookie-less request
// with no projected session renders an empty control rather than a broken form.
func TestAddToItineraryActionAnonymousRendersNothing(t *testing.T) {
	for _, variant := range []AddToItineraryVariant{AddToItineraryCompact, AddToItineraryRow, AddToItineraryBlock} {
		var output bytes.Buffer
		if err := AddToItineraryAction(addTestArtworkID, variant).Render(context.Background(), &output); err != nil {
			t.Fatalf("render anonymous typed add action: %v", err)
		}
		if output.String() != "" {
			t.Errorf("anonymous variant %s must render nothing, got %q", variant, output.String())
		}
	}
}

// TestAddToItineraryActionNormalisesInvalidVariant proves an unknown variant
// normalises to the compact presentation instead of emitting arbitrary markup.
func TestAddToItineraryActionNormalisesInvalidVariant(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})
	rendered := renderAddAction(t, ctx, AddToItineraryVariant("bogus"))

	for _, expected := range []string{
		`class="shrink-0 self-center"`,
		"text-(length:--t-10) px-2.5 py-1.5",
		">ADD</button>",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("invalid variant must normalise to compact, missing %q\nrendered: %s", expected, rendered)
		}
	}

	if strings.Contains(rendered, "h-[46px]") || strings.Contains(rendered, "h-[50px]") {
		t.Error("invalid variant must not render row/block dimensions")
	}
}

// TestAddToItineraryButtonLegacyPreservesOriginalRendering locks the legacy
// string-based control to its prior output so existing artwork/Dual/card call
// sites do not change before task 9.2 opts them into the typed component.
func TestAddToItineraryButtonLegacyPreservesOriginalRendering(t *testing.T) {
	base := "border font-mono text-(length:--t-10) tracking-[1px] transition-colors"

	cases := []struct {
		name         string
		variant      string
		added        bool
		button       string
		label        string
		disabled     bool
		ariaDisabled bool
	}{
		{
			name:    "compact available",
			variant: "compact",
			button:  base + " border-control px-3 py-1.5 text-base-content/70 hover:border-base-content",
			label:   "ADD TO ITINERARY",
		},
		{
			name:    "block available",
			variant: "block",
			button:  base + " border-control px-3 py-1.5 text-base-content/70 hover:border-base-content block w-full",
			label:   "ADD TO ITINERARY",
		},
		{
			name:         "compact added",
			variant:      "compact",
			added:        true,
			button:       base + " cursor-not-allowed border-base-content/20 px-3 py-1.5 text-base-content/40",
			label:        "ADDED",
			disabled:     true,
			ariaDisabled: true,
		},
		{
			name:         "block added",
			variant:      "block",
			added:        true,
			button:       base + " cursor-not-allowed border-base-content/20 px-3 py-1.5 text-base-content/40 block w-full",
			label:        "ADDED",
			disabled:     true,
			ariaDisabled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			added := map[string]bool{}
			if tc.added {
				added[addTestArtworkID] = true
			}
			ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, added)

			rendered := renderAddButton(t, ctx, tc.variant)

			for _, expected := range []string{
				`method="post"`,
				`action="/itineraries/draft/add"`,
				`class="inline"`,
				`hx-post="/itineraries/draft/add"`,
				`hx-target="#itinerary-tray"`,
				`hx-swap="outerHTML"`,
				`hx-select="unset"`,
				`name="artwork_id" value="` + addTestArtworkID + `"`,
				`name="_csrf" value="csrf-token"`,
				`class="` + tc.button + `"`,
				">" + tc.label + "</button>",
			} {
				if !strings.Contains(rendered, expected) {
					t.Errorf("legacy variant %s does not contain %q\nrendered: %s", tc.variant, expected, rendered)
				}
			}

			if tc.disabled {
				if !strings.Contains(rendered, "disabled") {
					t.Errorf("legacy added variant %s must be disabled", tc.variant)
				}
				if tc.ariaDisabled && !strings.Contains(rendered, `aria-disabled="true"`) {
					t.Errorf("legacy added variant %s must carry aria-disabled", tc.variant)
				}
			} else if strings.Contains(rendered, "disabled") {
				t.Errorf("legacy available variant %s must not be disabled", tc.variant)
			}

			if strings.Contains(rendered, "hx-disinherit") {
				t.Errorf("legacy variant %s must not use hx-disinherit", tc.variant)
			}
		})
	}
}

// TestAddToItineraryButtonLegacyRendersNothingWithoutSession proves the legacy
// control still renders nothing without a projected session.
func TestAddToItineraryButtonLegacyRendersNothingWithoutSession(t *testing.T) {
	var output bytes.Buffer
	if err := AddToItineraryButton(addTestArtworkID, "block").Render(context.Background(), &output); err != nil {
		t.Fatalf("render legacy add button: %v", err)
	}
	if output.String() != "" {
		t.Error("legacy add button must render nothing without a projected session")
	}
}
