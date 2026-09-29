package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHighlightMatchEscapesScriptLikeContent(t *testing.T) {
	var output bytes.Buffer
	component := HighlightMatch(`<script>alert("x")</script>`, "<script>")
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatalf("render highlight: %v", err)
	}
	rendered := output.String()

	if strings.Contains(rendered, "<script>") {
		t.Errorf("script-like text was not escaped: %q", rendered)
	}
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Errorf("expected escaped text content, got %q", rendered)
	}
	if !strings.Contains(rendered, "<mark>&lt;script&gt;</mark>") {
		t.Errorf("expected escaped match wrapped in mark, got %q", rendered)
	}
}

func TestSplitHighlightUnicodeSafe(t *testing.T) {
	// "İ" (U+0130) lowercases to a shorter byte sequence than the source rune,
	// so a byte index taken from the lowercased string must never be used to
	// slice the original. The old implementation sliced the original name by
	// such an index and produced invalid UTF-8 here.
	before, match, after := splitHighlight("İzmir", "zmir")
	if before != "İ" || match != "zmir" || after != "" {
		t.Errorf("splitHighlight = (%q, %q, %q), want (%q, %q, %q)", before, match, after, "İ", "zmir", "")
	}
	if !utf8.ValidString(before + match + after) {
		t.Errorf("splitHighlight output is not valid UTF-8: %q", before+match+after)
	}
}

func TestSplitHighlightCaseInsensitiveAndEscaped(t *testing.T) {
	before, match, after := splitHighlight("Rembrandt van Rijn", "VAN")
	if before != "Rembrandt " || match != "van" || after != " Rijn" {
		t.Errorf("splitHighlight = (%q, %q, %q), want case-insensitive (%q, %q, %q)", before, match, after, "Rembrandt ", "van", " Rijn")
	}
}

func TestSplitHighlightUsesUnicodeCaseFolding(t *testing.T) {
	// unicode.ToLower maps "Σ" to "σ" but leaves final sigma "ς" unchanged, so
	// only case folding treats them as equal.
	before, match, after := splitHighlight("ΟΔΥΣΣΕΥς", "Σ")
	if before != "ΟΔΥ" || match != "Σ" || after != "ΣΕΥς" {
		t.Errorf("splitHighlight = (%q, %q, %q), want first sigma matched", before, match, after)
	}
	before, match, after = splitHighlight("λόγος", "Σ")
	if before != "λόγο" || match != "ς" || after != "" {
		t.Errorf("splitHighlight = (%q, %q, %q), want final sigma matched", before, match, after)
	}
}
