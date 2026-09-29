package components

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// HighlightMatch renders text with the first case-insensitive occurrence of
// query wrapped in a mark element. All dynamic content is escaped; the markup
// itself is static, so script-like text and search strings cannot inject
// markup.
func HighlightMatch(text string, query string) templ.Component {
	before, match, after := splitHighlight(text, query)
	escaped := templ.EscapeString(before)
	if match != "" {
		escaped += "<mark>" + templ.EscapeString(match) + "</mark>" + templ.EscapeString(after)
	}
	content := escaped
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, content)
		return err
	})
}

func splitHighlight(text string, query string) (string, string, string) {
	if query == "" {
		return text, "", ""
	}

	// Compare rune windows of the original text with Unicode case folding and
	// slice the original by rune index. Lowercasing can change a string's byte
	// and rune length (for example "İ" folds to "i"+combining dot), so byte
	// indexes obtained from a folded string must never be used to slice the
	// original. EqualFold also equates case variants that ToLower does not,
	// such as Greek "Σ" and final "ς".
	textRunes := []rune(text)
	queryRunes := []rune(query)
	if len(queryRunes) == 0 || len(queryRunes) > len(textRunes) {
		return text, "", ""
	}

	for i := 0; i+len(queryRunes) <= len(textRunes); i++ {
		if strings.EqualFold(string(textRunes[i:i+len(queryRunes)]), query) {
			return string(textRunes[:i]), string(textRunes[i : i+len(queryRunes)]), string(textRunes[i+len(queryRunes):])
		}
	}

	return text, "", ""
}
