package static

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/utils"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func withTableOfContents(content string) (string, []pages.StaticPageTOCItem, error) {
	container := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(content), container)
	if err != nil {
		return "", nil, err
	}

	items := []pages.StaticPageTOCItem{}
	usedIDs := map[string]int{}
	for _, node := range nodes {
		collectHeadings(node, &items, usedIDs)
	}

	var rendered bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&rendered, node); err != nil {
			return "", nil, err
		}
	}

	return rendered.String(), items, nil
}

func collectHeadings(node *html.Node, items *[]pages.StaticPageTOCItem, usedIDs map[string]int) {
	if node.Type == html.ElementNode && (node.Data == "h2" || node.Data == "h3") {
		title := strings.TrimSpace(textContent(node))
		if title != "" {
			id := headingID(node, title, usedIDs)
			level := 2
			if node.Data == "h3" {
				level = 3
			}
			*items = append(*items, pages.StaticPageTOCItem{ID: id, Title: title, Level: level})
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectHeadings(child, items, usedIDs)
	}
}

func headingID(node *html.Node, title string, usedIDs map[string]int) string {
	base := ""
	idIndex := -1

	for index, attribute := range node.Attr {
		if attribute.Key != "id" {
			continue
		}

		idIndex = index
		if attribute.Val != "" {
			base = attribute.Val
		}
		break
	}

	if base == "" {
		base = utils.Slugify(title)
	}
	if base == "" {
		base = "section"
	}

	id := uniqueID(base, usedIDs)

	if idIndex >= 0 {
		node.Attr[idIndex].Val = id
	} else {
		node.Attr = append(node.Attr, html.Attribute{Key: "id", Val: id})
	}

	return id
}

// uniqueID returns base when it has not been used yet, otherwise the first
// base-N suffix that is still unused. It records every assignment so later
// headings resolve deterministically, including three or more duplicates.
func uniqueID(base string, usedIDs map[string]int) string {
	if usedIDs[base] == 0 {
		usedIDs[base] = 1
		return base
	}

	for number := 2; ; number++ {
		candidate := fmt.Sprintf("%s-%d", base, number)
		if usedIDs[candidate] == 0 {
			usedIDs[candidate] = 1
			return candidate
		}
	}
}

func textContent(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}

	var text strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(textContent(child))
	}

	return text.String()
}
