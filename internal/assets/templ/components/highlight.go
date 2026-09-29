package components

import (
	"context"
	"io"
	"unicode"

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

	// Match case-insensitively over runes so a match found in a lowercased
	// string is sliced from the original by rune index. Lowercasing can change
	// a string's byte and rune length (for example "İ" folds to "i"+combining
	// dot), so byte indexes obtained from the folded string must never be used
	// to slice the original.
	textRunes := []rune(text)
	queryRunes := []rune(query)
	if len(queryRunes) == 0 || len(queryRunes) > len(textRunes) {
		return text, "", ""
	}

	for i := 0; i+len(queryRunes) <= len(textRunes); i++ {
		matched := true
		for j := 0; j < len(queryRunes); j++ {
			if unicode.ToLower(textRunes[i+j]) != unicode.ToLower(queryRunes[j]) {
				matched = false
				break
			}
		}
		if matched {
			return string(textRunes[:i]), string(textRunes[i : i+len(queryRunes)]), string(textRunes[i+len(queryRunes):])
		}
	}

	return text, "", ""
}
