package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCitationBlockPreConstrainsAndScrollsBibTeX(t *testing.T) {
	var output bytes.Buffer
	citation := Citation{
		Key:   "wga-durer",
		Title: "Albrecht Dürer",
		URL:   "https://gallery.example/artists/durer",
	}
	if err := CitationBlock(citation).Render(context.Background(), &output); err != nil {
		t.Fatalf("render citation block: %v", err)
	}

	rendered := output.String()

	// The BibTeX pre must establish a definite border-box inline size with
	// horizontal scrolling, and the code must be a contained block, so its long
	// unwrapped line only contributes to the pre's own scrollWidth.
	if !strings.Contains(rendered, `style="display:block; width:100%; max-width:100%; box-sizing:border-box; overflow-x:auto; min-width:0"`) {
		t.Error("citation pre must carry the explicit border-box containment style")
	}
	if !strings.Contains(rendered, `<code style="display:block; max-width:100%"`) {
		t.Error("citation code must be a block constrained to the pre width")
	}
	if strings.Contains(rendered, "overflow-x-auto") || strings.Contains(rendered, "max-w-full") {
		t.Error("citation must not rely on the ineffective overflow-x-auto or max-w-full utilities")
	}

	// Copy-function compatibility: the copy button keeps targeting the pre id,
	// and the rendered BibTeX text is unchanged.
	if !strings.Contains(rendered, `id="bibtex-wga-durer"`) {
		t.Error("citation pre must keep its bibtex id for the copy target")
	}
	if !strings.Contains(rendered, `data-copy-target="#bibtex-wga-durer"`) {
		t.Error("copy button must keep targeting the bibtex pre")
	}
	for _, expected := range []string{
		"@online{wga-durer,",
		"author       = {Krén, Emil and Marx, Daniel},",
		"title        = {Albrecht Dürer},",
		"url          = {https://gallery.example/artists/durer},",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("citation block does not contain BibTeX %q", expected)
		}
	}
}
