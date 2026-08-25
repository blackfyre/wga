package itineraries

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// textSanitizer strips all markup from visitor-authored plain-text fields.
// It is deliberately stricter than the record biography policy because
// itinerary title, intro, creator, and narration are prose, not rich content.
var textSanitizer = bluemonday.StrictPolicy()

// SanitiseText returns value with all markup removed and surrounding
// whitespace trimmed. It is applied before persistence so the stored data is
// always safe plain text.
func SanitiseText(value string) string {
	return strings.TrimSpace(textSanitizer.Sanitize(value))
}
