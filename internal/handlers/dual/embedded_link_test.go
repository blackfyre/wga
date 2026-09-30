package dual

import (
	neturl "net/url"
	"strings"
	"testing"
)

// escapedDualHref is a routed /dual-mode URL as html.Render writes it in an
// attribute value.
func escapedDualHref(state dualState, side string, path string) string {
	return strings.ReplaceAll(state.withPanePath(side, path).path(), "&", "&amp;")
}

func TestDualEmbeddedLinkRoutesToOppositePane(t *testing.T) {
	state := parseDualState(mustParseQuery(t, "left=/artists/rembrandt-artistone000001&right=/artists/vermeer-artisttwo000001"))
	input := `<p>See <a href="/artists/hals-artistthree0001" title="Hals">Frans <em>Hals</em></a> too.</p>`

	got := dualRoutedProseHTML(input, state.left, state)

	href := escapedDualHref(state, "right", "/artists/hals-artistthree0001")
	for _, fragment := range []string{
		`href="` + href + `"`,
		`hx-get="` + href + `"`,
		`hx-target="#dual-right"`,
		`hx-select="#dual-right"`,
		`hx-swap="outerHTML"`,
		`title="Hals"`,
		`<p>See <a `,
		`>Frans <em>Hals</em></a> too.</p>`,
	} {
		if !strings.Contains(got, fragment) {
			t.Errorf("routed link missing %q in %q", fragment, got)
		}
	}
	if !strings.Contains(href, "left=%2Fartists%2Frembrandt-artistone000001") || !strings.Contains(href, "right=%2Fartists%2Fhals-artistthree0001") {
		t.Fatalf("routed URL must keep the left pane and load the right pane, got %q", href)
	}
}

func TestDualEmbeddedLinkRoutesToSamePane(t *testing.T) {
	state := parseDualState(mustParseQuery(t, "left=/artists/rembrandt-artistone000001&left_render_to=left&right=/artists/vermeer-artisttwo000001"))
	input := `<p><a href="/artists/hals-artistthree0001/">Hals</a></p>`

	got := dualRoutedProseHTML(input, state.left, state)

	href := escapedDualHref(state, "left", "/artists/hals-artistthree0001")
	if !strings.Contains(got, `href="`+href+`"`) || !strings.Contains(got, `hx-target="#dual-left"`) || !strings.Contains(got, `hx-select="#dual-left"`) {
		t.Fatalf("same-pane routed link = %q, want href %q targeting #dual-left", got, href)
	}
	if !strings.Contains(href, "right=%2Fartists%2Fvermeer-artisttwo000001") || !strings.Contains(href, "left_render_to=left") {
		t.Fatalf("routed URL must keep the right pane and routing state, got %q", href)
	}
}

func TestDualEmbeddedLinkRoutesArtworkAndSelectionRoutes(t *testing.T) {
	state := parseDualState(neturl.Values{})
	for _, path := range []string{
		"/artists/hals-artistthree0001/laughing-cavalier-artworkthree01",
		"/artists/hals-artistthree0001/artworks/laughing-cavalier-artworkthree01",
		"/artworks/laughing-cavalier-artworkthree01",
		"/artists/hals-artistthree0001/selections/selectionthree1",
	} {
		got := dualRoutedProseHTML(`<a href="`+path+`">x</a>`, state.left, state)
		href := escapedDualHref(state, "right", path)
		if !strings.Contains(got, `href="`+href+`"`) || !strings.Contains(got, `hx-get="`+href+`"`) {
			t.Errorf("record link %q = %q, want routed href %q", path, got, href)
		}
	}
}

func TestDualEmbeddedLinkLeavesOtherLinksUnchanged(t *testing.T) {
	state := parseDualState(mustParseQuery(t, "left=/artists/rembrandt-artistone000001"))
	for _, input := range []string{
		`<p><a href="https://example.org/artists/hals-artistthree0001">external</a></p>`,
		`<p><a href="//example.org/artists/hals-artistthree0001">protocol relative</a></p>`,
		`<p><a href="/glossary">glossary</a> and <a href="/artists">index</a></p>`,
		`<p><a href="artists/hals-artistthree0001">relative</a></p>`,
		`<p>No links here.</p>`,
	} {
		if got := dualRoutedProseHTML(input, state.left, state); got != input {
			t.Errorf("dualRoutedProseHTML(%q) = %q, want unchanged", input, got)
		}
	}
}

