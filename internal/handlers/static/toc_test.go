package static

import (
	"strings"
	"testing"
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
