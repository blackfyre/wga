package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTopNavRendersPrimaryAndMoreDestinations(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	region := desktopNavRegion(output.String())
	if region == "" {
		t.Fatal("expected desktop navigation region")
	}
	assertDestinationOrder(t, region, []string{
		`href="/artists" hx-get="/artists">ARTISTS`,
		`href="/artworks" hx-get="/artworks">ARTWORKS`,
		`href="/timeline">TIMELINE`,
		`href="/dual-mode" hx-get="/dual-mode">DUAL MODE`,
		`href="/tours">GUIDED TOURS`,
		`href="/itineraries">ITINERARIES`,
		`href="/inspire" hx-get="/inspire">INSPIRATION`,
	})
	assertDestinationOrder(t, region, []string{
		`href="/statistics"`,
		`href="/glossary"`,
		`href="/guestbook"`,
		`href="/postcard"`,
		`href="/pages/about"`,
	})
}

func TestTopNavMobileDisclosureListsFlatInventoryInOrder(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	region := mobileNavRegion(output.String())
	if region == "" {
		t.Fatal("expected mobile navigation region")
	}

	if strings.Contains(region, "<details") {
		t.Fatal("mobile disclosure must not nest a MORE disclosure")
	}

	assertDestinationOrder(t, region, []string{
		`href="/artists"`,
		`href="/artworks"`,
		`href="/dual-mode"`,
		`href="/timeline"`,
		`href="/inspire"`,
		`href="/tours"`,
		`href="/itineraries"`,
		`href="/itineraries/new"`,
		`href="/statistics"`,
		`href="/glossary"`,
		`href="/guestbook"`,
		`href="/postcard"`,
		`href="/pages/about"`,
		`href="/contributors"`,
		`href="/pages/privacy-policy"`,
	})

	for _, expected := range []string{
		`href="/artists" hx-get="/artists"`,
		`href="/artworks" hx-get="/artworks"`,
		`href="/dual-mode" hx-get="/dual-mode"`,
		`href="/inspire" hx-get="/inspire"`,
		`href="/statistics" hx-get="/statistics"`,
		`href="/guestbook" hx-get="/guestbook"`,
		`href="/postcard" hx-get="/postcard"`,
		`href="/pages/about" hx-get="/pages/about"`,
		`href="/contributors" hx-get="/contributors"`,
		`href="/pages/privacy-policy" hx-get="/pages/privacy-policy"`,
	} {
		if !strings.Contains(region, expected) {
			t.Fatalf("expected hx-get convention %q", expected)
		}
	}

	for _, plain := range []string{
		`href="/timeline" hx-get`,
		`href="/tours" hx-get`,
		`href="/itineraries" hx-get`,
		`href="/itineraries/new" hx-get`,
		`href="/glossary" hx-get`,
	} {
		if strings.Contains(region, plain) {
			t.Fatalf("expected ordinary anchor without hx-get, found %q", plain)
		}
	}

	if got := strings.Count(region, "→"); got != 15 {
		t.Fatalf("expected 15 arrow-ended rows, got %d", got)
	}
}

func TestTopNavMobileDisclosurePreservesKeyboardContracts(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`<details class="col-span-full row-start-1" data-kbd-mobile-navigation>`,
		`<summary class="ml-auto flex h-11 w-11 cursor-pointer list-none flex-col items-center justify-center gap-1.5" aria-label="Open primary navigation">`,
		`<nav class="mt-4 border-t border-base-content/15 bg-base-100 pt-4" aria-label="Primary navigation" data-mobile-navigation`,
		`data-kbd-search`,
		`hx-on:click="if (event.target.closest('a')) this.closest('details').removeAttribute('open')"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected navigation contract %q", expected)
		}
	}
}

func TestTopNavDoesNotDuplicateAppearanceControl(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	if strings.Contains(output.String(), `data-wga-theme-toggle`) {
		t.Fatal("appearance control belongs in the shared footer")
	}
}

func TestTopNavKeepsMobileIdentityOutsideDisclosureContent(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	rendered := output.String()
	identity := strings.Index(rendered, `<a class="col-start-1 row-start-1 z-10 flex items-center gap-3.5" href="/" hx-get="/">`)
	disclosure := strings.Index(rendered, `<details class="col-span-full row-start-1" data-kbd-mobile-navigation>`)
	if identity < 0 || disclosure < 0 || identity > disclosure {
		t.Fatal("expected mobile identity to precede and remain outside the disclosure content")
	}
}

func TestTopNavKeepsMoreInlineWithDesktopDestinations(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	rendered := output.String()
	if strings.Contains(rendered, `items-start justify-between gap-6`) {
		t.Fatal("desktop navigation must not push MORE to the far edge")
	}
	if !strings.Contains(rendered, `hidden items-start gap-6 px-4 pt-3 md:flex`) {
		t.Fatal("expected inline desktop navigation layout")
	}
}

func TestTopNavMobileToggleMeetsTouchTarget(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	const summary = `<summary class="ml-auto flex h-11 w-11 cursor-pointer list-none flex-col items-center justify-center gap-1.5" aria-label="Open primary navigation">`
	if !strings.Contains(output.String(), summary) {
		t.Fatal("expected mobile toggle summary to expose a 44px touch target")
	}
}

func TestTopNavMobileDisclosureUsesNormalFlowFullWidthPanel(t *testing.T) {
	var output bytes.Buffer
	if err := TopNav().Render(context.Background(), &output); err != nil {
		t.Fatalf("render top navigation: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`<div class="grid grid-cols-[minmax(0,1fr)_2.75rem] items-start gap-6 md:hidden">`,
		`<details class="col-span-full row-start-1" data-kbd-mobile-navigation>`,
		`<summary class="ml-auto flex h-11 w-11 cursor-pointer list-none flex-col items-center justify-center gap-1.5" aria-label="Open primary navigation">`,
		`<nav class="mt-4 border-t border-base-content/15 bg-base-100 pt-4" aria-label="Primary navigation" data-mobile-navigation`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected mobile disclosure panel contract %q", expected)
		}
	}
}

func assertDestinationOrder(t *testing.T, rendered string, destinations []string) {
	t.Helper()
	position := 0
	for _, destination := range destinations {
		next := strings.Index(rendered[position:], destination)
		if next < 0 {
			t.Fatalf("expected destination %q", destination)
		}
		position += next + len(destination)
	}
}

func mobileNavRegion(rendered string) string {
	start := strings.Index(rendered, "data-mobile-navigation")
	if start < 0 {
		return ""
	}
	end := strings.Index(rendered[start:], "</nav>")
	if end < 0 {
		return ""
	}
	return rendered[start : start+end+len("</nav>")]
}

func desktopNavRegion(rendered string) string {
	marker := `class="container mx-auto hidden items-start gap-6 px-4 pt-3 md:flex md:px-0 md:pt-0" aria-label="Primary navigation"`
	start := strings.Index(rendered, marker)
	if start < 0 {
		return ""
	}
	end := strings.Index(rendered[start:], "</nav>")
	if end < 0 {
		return ""
	}
	return rendered[start : start+end+len("</nav>")]
}
