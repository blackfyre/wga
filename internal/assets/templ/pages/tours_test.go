package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestToursPageExplainsEditorialContractAndEmptyState(t *testing.T) {
	view := dto.TourIndex{Filters: []dto.TourFilter{
		{Label: "All", Href: "/tours", Active: true}, {Label: "Survey", Href: "/tours?kind=survey"},
		{Label: "Artist", Href: "/tours?kind=artist"}, {Label: "Site", Href: "/tours?kind=site"}, {Label: "Theme", Href: "/tours?kind=theme"},
	}}
	var output bytes.Buffer
	if err := ToursPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	for _, expected := range []string{"permanent, revisioned works by named editors", "Visitor itineraries", "expire", "No published tours yet", `href="/itineraries"`, `href="/tours?kind=survey"`, "Survey", "Artist", "Site", "Theme",
		"WRITTEN BY", "SHAPE", "LENGTH", "READING", "LIFETIME",
		"under their own names", "page by page", "never removed; visitor itineraries expire",
		"Rebuilt tours state their derived page count on their cards; original-layout tours keep their original form.",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("missing %q", expected)
		}
	}
}

func TestToursPageGroupsRebuiltAndOriginalWithNumbers(t *testing.T) {
	view := dto.TourIndex{
		Filters: []dto.TourFilter{{Label: "All", Href: "/tours", Active: true}},
		Rebuilt: []dto.TourCard{
			{Slug: "tour-a", Title: "Rebuilt Tour", Kind: "survey", Number: "1", Editor: "Named Editor", Revision: "Revised 2002", Blurb: "A bounded editorial reading.", Pages: 5},
		},
		Original: []dto.TourCard{
			{Slug: "tour-b", Title: "Original Tour", Kind: "site", Number: "6a", Editor: "Named Editor", Revision: "Revision 1", Blurb: "A legacy reading."},
		},
	}
	var output bytes.Buffer
	if err := ToursPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	for _, expected := range []string{
		"REBUILT PAGE BY PAGE", "1 TOURS PUBLISHED",
		"THE REST OF THE SERIES", "1 IN THE ORIGINAL LAYOUT",
		"survey · #1", "TOUR #6a",
		"5 PAGES · SCOPE NOT YET APPROVED",
		`hx-get="/tours/tour-a"`, `hx-get="/tours/tour-b"`, `hx-target="#tours"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("missing %q", expected)
		}
	}
}

func TestTourPageRendersContentsSectionHeadingsAsDistinctRows(t *testing.T) {
	view := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Text", PageType: "text", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 4, Section: "Opening",
		Contents: []dto.TourContentsItem{
			{Number: 1, Title: "Fixture Tour", Href: "/tours/fixture"},
			{Number: 2, Title: "Text", Section: "Opening", SectionID: "sec-1", Href: "/tours/fixture/2", Current: true},
			{Number: 3, Title: "Picture", Section: "Plates", SectionID: "sec-2", Href: "/tours/fixture/3"},
			{Number: 4, Title: "Index", Section: "Plates", SectionID: "sec-2", Href: "/tours/fixture/4"},
		},
	}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	// "Opening" appears once as its own section heading row, not repeated per page.
	if got := strings.Count(body, ">Opening</li>"); got != 1 {
		t.Errorf("section heading row count = %d, want 1: %s", got, body)
	}
	// "Plates" appears once as a section heading row even though two pages share it.
	if got := strings.Count(body, ">Plates</li>"); got != 1 {
		t.Errorf("section heading row count = %d, want 1: %s", got, body)
	}
}

func TestTourPageRendersDistinctSectionsWithIdenticalTitles(t *testing.T) {
	view := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Text", PageType: "text", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 4,
		Contents: []dto.TourContentsItem{
			{Number: 1, Title: "Fixture Tour", Href: "/tours/fixture"},
			{Number: 2, Title: "First", Section: "Opening", SectionID: "sec-1", Href: "/tours/fixture/2", Current: true},
			{Number: 3, Title: "Second", Section: "Opening", SectionID: "sec-2", Href: "/tours/fixture/3"},
			{Number: 4, Title: "Third", Section: "Opening", SectionID: "sec-2", Href: "/tours/fixture/4"},
		},
	}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	// Two distinct sections share the title "Opening" but carry different IDs,
	// so each must render its own heading row.
	if got := strings.Count(body, ">Opening</li>"); got != 2 {
		t.Errorf("distinct same-title sections rendered %d headings, want 2: %s", got, body)
	}
}

func TestTourPageShowsArrowTurnGuidanceContextually(t *testing.T) {
	multi := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Text", PageType: "text", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 3,
	}
	var output bytes.Buffer
	if err := TourPage(multi).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "← → TO TURN THE PAGE") {
		t.Errorf("multi-page tour missing arrow guidance: %s", output.String())
	}

	single := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Fixture Tour", PageType: "title", Kind: "site", Number: "6a",
		Address: 1, TotalPages: 1, PresentationStatus: "original",
		Contents: []dto.TourContentsItem{{Number: 1, Title: "Fixture Tour", Href: "/tours/fixture", Current: true}},
	}
	output.Reset()
	if err := TourPage(single).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "← → TO TURN THE PAGE") {
		t.Errorf("single-page tour must not offer arrow turn guidance")
	}
}

func TestTourBlocksRenderFragmentsWithoutShell(t *testing.T) {
	index := dto.TourIndex{Filters: []dto.TourFilter{{Label: "All", Href: "/tours", Active: true}}}
	var output bytes.Buffer
	if err := ToursBlock(index).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), `id="mc-area"`) {
		t.Errorf("ToursBlock must not carry the shared shell #mc-area")
	}
	if !strings.Contains(output.String(), `id="tours"`) {
		t.Errorf("ToursBlock missing its #tours section")
	}

	page := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Text", PageType: "text", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 3,
	}
	output.Reset()
	if err := TourBlock(page).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), `id="mc-area"`) {
		t.Errorf("TourBlock must not carry the shared shell #mc-area")
	}
	if !strings.Contains(output.String(), `id="tour"`) || !strings.Contains(output.String(), `data-tour-reading`) {
		t.Errorf("TourBlock missing its #tour article")
	}
}

func TestTourPageRendersAddressContextViewerAndOrdinaryNavigation(t *testing.T) {
	view := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Picture", PageType: "picture", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 3, DisplayURL: "/image?thumb=1400x0", ZoomURL: "/image",
		ArtworkAlt: "Work", ArtworkURL: "/artworks/work", ArtworkCredit: "Credit", PreviousURL: "/tours/fixture", PreviousLabel: "Fixture Tour", NextURL: "/tours/fixture/3", NextLabel: "Next",
		Contents: []dto.TourContentsItem{{Number: 1, Title: "Fixture Tour", Href: "/tours/fixture"}, {Number: 2, Title: "Picture", Href: "/tours/fixture/2", Current: true}},
	}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	for _, expected := range []string{`data-viewer`, `data-zoom-url="/image"`, `href="/image"`, `src="/image?thumb=1400x0"`, `href="/tours/fixture"`, `rel="prev"`, `href="/tours/fixture/3"`, `rel="next"`, `aria-current="page"`, "PAGE 02 OF 03", "GUIDED TOURS", "Credit", `data-tour-reading`, `data-tour-nav="prev"`, `data-tour-nav="next"`} {
		if !strings.Contains(body, expected) {
			t.Errorf("missing %q", expected)
		}
	}
	// The ViewerJS url callback reads data-zoom-url from the <img>, so the
	// 1400px display source and the 2000px zoom source must be distinct and the
	// zoom source must sit on the image element itself.
	if !strings.Contains(body, `<img src="/image?thumb=1400x0" alt="Work" loading="lazy" data-zoom-url="/image"`) {
		t.Errorf("picture img must carry the zoom source on the img element: %s", body)
	}
	if !strings.Contains(body, `href="/image" class="relative block h-full w-full" data-viewer`) {
		t.Errorf("picture anchor must keep the ordinary zoom fallback link and viewer trigger: %s", body)
	}
}

func TestTourPageRendersReferenceReadingStructure(t *testing.T) {
	title := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Fixture Tour", PageType: "title", Kind: "survey", Number: "6a",
		Editor: "Named Editor", PublishedYear: 2001, RevisedYear: 2002, Address: 1, TotalPages: 3, NextURL: "/tours/fixture/2", NextLabel: "Text page",
		Contents: []dto.TourContentsItem{{Number: 1, Title: "Fixture Tour", Href: "/tours/fixture", Current: true}},
	}
	var output bytes.Buffer
	if err := TourPage(title).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	for _, expected := range []string{
		`role="progressbar"`, `TOUR #6a`, "PAGE 01 OF 03",
		"CONTENTS", "EDITOR", "FIRST PUBLISHED", "LAST REVISED",
		"Named Editor", "2001", "2002", "START THE TOUR →",
		"© Web Gallery of Art.",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("title structure missing %q", expected)
		}
	}

	picture := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Picture", PageType: "picture", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 3, DisplayURL: "/image", ZoomURL: "/zoom", ArtworkAlt: "Work", ArtworkURL: "/artworks/work",
	}
	output.Reset()
	if err := TourPage(picture).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body = output.String()
	for _, expected := range []string{"CLICK TO ZOOM", "OPEN THE FULL RECORD →", `data-zoom-url="/zoom"`,
		`h-[300px] md:h-[460px] items-center justify-center`, `object-contain`} {
		if !strings.Contains(body, expected) {
			t.Errorf("picture structure missing %q", expected)
		}
	}
	// The frame is fixed at mobile/desktop heights with the image contained
	// inside it, rather than a natural-aspect plate.
	if !strings.Contains(body, `class="h-full w-full object-contain"`) {
		t.Errorf("picture image must use contained presentation in a fixed frame: %s", body)
	}

	list := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Index", PageType: "list", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 3, IndexRows: []dto.TourIndexEntry{{Name: "Safe", Dates: "1500–1550", TargetPath: "/artists/approved"}},
	}
	output.Reset()
	if err := TourPage(list).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body = output.String()
	if !strings.Contains(body, `data-kbd-list`) || !strings.Contains(body, `data-kbd-cols="1"`) {
		t.Errorf("list structure missing keyboard list markers: %s", body)
	}

	sources := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Sources", PageType: "sources", Kind: "survey", Number: "6a",
		Address: 3, TotalPages: 3, PublishedYear: 2001,
		Sources: []dto.TourSource{{Citation: "Approved source"}},
	}
	output.Reset()
	if err := TourPage(sources).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body = output.String()
	for _, expected := range []string{"END OF THE TOUR", "Sources", "01", "Approved source"} {
		if !strings.Contains(body, expected) {
			t.Errorf("sources structure missing %q", expected)
		}
	}
}

