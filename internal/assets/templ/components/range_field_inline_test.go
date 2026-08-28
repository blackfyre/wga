package components

import (
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func inlineRangeField() dto.RangeField {
	return dto.RangeField{
		Label:     "YEAR RANGE",
		FromID:    "year_from",
		FromName:  "year_from",
		FromValue: 1500,
		ToID:      "year_to",
		ToName:    "year_to",
		ToValue:   1700,
		Min:       200,
		Max:       1900,
		Step:      10,
	}
}

func TestRangeFieldInlineOmitsDuplicateLegendAndOuterPadding(t *testing.T) {
	rendered := renderComponent(t, RangeField(inlineRangeFieldWithInline(true)))
	if strings.Contains(rendered, "<legend") {
		t.Fatal("inline range field must omit its inner legend")
	}
	if strings.Contains(rendered, "py-5") {
		t.Fatal("inline range field must omit its outer padding")
	}
	for _, expected := range []string{`id="year_from"`, `name="year_from"`, `value="1500"`, `id="year_to"`, `name="year_to"`, `value="1700"`, `aria-label="Range starts"`, `aria-label="Range ends"`, ">1500–1700</output>"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("inline range field missing %s", expected)
		}
	}
}

func TestRangeFieldDefaultKeepsLegendAndOuterPadding(t *testing.T) {
	rendered := renderComponent(t, RangeField(inlineRangeFieldWithInline(false)))
	if !strings.Contains(rendered, "<legend") || !strings.Contains(rendered, "YEAR RANGE</legend>") {
		t.Fatal("default range field must keep its inner legend")
	}
	if !strings.Contains(rendered, `class="py-5"`) {
		t.Fatal("default non-brush range field must keep its outer padding")
	}
}

func inlineRangeFieldWithInline(inline bool) dto.RangeField {
	field := inlineRangeField()
	field.Inline = inline
	return field
}
