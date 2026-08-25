package static

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestWithTableOfContentsAddsHeadingAnchors(t *testing.T) {
	content, toc, err := withTableOfContents("<p>Introduction</p><h2>Data we collect</h2><h3>Cookies</h3><h2>Data we collect</h2>")
	if err != nil {
		t.Fatalf("generate contents: %v", err)
	}

	if got, want := len(toc), 3; got != want {
		t.Fatalf("contents length = %d, want %d", got, want)
	}
	if got, want := toc[2].ID, "data-we-collect-2"; got != want {
		t.Errorf("duplicate heading ID = %q, want %q", got, want)
	}
	if !strings.Contains(content, `id="cookies"`) {
		t.Errorf("generated content does not contain heading anchors: %s", content)
	}
}

func TestWithTableOfContentsPreservesHeadingOrderAndLevels(t *testing.T) {
	content, toc, err := withTableOfContents(`<h2>First</h2><h3>Nested</h3><h2>Second</h2><h3>Nested two</h3><h2>First</h2>`)
	if err != nil {
		t.Fatalf("generate contents: %v", err)
	}

	want := []struct {
		title string
		level int
	}{
		{title: "First", level: 2},
		{title: "Nested", level: 3},
		{title: "Second", level: 2},
		{title: "Nested two", level: 3},
		{title: "First", level: 2},
	}

	if got := len(toc); got != len(want) {
		t.Fatalf("contents length = %d, want %d", got, len(want))
	}
	for index, item := range want {
		if toc[index].Title != item.title || toc[index].Level != item.level {
			t.Errorf("contents[%d] = %q (level %d), want %q (level %d)", index, toc[index].Title, toc[index].Level, item.title, item.level)
		}
	}
	if !strings.Contains(content, `id="first"`) || !strings.Contains(content, `id="nested"`) {
		t.Errorf("generated content does not preserve document-order anchors: %s", content)
	}
}

func TestWithTableOfContentsAssignsUniqueDeterministicIDs(t *testing.T) {
	content, toc, err := withTableOfContents(`<h2>Data</h2><h2>Data</h2><h2>Data</h2>`)
	if err != nil {
		t.Fatalf("generate contents: %v", err)
	}

	want := []string{"data", "data-2", "data-3"}
	if got := len(toc); got != len(want) {
		t.Fatalf("contents length = %d, want %d", got, len(want))
	}
	for index, id := range want {
		if toc[index].ID != id {
			t.Errorf("contents[%d].ID = %q, want %q", index, toc[index].ID, id)
		}
	}

	ids := collectIDs(t, content)
	for _, id := range want {
		if ids[id] != 1 {
			t.Errorf("id %q appears %d times, want exactly once", id, ids[id])
		}
	}
}

func TestWithTableOfContentsDeduplicatesExplicitIDs(t *testing.T) {
	content, toc, err := withTableOfContents(`<h2 id="terms">Terms</h2><h2 id="terms">Terms again</h2>`)
	if err != nil {
		t.Fatalf("generate contents: %v", err)
	}

	want := []string{"terms", "terms-2"}
	if got := len(toc); got != len(want) {
		t.Fatalf("contents length = %d, want %d", got, len(want))
	}
	for index, id := range want {
		if toc[index].ID != id {
			t.Errorf("contents[%d].ID = %q, want %q", index, toc[index].ID, id)
		}
	}

	ids := collectIDs(t, content)
	for _, id := range want {
		if ids[id] != 1 {
			t.Errorf("id %q appears %d times, want exactly once", id, ids[id])
		}
	}
}

func TestWithTableOfContentsDerivesIDsFromVisibleTextOnly(t *testing.T) {
	content, toc, err := withTableOfContents(`<h2>Hello <em>World</em> &amp; friends</h2>`)
	if err != nil {
		t.Fatalf("generate contents: %v", err)
	}

	if got, want := toc[0].Title, "Hello World & friends"; got != want {
		t.Errorf("visible title = %q, want %q", got, want)
	}
	if got, want := toc[0].ID, "hello-world--friends"; got != want {
		t.Errorf("derived ID = %q, want %q", got, want)
	}
	if !strings.Contains(content, `<em>World</em>`) {
		t.Errorf("managed markup must be preserved in content: %s", content)
	}
	if ids := collectIDs(t, content); ids[toc[0].ID] != 1 {
		t.Errorf("anchor %q must appear exactly once", toc[0].ID)
	}
}

func TestWithTableOfContentsFallsBackToNonEmptyIDs(t *testing.T) {
	content, toc, err := withTableOfContents(`<h2>你好</h2><h2>你好</h2><h2>###</h2>`)
	if err != nil {
		t.Fatalf("generate contents: %v", err)
	}

	if got := len(toc); got != 3 {
		t.Fatalf("contents length = %d, want 3", got)
	}
	for index, item := range toc {
		if item.ID == "" {
			t.Errorf("contents[%d].ID must not be empty", index)
		}
	}
	if toc[0].ID == toc[1].ID {
		t.Errorf("non-ASCII duplicate IDs must stay unique: %q", toc[0].ID)
	}

	ids := collectIDs(t, content)
	for _, item := range toc {
		if ids[item.ID] != 1 {
			t.Errorf("anchor %q appears %d times, want exactly once", item.ID, ids[item.ID])
		}
	}
}

func collectIDs(t *testing.T, content string) map[string]int {
	t.Helper()

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parse rendered content: %v", err)
	}

	ids := map[string]int{}
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attribute := range node.Attr {
				if attribute.Key == "id" {
					ids[attribute.Val]++
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	return ids
}