func TestDualEmbeddedLinkSkipsGlossaryDefinitionTemplates(t *testing.T) {
	state := parseDualState(neturl.Values{})
	input := `<p><span class="glossary-term">term<template class="glossary-definition"><a href="/artists/hals-artistthree0001">def</a></template></span> <a href="/artists/vermeer-artisttwo000001">Vermeer</a></p>`

	got := dualRoutedProseHTML(input, state.left, state)

	if !strings.Contains(got, `<a href="/artists/hals-artistthree0001">def</a>`) {
		t.Errorf("glossary definition link must be unchanged, got %q", got)
	}
	if strings.Count(got, "hx-get=") != 1 {
		t.Errorf("only the prose link should be routed, got %q", got)
	}
}

func TestDualEmbeddedLinkRoutesBiographyAndSelectionCommentary(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	artist, err := app.FindRecordById("artists", "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("bio", `<p>Pupil of <a href="/artists/lastman-artistlast00001">Lastman</a>; see <a href="https://example.org/">elsewhere</a>.</p>`)
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist bio: %v", err)
	}
	saveDualRecord(t, app, "art_selections", "selectionone001", map[string]any{
		"artist": "artistone000001", "title": "Paintings", "display_title": "Paintings",
		"commentary": `<p>Compare <a href="/artists/rembrandt-artistone000001/the-night-watch-artworkone00001">the Night Watch</a>.</p>`,
		"artworks":   []string{"artworkone00001"}, "published": true,
	})
	saveDualRecord(t, app, "art_selections", "selectiontwo001", map[string]any{
		"artist": "artistone000001", "title": "Drawings", "display_title": "Drawings",
		"commentary": "<p>Second.</p>", "artworks": []string{"artworkone00001"}, "published": true,
	})

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}
	state := parseDualState(mustParseQuery(t, "left=/artists/rembrandt-artistone000001"))
	record, err := buildDualArtistRecord(app, "left", state.left, state, ref)
	if err != nil {
		t.Fatalf("build artist record: %v", err)
	}
	bioHref := escapedDualHref(state, "right", "/artists/lastman-artistlast00001")
	if !strings.Contains(record.Bio, `href="`+bioHref+`"`) || !strings.Contains(record.Bio, `hx-target="#dual-right"`) {
		t.Errorf("biography record link not routed to the right pane: %q", record.Bio)
	}
	if !strings.Contains(record.Bio, `<a href="https://example.org/" rel="nofollow">elsewhere</a>`) {
		t.Errorf("external biography link must be unchanged: %q", record.Bio)
	}

	artworkPath := "/artists/rembrandt-artistone000001/the-night-watch-artworkone00001"
	var previewCommentary string
	for _, preview := range record.Selections {
		if preview.DisplayTitle == "Paintings" {
			previewCommentary = preview.Commentary
		}
	}
	if !strings.Contains(previewCommentary, `href="`+escapedDualHref(state, "right", artworkPath)+`"`) || !strings.Contains(previewCommentary, `hx-select="#dual-right"`) {
		t.Errorf("selection preview commentary link not routed: %q", previewCommentary)
	}

	state.left.path = "/artists/rembrandt-artistone000001/selections/selectionone001"
	window, err := buildWindow(app, "left", state.left, state, ref)
	if err != nil {
		t.Fatalf("build selection window: %v", err)
	}
	if !strings.Contains(window.Selection.Commentary, `href="`+escapedDualHref(state, "right", artworkPath)+`"`) || !strings.Contains(window.Selection.Commentary, `hx-swap="outerHTML"`) {
		t.Errorf("selection record commentary link not routed: %q", window.Selection.Commentary)
	}
}
