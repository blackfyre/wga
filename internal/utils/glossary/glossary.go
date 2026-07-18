package glossary

import (
	"bytes"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var sanitizer = glossarySanitizer()

func glossarySanitizer() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements("a", "b", "blockquote", "br", "em", "i", "li", "ol", "p", "strong", "sub", "sup", "u", "ul")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowStandardURLs()

	return policy
}

const (
	glossaryCacheKey = "glossary:entries"
	glossaryTTL      = 6 * time.Hour
)

// GlossaryEntry represents a single glossary term with its matchable form and definition.
type GlossaryEntry struct {
	Expression string // full expression from DB
	MatchTerm  string // cleaned term used for matching
	Definition string // HTML definition
}

// CleanExpression extracts the matchable portion of a glossary expression
// by stripping parenthetical etymological notes.
// e.g. "acanthus (Lat. _acanthus_ ...)" -> "acanthus"
// e.g. "aerial perspective" -> "aerial perspective"
func CleanExpression(expr string) string {
	if idx := strings.Index(expr, "("); idx >= 0 {
		expr = expr[:idx]
	}
	return strings.TrimSpace(expr)
}

// GetGlossaryEntries fetches all glossary records from the database,
// cleans expressions, sorts longest-first, and caches for 6 hours.
func GetGlossaryEntries(app *pocketbase.PocketBase) ([]GlossaryEntry, error) {
	if cached, ok := utils.GetCachedValue[[]GlossaryEntry](app, glossaryCacheKey); ok {
		return cached, nil
	}

	records, err := app.FindRecordsByFilter(constants.CollectionGlossary, "", "+expression", 0, 0)
	if err != nil {
		return nil, err
	}

	entries := make([]GlossaryEntry, 0, len(records))
	for _, r := range records {
		expr := r.GetString("expression")
		matchTerm := CleanExpression(expr)
		if matchTerm == "" {
			continue
		}
		entries = append(entries, GlossaryEntry{
			Expression: expr,
			MatchTerm:  matchTerm,
			Definition: sanitizer.Sanitize(r.GetString("definition")),
		})
	}

	// Sort longest match term first so "altarpiece" matches before "altar"
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].MatchTerm) > len(entries[j].MatchTerm)
	})

	utils.SetCachedValue(app, glossaryCacheKey, entries, glossaryTTL)
	return entries, nil
}

// skipTags is the set of HTML elements whose text content should not be annotated.
var skipTags = map[atom.Atom]bool{
	atom.A:      true,
	atom.Script: true,
	atom.Style:  true,
	atom.Code:   true,
	atom.Pre:    true,
}

// AnnotateHTML scans HTML text for glossary terms and wraps the first occurrence
// of each term with a <span> containing the definition as a data attribute.
// It only modifies text nodes, never attribute values or content inside
// links, scripts, styles, or already-annotated spans.
func AnnotateHTML(htmlStr string, entries []GlossaryEntry) string {
	if htmlStr == "" || len(entries) == 0 {
		return htmlStr
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}

	matched := make(map[string]bool)

	var annotateNode func(n *html.Node)
	annotateNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Skip certain tags entirely
			if skipTags[n.DataAtom] {
				return
			}
			// Skip already-annotated glossary spans
			if n.DataAtom == atom.Span && hasClass(n, "glossary-term") {
				return
			}
		}

		// Process children (collect first since we may modify the tree)
		var children []*html.Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			children = append(children, c)
		}

		for _, child := range children {
			if child.Type == html.TextNode {
				annotateTextNode(child, entries, matched)
			} else {
				annotateNode(child)
			}
		}
	}

	annotateNode(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return htmlStr
	}

	// html.Parse wraps in <html><head><body>; extract just the body content
	return extractBodyContent(buf.String())
}