func TestTourPageRendersListRowsWithSafeTargets(t *testing.T) {
	view := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Index", PageType: "list", Kind: "survey",
		Address: 3, TotalPages: 4,
		IndexRows: []dto.TourIndexEntry{{Name: "Safe", Dates: "1500–1550", TargetPath: "/artists/approved"}, {Name: "Unsafe", TargetPath: ""}},
	}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	if !strings.Contains(body, `href="/artists/approved"`) || !strings.Contains(body, "Unsafe") {
		t.Fatalf("list rows missing: %s", body)
	}
}

func TestOriginalTourOnlyRendersStoredSafeDestination(t *testing.T) {
	view := dto.TourPage{Slug: "old", TourTitle: "Old Tour", PageTitle: "Old Tour", PageType: "title", PresentationStatus: "original", Kind: "site", Number: "6a", Editor: "Editor", PublishedYear: 1998, LegacyURL: "https://example.org/tour", Contents: []dto.TourContentsItem{{Number: 1, Title: "Old Tour", Href: "/tours/old", Current: true}}}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	if !strings.Contains(body, "STILL IN ITS ORIGINAL LAYOUT") || !strings.Contains(body, `href="https://example.org/tour"`) {
		t.Fatalf("safe legacy output missing: %s", body)
	}
	if strings.Contains(body, "START THE TOUR") {
		t.Fatalf("original tour must not offer a start-the-tour call to action: %s", body)
	}
	view.LegacyURL = ""
	output.Reset()
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body = output.String()
	if strings.Contains(body, "Open the original tour") || !strings.Contains(body, "No safe original destination") {
		t.Fatalf("unsafe legacy fallback wrong: %s", body)
	}
}

