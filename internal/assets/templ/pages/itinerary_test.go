package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestItineraryBuilderRendersStopsNarrationAndCsrf(t *testing.T) {
	view := dto.ItineraryBuilderView{
		CSRF:       "csrf-token",
		Max:        15,
		Count:      1,
		PickerOpen: true,
		Stops: []dto.ItineraryStop{
			{ID: "stop-1", ArtworkID: "aw0000000000001", Title: "Work One", Artist: "Artist", Narration: "A note", Position: 0, URL: "/artists/artist/work"},
		},
		Picker: []dto.ItineraryPickerWork{
			{ArtworkID: "aw0000000000002", Title: "Work Two"},
		},
	}

	var output bytes.Buffer
	if err := ItineraryBuilder(view, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render builder: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`id="itinerary-builder"`,
		`name="_csrf" value="csrf-token"`,
		`id="itinerary-publish-form"`,
		`form="itinerary-publish-form"`,
		`name="narration.stop-1"`,
		"Work One",
		"A note",
		"YOUR NARRATION",
		`maxlength="600"`,
		`name="artwork_id" value="aw0000000000002"`,
		`action="/itineraries/draft/move"`,
		`action="/itineraries/draft/remove"`,
		"PUBLISH",
		"ORDER OF PRESENTATION",
		"CLOSE PICKER",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("builder does not contain %q", expected)
		}
	}
}

func TestItineraryBuilderFilmstripNarrationStatusAndSelectedEditor(t *testing.T) {
	view := dto.ItineraryBuilderView{
		CSRF:     "csrf-token",
		Max:      15,
		Count:    3,
		Selected: 1,
		Stops: []dto.ItineraryStop{
			{ID: "stop-1", ArtworkID: "aw0000000000001", Title: "Work One", Narration: "Some note", Position: 0},
			{ID: "stop-2", ArtworkID: "aw0000000000002", Title: "Work Two", Narration: "", Position: 1},
			{ID: "stop-3", ArtworkID: "aw0000000000003", Title: "Work Three", Narration: "Later", Position: 2},
		},
	}

	var output bytes.Buffer
	if err := ItineraryBuilder(view, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render builder: %v", err)
	}

	rendered := output.String()
	// The filmstrip marks the second stop as missing narration.
	if !strings.Contains(rendered, "NO NARRATION YET") {
		t.Error("filmstrip must mark a stop without narration")
	}
	if !strings.Contains(rendered, "9 CHARS") {
		t.Error("filmstrip must show the character count for a narrated stop")
	}
	// The selected editor renders for the selected stop and offers next-stop.
	if !strings.Contains(rendered, "STOP 02 OF 3") {
		t.Error("selected editor must render the stop position")
	}
	if !strings.Contains(rendered, "A STOP WITH NO NARRATION IS SHOWN AS A PLATE ONLY") {
		t.Error("selected editor must explain the missing-narration outcome")
	}
	if !strings.Contains(rendered, "NEXT STOP") {
		t.Error("selected editor must offer next-stop navigation when more stops follow")
	}
	if strings.Count(rendered, "YOUR NARRATION") != 3 {
		t.Error("every stop editor must remain in the publish form for client-side tabs")
	}
	if !strings.Contains(rendered, `data-itinerary-tab="2"`) || !strings.Contains(rendered, `data-itinerary-editor="2"`) {
		t.Error("stop links and editor panels must share client-side tab identifiers")
	}
	// The selected stop carries aria-current in the filmstrip.
	if !strings.Contains(rendered, `aria-current="page"`) {
		t.Error("the selected stop must be marked with aria-current")
	}
}

func TestNarrationStatus(t *testing.T) {
	if got := narrationStatus(""); got != "NO NARRATION YET" {
		t.Errorf("empty narration status = %q", got)
	}
	if got := narrationStatus("A note"); got != "6 CHARS" {
		t.Errorf("narrated status = %q", got)
	}
}