// annotateTextNode scans a text node from left to right and replaces the first
// occurrence of each unmatched term with annotated HTML. When two terms begin
// at the same position, the longer term takes precedence.
func annotateTextNode(textNode *html.Node, entries []GlossaryEntry, matched map[string]bool) {
	text := textNode.Data
	if strings.TrimSpace(text) == "" {
		return
	}

	parent := textNode.Parent
	cursor := 0

	for cursor < len(text) {
		matchStart := -1
		matchEnd := -1
		matchLength := 0
		var matchEntry *GlossaryEntry

		for i := range entries {
			entry := &entries[i]
			key := strings.ToLower(entry.MatchTerm)
			if matched[key] {
				continue
			}

			idx, end := indexWholeWordBounds(text[cursor:], entry.MatchTerm)
			if idx < 0 {
				continue
			}
			idx += cursor
			end += cursor
			termLength := utf8.RuneCountInString(entry.MatchTerm)

			if matchStart == -1 || idx < matchStart || (idx == matchStart && termLength > matchLength) {
				matchStart = idx
				matchEnd = end
				matchLength = termLength
				matchEntry = entry
			}
		}

		if matchEntry == nil {
			break
		}

		if matchStart > cursor {
			parent.InsertBefore(&html.Node{Type: html.TextNode, Data: text[cursor:matchStart]}, textNode)
		}

		termEnd := matchEnd
		span := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Span,
			Data:     "span",
			Attr: []html.Attribute{
				{Key: "class", Val: "glossary-term"},
				{Key: "role", Val: "button"},
				{Key: "tabindex", Val: "0"},
				{Key: "aria-expanded", Val: "false"},
				{Key: "aria-haspopup", Val: "dialog"},
			},
		}
		span.AppendChild(&html.Node{Type: html.TextNode, Data: text[matchStart:termEnd]})
		parent.InsertBefore(span, textNode)
		parent.InsertBefore(glossaryDefinitionTemplate(matchEntry.Definition), textNode)

		matched[strings.ToLower(matchEntry.MatchTerm)] = true
		cursor = termEnd
	}

	if cursor == 0 {
		return
	}
	if cursor < len(text) {
		parent.InsertBefore(&html.Node{Type: html.TextNode, Data: text[cursor:]}, textNode)
	}
	parent.RemoveChild(textNode)
}

func glossaryDefinitionTemplate(definition string) *html.Node {
	// Keep the sanitized markup in the server-rendered DOM so client code never
	// needs to parse a definition string.
	template := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Template,
		Data:     "template",
		Attr: []html.Attribute{
			{Key: "class", Val: "glossary-definition"},
		},
	}

	definitionNodes, err := html.ParseFragment(
		strings.NewReader(sanitizer.Sanitize(definition)),
		&html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"},
	)
	if err != nil {
		return template
	}

	for _, definitionNode := range definitionNodes {
		template.AppendChild(definitionNode)
	}

	return template
}

// indexWholeWord finds the first case-insensitive occurrence of term in text
// that is bounded by word boundaries (non-letter/digit characters or string edges).
func indexWholeWord(text, term string) int {
	idx, _ := indexWholeWordBounds(text, term)
	return idx
}

// indexWholeWordBounds returns the matching byte range in the original text.
func indexWholeWordBounds(text, term string) (int, int) {
	termRunes := utf8.RuneCountInString(term)
	if termRunes == 0 {
		return -1, -1
	}

	for start := 0; start < len(text); {
		end := start
		for range termRunes {
			if end >= len(text) {
				break
			}
			_, size := utf8.DecodeRuneInString(text[end:])
			end += size
		}

		if end > start && strings.EqualFold(text[start:end], term) {
			if start > 0 {
				r, _ := utf8.DecodeLastRuneInString(text[:start])
				if r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
					_, size := utf8.DecodeRuneInString(text[start:])
					start += size
					continue
				}
			}

			if end < len(text) {
				r, _ := utf8.DecodeRuneInString(text[end:])
				if r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
					_, size := utf8.DecodeRuneInString(text[start:])
					start += size
					continue
				}
			}

			return start, end
		}

		_, size := utf8.DecodeRuneInString(text[start:])
		start += size
	}

	return -1, -1
}

// hasClass checks if an HTML element node has a specific CSS class.
func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// extractBodyContent extracts the inner HTML of the <body> tag from a full HTML document.
func extractBodyContent(s string) string {
	bodyStart := strings.Index(s, "<body>")
	bodyEnd := strings.LastIndex(s, "</body>")
	if bodyStart < 0 || bodyEnd < 0 {
		return s
	}
	return s[bodyStart+len("<body>") : bodyEnd]
}