func TestTourLinkHtmxBoundaries(t *testing.T) {
	view := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Text", PageType: "text", Kind: "survey", Number: "6a",
		Address: 2, TotalPages: 4,
		PreviousURL: "/tours/fixture", PreviousLabel: "Fixture Tour", NextURL: "/tours/fixture/3", NextLabel: "Next",
		Contents: []dto.TourContentsItem{
			{Number: 1, Title: "Fixture Tour", Href: "/tours/fixture"},
			{Number: 2, Title: "Text", Section: "Opening", SectionID: "sec-1", Href: "/tours/fixture/2", Current: true},
			{Number: 3, Title: "Picture", Section: "Opening", SectionID: "sec-1", Href: "/tours/fixture/3"},
		},
	}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()

	// Same-tour reading navigation (rail contents, page turns) replaces #tour.
	if !strings.Contains(body, `hx-get="/tours/fixture/3" hx-target="#tour" hx-select="#tour" hx-swap="outerHTML"`) {
		t.Errorf("contents link must target #tour: %s", body)
	}
	if !strings.Contains(body, `hx-get="/tours/fixture" hx-target="#tour" hx-select="#tour" hx-swap="outerHTML" rel="prev"`) {
		t.Errorf("previous turn must target #tour: %s", body)
	}
	if !strings.Contains(body, `hx-get="/tours/fixture/3" hx-target="#tour" hx-select="#tour" hx-swap="outerHTML" rel="next"`) {
		t.Errorf("next turn must target #tour: %s", body)
	}

	// The breadcrumb to the index is cross-page and inherits the shell's #mc-area.
	if !strings.Contains(body, `href="/tours" hx-get="/tours" class="text-primary"`) {
		t.Errorf("breadcrumb must use the ordinary #mc-area shell navigation: %s", body)
	}
}

func TestTourTitleUsesSharedPlate(t *testing.T) {
	view := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Fixture Tour", PageType: "title", Kind: "survey", Number: "6a",
		Address: 1, TotalPages: 1, DisplayURL: "/image?thumb=1000x0",
	}
	var output bytes.Buffer
	if err := TourPage(view).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	if !strings.Contains(body, `src="/image?thumb=1000x0"`) || !strings.Contains(body, `loading="lazy"`) {
		t.Errorf("title plate missing display image/lazy behaviour: %s", body)
	}
	if strings.Contains(body, `data-viewer`) {
		t.Errorf("title plate must not open a viewer: %s", body)
	}

	empty := dto.TourPage{
		Slug: "fixture", TourTitle: "Fixture Tour", PageTitle: "Fixture Tour", PageType: "title", Kind: "survey", Number: "6a",
		Address: 1, TotalPages: 1,
	}
	output.Reset()
	if err := TourPage(empty).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body = output.String()
	if !strings.Contains(body, "IMAGE — Fixture Tour") {
		t.Errorf("title plate missing placeholder: %s", body)
	}
}