func TestItineraryBuilderMutationFormsDisableInheritedSelect(t *testing.T) {
	view := dto.ItineraryBuilderView{
		CSRF:       "csrf-token",
		Max:        15,
		Count:      1,
		PickerOpen: true,
		Stops: []dto.ItineraryStop{
			{ID: "stop-1", ArtworkID: "aw0000000000001", Title: "Work One", Position: 0},
		},
		Picker: []dto.ItineraryPickerWork{
			{ArtworkID: "aw0000000000002", Title: "Work Two"},
		},
	}

	var output bytes.Buffer
	if err := ItineraryBuilder(view, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render builder: %v", err)
	}

	rendered := output.String()
	for _, form := range []string{
		`hx-post="/itineraries/draft/add" hx-target="#itinerary-tray" hx-swap="outerHTML" hx-select="unset"`,
		`hx-post="/itineraries/draft/clear" hx-target="#itinerary-builder" hx-swap="outerHTML" hx-select="unset"`,
		`hx-post="/itineraries/draft/remove" hx-target="#itinerary-builder" hx-swap="outerHTML" hx-select="unset"`,
		`hx-post="/itineraries/draft/move" hx-target="#itinerary-builder" hx-swap="outerHTML" hx-select="unset"`,
	} {
		if !strings.Contains(rendered, form) {
			t.Errorf("mutation form missing hx-select override: %q", form)
		}
	}
	for _, forbidden := range []string{"/itineraries/draft/meta", "/itineraries/draft/narration", `hx-trigger="input changed delay:600ms"`} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("builder must not autosave through %q", forbidden)
		}
	}
	if !strings.Contains(rendered, `data-itinerary-visibility`) || !strings.Contains(rendered, `border-primary bg-primary text-primary-content`) {
		t.Error("visibility controls must expose their selected state for client-side updates")
	}
	if strings.Count(rendered, `hx-confirm="Discard unfinished title, introduction, maker, and narration?"`) < 4 {
		t.Error("builder replacements must warn before discarding unpublished fields")
	}
}

