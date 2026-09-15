package postcards

import (
	"html/template"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

// MessageLimit is the maximum number of visible Unicode characters in a
// postcard message.
const MessageLimit = 300

// Message is the canonical postcard message projection. HTML has passed the
// postcard-owned element-only allowlist; Text is the visible value used for
// required-field and character-limit validation.
type Message struct {
	html string
	text string
}

var postcardMessagePolicy = func() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements("p", "b", "i", "ul", "ol", "li")
	return policy
}()

// SanitiseMessage applies the postcard rich-text contract without allowing any
// element attributes. It is safe to call again when reading persisted content.
func SanitiseMessage(raw string) Message {
	html := strings.TrimSpace(postcardMessagePolicy.Sanitize(raw))
	text := strings.TrimSpace(bluemonday.StrictPolicy().Sanitize(html))
	return Message{html: html, text: text}
}

// HTML returns the sanitised markup for persistence or a separately guarded
// rendering boundary.
func (m Message) HTML() string {
	return m.html
}

// Text returns the visible plain-text value.
func (m Message) Text() string {
	return m.text
}

// RuneCount reports the visible Unicode character count rather than charging
// the visitor for formatting markup.
func (m Message) RuneCount() int {
	return utf8.RuneCountInString(m.text)
}

// TrustedHTML returns the already-sanitised markup for html/template email
// rendering. Callers cannot construct the trusted value without passing through
// SanitiseMessage.
func (m Message) TrustedHTML() template.HTML {
	return template.HTML(m.html) // #nosec G203 -- HTML is produced by the fixed element-only bluemonday policy above.
}
