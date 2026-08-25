package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestGlossaryTermRendersAccessiblePlainTextTooltip(t *testing.T) {
	var output bytes.Buffer
	if err := GlossaryTerm("Triptych", "OBJECT TYPE", "A three-panel work <em>without markup</em>.").Render(context.Background(), &output); err != nil {
		t.Fatalf("render glossary term: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`<dfn`,
		`class="wga-term"`,
		`role="note"`,
		`tabindex="0"`,
		`aria-label="Triptych: A three-panel work &lt;em&gt;without markup&lt;/em&gt;."`,
		`data-bionic="off"`,
		`Triptych`,
		`class="wga-term__tooltip" aria-hidden="true"`,
		`>GLOSSARY</span>`,
		`class="wga-tooltip__category">OBJECT TYPE</span>`,
		`class="wga-tooltip__body">A three-panel work &lt;em&gt;without markup&lt;/em&gt;.</span>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected glossary contract %q in %s", expected, rendered)
		}
	}
	for _, forbidden := range []string{"href=", "<button", "<em>", `role="button"`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("glossary term must not contain %q", forbidden)
		}
	}
}

func TestGlossaryTermOmitsEmptyCategory(t *testing.T) {
	var output bytes.Buffer
	if err := GlossaryTerm("Fresco", "", "Painting on wet plaster.").Render(context.Background(), &output); err != nil {
		t.Fatalf("render glossary term: %v", err)
	}

	if strings.Contains(output.String(), "wga-tooltip__category") {
		t.Fatal("expected empty glossary category to be omitted")
	}
}

func TestHelpTipRendersAccessiblePlainTextTooltip(t *testing.T) {
	var output bytes.Buffer
	if err := HelpTip("Selection", "Choose up to three works <strong>only</strong>.").Render(context.Background(), &output); err != nil {
		t.Fatalf("render help tip: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`class="wga-help"`,
		`role="note"`,
		`tabindex="0"`,
		`aria-label="Selection: Choose up to three works &lt;strong&gt;only&lt;/strong&gt;."`,
		`data-bionic="off"`,
		`?`,
		`class="wga-help__tooltip" aria-hidden="true"`,
		`class="wga-tooltip__meta">Selection</span>`,
		`class="wga-tooltip__body">Choose up to three works &lt;strong&gt;only&lt;/strong&gt;.</span>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected help contract %q in %s", expected, rendered)
		}
	}
	for _, forbidden := range []string{"href=", "<button", "<strong>", `role="button"`} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("help tip must not contain %q", forbidden)
		}
	}
}

func TestHelpTipEscapesScriptMarkupInAccessibleAndVisibleOutput(t *testing.T) {
	var output bytes.Buffer
	if err := HelpTip("Selection", "<script>alert(1)</script>").Render(context.Background(), &output); err != nil {
		t.Fatalf("render help tip: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`aria-label="Selection: &lt;script&gt;alert(1)&lt;/script&gt;"`,
		`class="wga-tooltip__body">&lt;script&gt;alert(1)&lt;/script&gt;</span>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected escaped script contract %q in %s", expected, rendered)
		}
	}
	if strings.Contains(rendered, "<script>") {
		t.Fatal("script markup must not be rendered")
	}
}
