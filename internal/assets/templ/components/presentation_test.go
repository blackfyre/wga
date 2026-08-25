package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderComponent(t *testing.T, component templ.Component) string {
	return renderComponentWithChildren(t, component, templ.NopComponent)
}

func renderComponentWithChildren(t *testing.T, component templ.Component, children templ.Component) string {
	t.Helper()

	var output bytes.Buffer
	if err := component.Render(templ.WithChildren(context.Background(), children), &output); err != nil {
		t.Fatalf("render component: %v", err)
	}

	return output.String()
}

func TestPageHeadAndSectionRuleUseHeadingHierarchy(t *testing.T) {
	head := renderComponent(t, PageHead("01", "Collection", "Browse the collection."))
	if !strings.Contains(head, "<h1") || !strings.Contains(head, ">Collection</h1>") {
		t.Fatal("page head must provide the page h1")
	}
	for _, expected := range []string{"text-(length:--t-11)", "text-muted", "text-(length:--t-32)", "md:text-(length:--t-44)"} {
		if !strings.Contains(head, expected) {
			t.Fatalf("page head missing relative type token %s", expected)
		}
	}

	rule := renderComponent(t, SectionRule("Featured works", "View all", "/artworks"))
	if !strings.Contains(rule, "<h2") || !strings.Contains(rule, ">Featured works</h2>") {
		t.Fatal("section rule must provide a section h2")
	}
	if !strings.Contains(rule, `href="/artworks"`) || !strings.Contains(rule, "text-(length:--t-11)") || !strings.Contains(rule, "transition-opacity") {
		t.Fatal("section rule must render its ordinary link")
	}
}

func TestWorkCardsAndRowsAreOrdinaryLinksWithoutViewerHook(t *testing.T) {
	work := dto.Work{URL: "/artworks/work", ImageURL: "/images/work.jpg", Title: "Work", Artist: "Artist", Metadata: "Oil on canvas"}

	for _, component := range []templ.Component{WorkCard(work), WorkRow(work)} {
		rendered := renderComponent(t, component)
		if !strings.HasPrefix(rendered, "<a ") || strings.Contains(rendered, "<article") {
			t.Fatal("work presentation must use its ordinary link as the root element without an article wrapper")
		}
		if !strings.Contains(rendered, `href="/artworks/work"`) {
			t.Fatal("work presentation must use its ordinary record link")
		}
		if strings.Contains(rendered, "data-viewer") || strings.Contains(rendered, "data-zoom-url") {
			t.Fatal("work presentation must not mount the viewer")
		}
		if strings.Contains(rendered, "thumb=") {
			t.Fatal("work presentation must use its supplied image URL unchanged")
		}
		if !strings.Contains(rendered, "text-(length:--t-11)") {
			t.Fatal("work presentation must use relative metadata type")
		}
	}

	card := renderComponent(t, WorkCard(work))
	if !strings.Contains(card, `class="flex items-end border border-base-content/15 bg-base-300 p-2.5 aspect-[4/5] mb-2.5"`) || strings.Contains(card, `class="mt-3 text-sm`) {
		t.Fatal("work card must place its reference spacing on the plate rather than the title")
	}

	row := renderComponentWithChildren(t, WorkRow(work), templ.Raw(`<span class="extra-column">Extra</span>`))
	if !strings.Contains(row, `<span class="extra-column">Extra</span>`) {
		t.Fatal("work row must render extension columns inside its link")
	}
}

func TestMetaListUsesTermsValuesAndRetainsIntentionalBlankValue(t *testing.T) {
	rendered := renderComponent(t, MetaList([]MetaEntry{{Label: "Date", Value: "1500"}, {Label: "Location", Value: ""}}))
	if !strings.Contains(rendered, "<dl") || !strings.Contains(rendered, ">Date</dt>") || !strings.Contains(rendered, ">1500</dd>") || !strings.Contains(rendered, "text-(length:--t-11)") || !strings.Contains(rendered, "text-sm") {
		t.Fatal("metadata must render a definition list with term and value")
	}
	if !strings.Contains(rendered, ">Location</dt>") || !strings.Contains(rendered, "></dd>") {
		t.Fatal("metadata must retain an intentional blank value")
	}
}

