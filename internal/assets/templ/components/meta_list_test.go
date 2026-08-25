package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMetaListConstrainsLongValueAndFixesLabel(t *testing.T) {
	var output bytes.Buffer
	entries := []MetaEntry{
		{Label: "SOURCE", Value: "in/art/s/stazio/sagredo2.jpg"},
		{Label: "MEDIUM", Value: "Oil on canvas"},
	}
	if err := MetaList(entries).Render(context.Background(), &output); err != nil {
		t.Fatalf("render meta list: %v", err)
	}

	rendered := output.String()

	// The label is fixed so it never shrinks or clips while the value wraps.
	if !strings.Contains(rendered, `style="flex-shrink:0"`) {
		t.Error("meta label must carry a fixed flex-shrink:0 role")
	}
	// The value wraps long unbroken provenance paths within the container.
	if !strings.Contains(rendered, `style="min-width:0; max-width:100%; overflow-wrap:anywhere"`) {
		t.Error("meta value must carry min-width/max-width/overflow-wrap containment")
	}
	// Data values and labels are preserved verbatim (no alteration).
	for _, expected := range []string{
		"SOURCE",
		"in/art/s/stazio/sagredo2.jpg",
		"MEDIUM",
		"Oil on canvas",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("meta list does not contain %q", expected)
		}
	}
}