func TestItineraryBuilderMutationFormsCarryVolatileState(t *testing.T) {
	view := dto.ItineraryBuilderView{
		CSRF:       "csrf-token",
		Max:        15,
		Count:      1,
		Selected:   3,
		PickerOpen: true,
		Query:      "sun",
		Stops: []dto.ItineraryStop{
			{ID: "stop-1", ArtworkID: "aw0000000000001", Title: "Work One", Position: 0},
		},
		Picker: []dto.ItineraryPickerWork{
			{ArtworkID: "aw0000000000002", Title: "Work Two"},
		},
	}

	var output bytes.Buffer
	if err := ItineraryBuilder(view, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render builder: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`name="picker" value="1"`,
		`name="pq" value="sun"`,
		`name="stop" value="3"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("builder state must be carried in mutation forms, missing %q", expected)
		}
	}
}

func TestItineraryBuilderPickerDisclosureClosedByDefault(t *testing.T) {
	view := dto.ItineraryBuilderView{
		CSRF: "csrf-token",
		Max:  15,
		Picker: []dto.ItineraryPickerWork{
			{ArtworkID: "aw0000000000002", Title: "Work Two"},
		},
	}

	var output bytes.Buffer
	if err := ItineraryBuilder(view, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render builder: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "ADD WORKS +") {
		t.Error("closed picker must expose the disclosure link")
	}
	if strings.Contains(rendered, `name="artwork_id"`) {
		t.Error("closed picker must not render picker results")
	}
}

func TestItineraryBuilderPublishDisabledWhenEmpty(t *testing.T) {
	view := dto.ItineraryBuilderView{CSRF: "csrf-token", Max: 15, Count: 0}

	var output bytes.Buffer
	if err := ItineraryBuilder(view, false).Render(context.Background(), &output); err != nil {
		t.Fatalf("render builder: %v", err)
	}

	if !strings.Contains(output.String(), "ADD A WORK FIRST") {
		t.Error("empty builder must disable publish with a hint")
	}
}

func TestItineraryViewRendersSlideshowNavigation(t *testing.T) {
	view := dto.ItineraryView{
		Title:      "My Journey",
		Creator:    "Anon.",
		Total:      3,
		Index:      1,
		StopTitle:  "Work One",
		StopArtist: "Test Artist",
		StopDate:   "1565",
		StopSchool: "Italian",
		Narration:  "A note",
		Plate:      dto.Plate{DisplayURL: "/api/files/artworks/aw0000000000001/img.jpg", ZoomURL: "/api/files/artworks/aw0000000000001/img.jpg?thumb=2000x0", Alt: "Work One", Label: "Work One"},
		HasPrev:    true,
		HasNext:    true,
		PrevURL:    "/itineraries/token?stop=0",
		NextURL:    "/itineraries/token?stop=2",
		StopURLs:   []string{"/itineraries/token?stop=0", "/itineraries/token?stop=1", "/itineraries/token?stop=2"},
		ExitURL:    "/itineraries",
	}

	var output bytes.Buffer
	if err := ItineraryViewPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render view: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`data-itinerary-viewer`,
		"STOP 02 OF 03",
		"Work One",
		"A note",
		`rel="prev"`,
		`rel="next"`,
		`href="/itineraries/token?stop=0"`,
		`href="/itineraries/token?stop=2"`,
		`data-itinerary-prefetch`,
		`hx-target="#itinerary-viewer"`,
		`hx-select="#itinerary-viewer"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="true"`,
		"Test Artist",
		"1565",
		"Italian SCHOOL",
		"NARRATION BY Anon.",
		"USE ← AND → TO MOVE",
		`data-itinerary-exit`,
		`data-viewer`,
		`data-zoom-url`,
		"VISITOR ITINERARY",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("view does not contain %q", expected)
		}
	}

	// The progress strip links every stop directly.
	for _, href := range []string{
		`href="/itineraries/token?stop=0"`,
		`href="/itineraries/token?stop=1"`,
		`href="/itineraries/token?stop=2"`,
	} {
		if !strings.Contains(rendered, href) {
			t.Errorf("progress strip is missing direct stop link %q", href)
		}
	}
}

func TestItineraryViewRendersNoInlineKeyboardListener(t *testing.T) {
	view := dto.ItineraryView{
		Title:     "My Journey",
		Creator:   "Anon.",
		Total:     1,
		Index:     0,
		StopTitle: "Work One",
		Plate:     dto.Plate{DisplayURL: "/img.jpg", Alt: "Work One"},
		ExitURL:   "/itineraries",
	}

	var output bytes.Buffer
	if err := ItineraryViewPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render view: %v", err)
	}

	rendered := output.String()
	// Keyboard navigation lives in the itinerary JS module, bound synchronously
	// from the application entry, so the viewer must not ship its own inline
	// listener (which would double-bind and drift from the maintained code).
	if !strings.Contains(rendered, `data-itinerary-viewer`) {
		t.Fatal("viewer must still expose its data-itinerary-viewer contract")
	}
	for _, forbidden := range []string{
		`addEventListener("keydown"`,
		"handleKeydown",
		"itineraryKeyboard",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("viewer must not render an inline keyboard listener, found %q", forbidden)
		}
	}
}

func TestItineraryViewCarriesZoomURLOnImage(t *testing.T) {
	view := dto.ItineraryView{
		Title:     "My Journey",
		Creator:   "Anon.",
		Total:     1,
		Index:     0,
		StopTitle: "Work One",
		Plate: dto.Plate{
			DisplayURL: "/api/files/artworks/aw0000000000001/img.jpg?thumb=1400x0",
			ZoomURL:    "/api/files/artworks/aw0000000000001/img.jpg?thumb=2000x0",
			Alt:        "Work One",
			Label:      "Work One",
		},
		ExitURL: "/itineraries",
	}

	var output bytes.Buffer
	if err := ItineraryViewPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render view: %v", err)
	}

	rendered := output.String()
	// The plate renders the 1400px display source on the image...
	if !strings.Contains(rendered, `src="/api/files/artworks/aw0000000000001/img.jpg?thumb=1400x0"`) {
		t.Error("the image must render the 1400px display source")
	}
	// ...carries the deliberate 2000px zoom source on the same <img> (the
	// shared ViewerJS resolver reads it there, not on the anchor)...
	if !strings.Contains(rendered, `loading="lazy" data-zoom-url="/api/files/artworks/aw0000000000001/img.jpg?thumb=2000x0"`) {
		t.Error("slideshow image must carry data-zoom-url for the deliberate viewer")
	}
	// ...and keeps the anchor's ordinary 2000px href as the no-JS fallback.
	if !strings.Contains(rendered, `href="/api/files/artworks/aw0000000000001/img.jpg?thumb=2000x0"`) {
		t.Error("the anchor must keep the 2000px zoom href as the no-JS fallback")
	}
	if strings.Contains(rendered, `data-viewer data-zoom-url`) {
		t.Error("the zoom URL must not sit on the anchor; ViewerJS reads the image")
	}
	if !strings.Contains(rendered, `data-viewer data-viewer-label`) {
		t.Error("the anchor must keep the data-viewer and label contract")
	}
}

func TestItineraryViewRendersProgressDirectLinksAndGuidance(t *testing.T) {
	view := dto.ItineraryView{
		Title:      "Journey",
		Creator:    "Maker",
		Total:      2,
		Index:      1,
		StopTitle:  "Second",
		StopArtist: "Artist",
		Narration:  "",
		Plate:      dto.Plate{DisplayURL: "/img.jpg", Alt: "Second"},
		HasPrev:    true,
		PrevURL:    "/itineraries/token?stop=0",
		StopURLs:   []string{"/itineraries/token?stop=0", "/itineraries/token?stop=1"},
		ExitURL:    "/itineraries",
	}

	var output bytes.Buffer
	if err := ItineraryViewPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render view: %v", err)
	}

	rendered := output.String()
	// The progress strip exposes one direct link per stop with a truthful label.
	if !strings.Contains(rendered, `aria-label="Go to stop 1"`) || !strings.Contains(rendered, `aria-label="Go to stop 2"`) {
		t.Error("progress strip must label each direct stop link")
	}
	if !strings.Contains(rendered, "END OF ITINERARY") {
		t.Error("the last stop must render the end-of-itinerary label instead of a next link")
	}
}

func TestItineraryViewRendersHonestMissingNarrationCopy(t *testing.T) {
	view := dto.ItineraryView{
		Title:     "My Journey",
		Creator:   "Anon.",
		Total:     1,
		Index:     0,
		StopTitle: "Work One",
		Narration: "",
		Plate:     dto.Plate{DisplayURL: "/img.jpg", Alt: "Work One"},
	}

	var output bytes.Buffer
	if err := ItineraryViewPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render view: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "The maker left this plate without narration") {
		t.Error("view must render the honest missing-narration copy when there is no narration")
	}
}

func TestItineraryIndexRendersEntriesAndEmptyState(t *testing.T) {
	empty := dto.ItineraryIndexView{Itineraries: []dto.ItinerarySummary{}}
	var emptyOut bytes.Buffer
	if err := ItinerariesPage(empty).Render(context.Background(), &emptyOut); err != nil {
		t.Fatalf("render empty index: %v", err)
	}
	if !strings.Contains(emptyOut.String(), "No itineraries are listed yet.") {
		t.Error("empty index must render the empty state")
	}

	populated := dto.ItineraryIndexView{
		Total: 1,
		Itineraries: []dto.ItinerarySummary{
			{Title: "A Journey", Creator: "Anon.", URL: "/itineraries/token", Note: "A short note", Count: 2, Duration: "3 MIN", Published: "01 AUG 2026"},
		},
	}
	var out bytes.Buffer
	if err := ItinerariesPage(populated).Render(context.Background(), &out); err != nil {
		t.Fatalf("render index: %v", err)
	}
	rendered := out.String()
	for _, expected := range []string{
		"A Journey",
		`href="/itineraries/token"`,
		"A short note",
		"3 MIN",
		"01 AUG 2026",
		"1 LISTED PUBLICLY",
		"BUILD AN ITINERARY",
		"Fifteen works at most",
		"Up to 600 characters",
		"One year from the day",
		"Guided Tours",
		`data-kbd-list`,
		`data-kbd-idx="0"`,
		"UNLISTED ITINERARIES",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("populated index is missing %q", expected)
		}
	}
}

func TestItineraryPublishedRendersCopyMakerAndVisibility(t *testing.T) {
	listed := dto.ItineraryPublishedView{
		Title:     "My Journey",
		URL:       "https://wga.example/itineraries/token",
		TokenURL:  "/itineraries/token",
		Creator:   "Anon.",
		Expires:   "2027-08-01",
		Published: "01 AUG 2026",
		Duration:  "3 MIN",
		StopCount: 2,
		Listed:    true,
	}

	var listedOut bytes.Buffer
	if err := ItineraryPublishedPage(listed).Render(context.Background(), &listedOut); err != nil {
		t.Fatalf("render published page: %v", err)
	}
	listedRendered := listedOut.String()
	for _, expected := range []string{
		"https://wga.example/itineraries/token",
		`data-copy-itinerary`,
		`data-copy-target="#itinerary-url"`,
		"COPY LINK",
		">MAKER<",
		">Anon.<",
		">VISIBILITY<",
		">Listed publicly<",
		">2 of 15<",
		">3 MIN<",
		">AVAILABLE UNTIL<",
	} {
		if !strings.Contains(listedRendered, expected) {
			t.Errorf("listed published page is missing %q", expected)
		}
	}

	linkOnly := listed
	linkOnly.Listed = false
	var linkOut bytes.Buffer
	if err := ItineraryPublishedPage(linkOnly).Render(context.Background(), &linkOut); err != nil {
		t.Fatalf("render link-only published page: %v", err)
	}
	if !strings.Contains(linkOut.String(), ">Link only<") {
		t.Error("link-only published page must state the link-only visibility")
	}
}

func TestItineraryExpiredPageRendersGone(t *testing.T) {
	var output bytes.Buffer
	if err := ItineraryExpiredPage().Render(context.Background(), &output); err != nil {
		t.Fatalf("render expired page: %v", err)
	}
	if !strings.Contains(output.String(), "This itinerary has expired") {
		t.Error("expired page must explain the expiry")
	}
}

func TestItineraryRendersEscapedNarrationAndTitle(t *testing.T) {
	view := dto.ItineraryView{
		Title:     `A <script>alert("x")</script> journey`,
		Creator:   `Anon & <b>Co</b>`,
		StopTitle: `<img src=x onerror=alert(1)>`,
		Narration: `<b>bold</b> & <i>italic</i> <script>alert(2)</script>`,
		Total:     1,
		Index:     0,
		Plate:     dto.Plate{DisplayURL: "/img.jpg", Alt: "alt"},
	}

	var output bytes.Buffer
	if err := ItineraryViewPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render view: %v", err)
	}

	rendered := output.String()
	// The full raw malicious fragments must never appear unescaped.
	for _, raw := range []string{
		`<script>alert("x")</script>`,
		`<img src=x onerror=alert(1)>`,
		`<b>bold</b>`,
		`<i>italic</i>`,
		`<b>Co</b>`,
	} {
		if strings.Contains(rendered, raw) {
			t.Errorf("rendered output must escape user content, found raw %q", raw)
		}
	}
	// The escaped script tag survives.
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Error("rendered output must contain the escaped script tag")
	}
}
