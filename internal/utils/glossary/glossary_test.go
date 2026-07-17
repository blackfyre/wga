package glossary

import (
	"strings"
	"testing"
)

func TestCleanExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple word", "abacus", "abacus"},
		{"multi-word", "aerial perspective", "aerial perspective"},
		{"with parens and space", `acanthus (Lat. _acanthus_ Gk. _Akantha,_"thorn")`, "acanthus"},
		{"with parens no space", `alb(Lat. _alba tunica,_"white garment")`, "alb"},
		{"only parens", "(something)", ""},
		{"empty string", "", ""},
		{"trailing spaces", "  altarpiece  ", "altarpiece"},
		{"comma separated", "antependium, or altar frontal", "antependium, or altar frontal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanExpression(tt.input)
			if got != tt.expected {
				t.Errorf("CleanExpression(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAnnotateHTML(t *testing.T) {
	entries := []GlossaryEntry{
		{MatchTerm: "aerial perspective", Definition: "A way of suggesting far distance"},
		{MatchTerm: "perspective", Definition: "The art of drawing solid objects"},
		{MatchTerm: "altarpiece", Definition: "A picture behind an altar"},
		{MatchTerm: "fresco", Definition: "Wall painting on wet plaster"},
		{MatchTerm: "altar", Definition: "A table for religious ceremonies"},
	}

	tests := []struct {
		name     string
		html     string
		contains []string
		excludes []string
	}{
		{
			name:     "plain text with matching term",
			html:     "<p>The fresco was beautiful.</p>",
			contains: []string{`<span class="glossary-term"`, `data-glossary-def=`, ">fresco</span>"},
		},
		{
			name:     "term inside link is not annotated",
			html:     `<p>See the <a href="/fresco">fresco</a> page.</p>`,
			excludes: []string{`glossary-term`},
		},
		{
			name:     "only first occurrence annotated",
			html:     "<p>The fresco is a fresco technique.</p>",
			contains: []string{`<span class="glossary-term"`},
		},
		{
			name:     "case insensitive matching",
			html:     "<p>The Fresco was painted.</p>",
			contains: []string{`>Fresco</span>`},
		},
		{
			name:     "longer term matches before shorter",
			html:     "<p>The aerial perspective technique was used.</p>",
			contains: []string{`>aerial perspective</span>`},
			excludes: []string{`>perspective</span>`},
		},
		{
			name:     "longer term altarpiece matches before altar",
			html:     "<p>The altarpiece stood on the altar.</p>",
			contains: []string{`>altarpiece</span>`, `>altar</span>`},
		},
		{
			name:     "no matches returns unchanged",
			html:     "<p>Nothing special here.</p>",
			contains: []string{"<p>Nothing special here.</p>"},
			excludes: []string{`glossary-term`},
		},
		{
			name:     "empty input returns empty",
			html:     "",
			contains: []string{""},
		},
		{
			name:     "whole word boundary respected",
			html:     "<p>The frescoes were old.</p>",
			excludes: []string{`glossary-term`},
		},
		{
			name:     "term in attribute not annotated",
			html:     `<p><img alt="a fresco painting"> Nice art.</p>`,
			excludes: []string{`glossary-term`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnnotateHTML(tt.html, entries)

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("expected output to contain %q\ngot: %s", want, got)
				}
			}

			for _, notWant := range tt.excludes {
				if strings.Contains(got, notWant) {
					t.Errorf("expected output NOT to contain %q\ngot: %s", notWant, got)
				}
			}
		})
	}
}

func TestAnnotateHTML_FirstOccurrenceCount(t *testing.T) {
	entries := []GlossaryEntry{
		{MatchTerm: "fresco", Definition: "Wall painting"},
	}

	html := "<p>A fresco is a fresco is a fresco.</p>"
	got := AnnotateHTML(html, entries)

	count := strings.Count(got, `class="glossary-term"`)
	if count != 1 {
		t.Errorf("expected exactly 1 glossary-term span, got %d\noutput: %s", count, got)
	}
}

func TestAnnotateHTML_PrioritisesFirstPositionThenLongestTerm(t *testing.T) {
	entries := []GlossaryEntry{
		{MatchTerm: "fresco painting", Definition: "A painting made on wet plaster"},
		{MatchTerm: "fresco", Definition: "A painting made on wet plaster"},
	}

	got := AnnotateHTML("<p>A fresco precedes a fresco painting.</p>", entries)
	firstTerm := strings.Index(got, ">fresco</span>")
	laterTerm := strings.Index(got, ">fresco painting</span>")

	if firstTerm < 0 || laterTerm < 0 || firstTerm > laterTerm {
		t.Errorf("expected the earlier short term before the later longer term\ngot: %s", got)
	}
}

func TestAnnotateHTML_PrefersLongestTermAtSamePosition(t *testing.T) {
	entries := []GlossaryEntry{
		{MatchTerm: "fresco", Definition: "A painting made on wet plaster"},
		{MatchTerm: "fresco painting", Definition: "A painting made on wet plaster"},
	}

	got := AnnotateHTML("<p>A fresco painting.</p>", entries)

	if !strings.Contains(got, ">fresco painting</span>") {
		t.Errorf("expected the longer term to be annotated\ngot: %s", got)
	}
	if strings.Contains(got, ">fresco</span>") {
		t.Errorf("expected the shorter overlapping term to remain unannotated\ngot: %s", got)
	}
}

func TestAnnotateHTML_SanitizesDefinitionAndSetsButtonSemantics(t *testing.T) {
	entries := []GlossaryEntry{
		{MatchTerm: "fresco", Definition: `<img src=x onerror="alert(1)"><a href="javascript:alert(1)">unsafe</a><strong>Safe</strong>`},
	}

	got := AnnotateHTML("<p>A fresco.</p>", entries)

	for _, unsafe := range []string{"onerror=", "javascript:"} {
		if strings.Contains(got, unsafe) {
			t.Errorf("expected definition to exclude %q\ngot: %s", unsafe, got)
		}
	}
	for _, attribute := range []string{`role="button"`, `aria-expanded="false"`, `aria-haspopup="dialog"`} {
		if !strings.Contains(got, attribute) {
			t.Errorf("expected annotation to contain %q\ngot: %s", attribute, got)
		}
	}
}

func TestIndexWholeWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		term     string
		expected int
	}{
		{"simple match", "the fresco was nice", "fresco", 4},
		{"start of string", "fresco was nice", "fresco", 0},
		{"end of string", "nice fresco", "fresco", 5},
		{"no match partial", "the frescoes were old", "fresco", -1},
		{"no match prefix", "unfrescoed wall", "fresco", -1},
		{"multi-word", "the aerial perspective was used", "aerial perspective", 4},
		{"not found", "nothing here", "fresco", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexWholeWord(tt.text, tt.term)
			if got != tt.expected {
				t.Errorf("indexWholeWord(%q, %q) = %d, want %d", tt.text, tt.term, got, tt.expected)
			}
		})
	}
}
