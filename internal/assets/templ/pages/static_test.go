package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func renderStatic(t *testing.T, content StaticPageDTO) string {
	t.Helper()

	var output bytes.Buffer
	if err := StaticPageBlock(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render static page block: %v", err)
	}

	return output.String()
}

func TestStaticPageRendersSingleResponsiveHeadingWithRouteKickers(t *testing.T) {
	cases := []struct {
		name   string
		page   StaticPageDTO
		kicker string
		title  string
	}{
		{
			name:   "about",
			page:   StaticPageDTO{Title: "About", Url: "/pages/about", Content: "<h2 id=\"the-collection\">The collection</h2><p>Body.</p>", TOC: []StaticPageTOCItem{{ID: "the-collection", Title: "The collection", Level: 2}}},
			kicker: "14 — ABOUT THE COLLECTION",
			title:  "About",
		},
		{
			name:   "privacy",
			page:   StaticPageDTO{Title: "Privacy policy", Url: "/pages/privacy-policy", Content: "<p>Body.</p>"},
			kicker: "06 — GENERAL CONTENT",
			title:  "Privacy policy",
		},
		{
			name:   "generic",
			page:   StaticPageDTO{Title: "Licences", Url: "/pages/open-source-licences", Content: "<p>Body.</p>"},
			kicker: "PUBLIC INFORMATION",
			title:  "Licences",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderStatic(t, tc.page)
			for _, expected := range []string{
				tc.kicker,
				">" + tc.title + "</h1>",
				"text-(length:--t-32)",
				"md:text-(length:--t-44)",
				"text-(length:--t-11)",
			} {
				if !strings.Contains(rendered, expected) {
					t.Errorf("rendered %s static page does not contain %q\ngot: %s", tc.name, expected, rendered)
				}
			}
			if strings.Count(rendered, "<h1") != 1 {
				t.Errorf("h1 count = %d, want 1", strings.Count(rendered, "<h1"))
			}
		})
	}
}

func TestStaticPageOmitsUnsupportedArchiveIntro(t *testing.T) {
	rendered := renderStatic(t, StaticPageDTO{Title: "About", Url: "/pages/about", Content: "<p>Body.</p>"})

	for _, unsupported := range []string{
		"What this archive contains",
		"how it is organised",
		"terms under which it may be used",
	} {
		if strings.Contains(rendered, unsupported) {
			t.Errorf("rendered about page must not contain unsupported intro copy %q\ngot: %s", unsupported, rendered)
		}
	}
}

func TestStaticPageSelectsKickerFromRouteNotEditableTitle(t *testing.T) {
	// A page whose editable title happens to read "About" but whose route is not
	// the about page must fall back to the generic kicker.
	rendered := renderStatic(t, StaticPageDTO{Title: "About", Url: "/pages/contact", Content: "<p>Body.</p>"})
	if strings.Contains(rendered, "14 — ABOUT THE COLLECTION") {
		t.Fatal("kicker must be selected from sp.Url, not the editable title")
	}
	if !strings.Contains(rendered, "PUBLIC INFORMATION") {
		t.Fatal("generic route must render the generic kicker")
	}
}

func TestStaticPageRetainsTableOfContentsAndManagedContent(t *testing.T) {
	rendered := renderStatic(t, StaticPageDTO{
		Title:   "About",
		Url:     "/pages/about",
		Content: `<h2 id="terms">Terms of use</h2><p>The collection is protected.</p>`,
		TOC:     []StaticPageTOCItem{{ID: "terms", Title: "Terms of use", Level: 2}, {ID: "detail", Title: "Detail", Level: 3}},
	})

	for _, expected := range []string{
		`<nav`,
		`aria-label="Contents"`,
		"CONTENTS",
		`<ol`,
		`<li`,
		`href="#terms"`,
		`href="#detail"`,
		">Terms of use</a>",
		`<article`,
		">Terms of use</h2>",
		">The collection is protected.</p>",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered static page does not contain %q\ngot: %s", expected, rendered)
		}
	}

	// Level-3 entries are visually indented but stay inside the ordered list.
	if !strings.Contains(rendered, `pl-4`) {
		t.Error("level-3 table of contents entries must be indented")
	}
}