func TestPlateUsesSeparateDisplayAndZoomURLsWithAccessibleFallbacks(t *testing.T) {
	rendered := renderComponent(t, Plate(dto.Plate{
		DisplayURL: "/images/display.jpg",
		ZoomURL:    "/images/zoom.jpg",
		Alt:        "A painted scene",
		Label:      "Open painted scene",
	}))
	for _, expected := range []string{`<a href="/images/zoom.jpg"`, `src="/images/display.jpg"`, `data-viewer`, `data-viewer-label="Open painted scene"`, `alt="A painted scene"`, `aria-label="Open painted scene"`, "CLICK TO ZOOM"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("plate missing %s", expected)
		}
	}
	// ViewerJS resolves url(image) from the <img>, so the zoom source must live
	// on the image element itself, distinct from the display src, while the
	// anchor keeps only its ordinary no-JS href fallback.
	if !strings.Contains(rendered, `<img src="/images/display.jpg" alt="A painted scene" loading="lazy" data-zoom-url="/images/zoom.jpg"`) {
		t.Fatalf("plate must carry the zoom source on the img element: %s", rendered)
	}
	if got := strings.Count(rendered, "data-zoom-url"); got != 1 {
		t.Fatalf("data-zoom-url rendered %d times, want exactly once on the img: %s", got, rendered)
	}
	if !strings.Contains(rendered, `<a href="/images/zoom.jpg" class="relative block h-full w-full" data-viewer`) {
		t.Fatalf("plate anchor must keep the ordinary zoom fallback link and viewer trigger: %s", rendered)
	}

	placeholder := renderComponent(t, Plate(dto.Plate{Placeholder: "Reproduction unavailable"}))
	if !strings.Contains(placeholder, "Reproduction unavailable") || !strings.Contains(placeholder, "text-(length:--t-9)") || !strings.Contains(placeholder, "text-faint-2") {
		t.Fatal("plate placeholder must be visible")
	}
}

func TestDialogBodyProvidesVisibleDismissalAndInitialFocusHook(t *testing.T) {
	rendered := renderComponent(t, DialogBody())
	for _, expected := range []string{`data-dialog-close`, `data-dialog-initial-focus`, `aria-label="Close dialog"`, `method="dialog"`, `class="modal-backdrop`} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("dialog body missing %s", expected)
		}
	}

	button := renderComponent(t, DialogButton("/feedback", "Feedback"))
	for _, expected := range []string{`hx-on:click="wga.dialog.open()"`, `hx-target="#d"`, `hx-select=".modal-box, form[method=dialog].modal-backdrop"`} {
		if !strings.Contains(button, expected) {
			t.Fatalf("dialog invoker missing %s", expected)
		}
	}
}

func TestFieldProvidesLabelledTextAndBoundedNoteControls(t *testing.T) {
	text := renderComponent(t, Field(dto.Field{ID: "search", Name: "q", Label: "Search the collection", Type: "search", Value: "Giotto", Required: true, MaxLength: 120, Error: "Enter at least two characters."}))
	for _, expected := range []string{`<label for="search"`, "Search the collection</label>", `id="search"`, `name="q"`, `type="search"`, `value="Giotto"`, `maxlength="120"`, "required", `aria-describedby="search-description"`, `role="alert"`, "Enter at least two characters.", "text-(length:--t-11)", "text-muted", "text-(length:--t-15)", "border-b", "focus:ring-0"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("field missing %s", expected)
		}
	}

	note := renderComponent(t, Field(dto.Field{ID: "note", Name: "note", Label: "Curator note", Type: "textarea", Value: "A note", Required: true, MaxLength: 500, Rows: 6}))
	for _, expected := range []string{`<textarea id="note"`, `name="note"`, `rows="6"`, `maxlength="500"`, "required", ">A note</textarea>"} {
		if !strings.Contains(note, expected) {
			t.Fatalf("note field missing %s", expected)
		}
	}

	plain := renderComponent(t, Field(dto.Field{ID: "title", Name: "title", Label: "Title"}))
	if strings.Contains(plain, "aria-describedby") {
		t.Fatal("field without a hint or error must not reference a missing description")
	}
	if !strings.Contains(text, `id="search-description"`) {
		t.Fatal("described field must render the referenced description")
	}
}

