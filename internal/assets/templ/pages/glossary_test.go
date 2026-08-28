package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestGlossaryBlockRendersSharedPageHeadOnce(t *testing.T) {
	view := GlossaryView{
		SelectedLetter: "",
		Letters:        []string{"A", "B"},
		Terms:          []GlossaryTerm{{Expression: "Chiaroscuro", Definition: "The treatment of light and shade."}},
	}
	var output bytes.Buffer
	if err := GlossaryBlock(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render glossary block: %v", err)
	}
	rendered := output.String()

	for _, expected := range []string{
		"13 — GLOSSARY",
		">Glossary</h1>",
		"Terms used in the commentaries and catalogue entries.",
		"text-(length:--t-11)",
		"text-muted",
		"text-(length:--t-32)",
		"md:text-(length:--t-44)",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("glossary page head does not contain %q", expected)
		}
	}
	if got := strings.Count(rendered, "<h1"); got != 1 {
		t.Errorf("h1 count = %d, want 1", got)
	}
	for _, expected := range []string{
		`aria-label="Filter terms by letter"`,
		`hx-get="/glossary"`,
		`hx-target="#glossary"`,
		`hx-select="#glossary"`,
		"Chiaroscuro",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("glossary filters do not contain %q", expected)
		}
	}
}