func TestChipGroupsUseRadioOptionsAndNavigationUsesOrdinaryLinks(t *testing.T) {
	group := renderComponent(t, ChipGroup(dto.ChipGroup{Legend: "Art form", Name: "form", Note: "Choose one form.", Inline: true, Options: []dto.ChipOption{{Label: "Painting", Value: "painting", Checked: true}, {Label: "Sculpture", Value: "sculpture"}}}))
	for _, expected := range []string{"<fieldset", "<legend", "Art form</legend>", `type="radio"`, `name="form"`, `value="painting"`, `value="sculpture"`, "checked", "Choose one form.", "text-(length:--t-11)", "gap-1.5", "border-control", "transition-colors"} {
		if !strings.Contains(group, expected) {
			t.Fatalf("chip group missing %s", expected)
		}
	}

	navigation := renderComponent(t, NavChips([]dto.NavChip{{Label: "Painting", Href: "/artworks?form=painting", Active: true}}))
	if !strings.Contains(navigation, `<a href="/artworks?form=painting"`) || !strings.Contains(navigation, `aria-current="page"`) {
		t.Fatal("selected navigation chip must be a current link")
	}
}

func TestRangeFieldProvidesDistinctDualInputsBoundsAndAssociatedOutput(t *testing.T) {
	rendered := renderComponent(t, RangeField(dto.RangeField{Label: "Date", FromID: "year-from", FromName: "year_from", FromValue: 1450, ToID: "year-to", ToName: "year_to", ToValue: 1550, Min: 1200, Max: 1600, Step: 10, Brush: true}))
	for _, expected := range []string{"<fieldset", "<legend", "Date</legend>", `id="year-from"`, `name="year_from"`, `value="1450"`, `aria-label="Range starts"`, `id="year-to"`, `name="year_to"`, `value="1550"`, `aria-label="Range ends"`, `min="1200"`, `max="1600"`, `step="10"`, `for="year-from year-to"`, ">1450–1550</output>", "wga-range-brush", "text-(length:--t-10)"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("range field missing %s", expected)
		}
	}
}

func TestRangeFieldHandlesDegenerateBounds(t *testing.T) {
	rendered := renderComponent(t, RangeField(dto.RangeField{Label: "Date", FromID: "year-from", FromName: "year_from", FromValue: 1500, ToID: "year-to", ToName: "year_to", ToValue: 1500, Min: 1500, Max: 1500}))
	if strings.Contains(rendered, "NaN") || !strings.Contains(rendered, "left:0") || !strings.Contains(rendered, "right:0") {
		t.Fatal("range field must provide a stable fill for degenerate bounds")
	}
}

func TestEmptyStateHasVisibleMeaningfulMessageAndRecoveryLink(t *testing.T) {
	rendered := renderComponent(t, EmptyState(dto.EmptyState{Title: "No works found", Message: "Try broadening your filters.", RecoveryLabel: "Clear filters", RecoveryHref: "/artworks"}))
	if !strings.Contains(rendered, "No works found") || !strings.Contains(rendered, "Try broadening your filters.") || !strings.Contains(rendered, `href="/artworks"`) || !strings.Contains(rendered, "py-14") || !strings.Contains(rendered, "border-b") || !strings.Contains(rendered, "text-(length:--t-11)") {
		t.Fatal("empty state must render visible title and guidance")
	}
}
