package itineraries

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
)

// TestAddResponseTrayCarriesClearAction proves the tray returned immediately by
// an HTMX add presents a usable CLEAR action under the same session CSRF
// contract as a full-page projection, not an empty synchroniser token that
// would reject the visitor's next mutation.
func TestAddResponseTrayCarriesClearAction(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	add := hxPostForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	if add.Code != http.StatusOK {
		t.Fatalf("add status = %d, want 200", add.Code)
	}
	body := add.Body.String()

	// The non-empty tray is the primary target, not an out-of-band swap.
	if !strings.Contains(body, `<div id="itinerary-tray"`) {
		t.Fatal("add response must render the tray as the primary target")
	}
	if strings.Contains(body, `id="itinerary-tray" hx-swap-oob`) {
		t.Error("add tray must not be OOB; it is the primary target")
	}

	// The builder link and CLEAR action are the reference tray presentation.
	if !strings.Contains(body, `href="/itineraries/new"`) {
		t.Error("add tray must render the builder link")
	}
	if !strings.Contains(body, `ARRANGE &amp; NARRATE →`) {
		t.Error("add tray must render the ARRANGE & NARRATE action")
	}
	if !strings.Contains(body, `>CLEAR</button>`) {
		t.Error("add tray must render the CLEAR action")
	}

	// The CLEAR form carries both the ordinary POST and HTMX semantics the
	// frontend expects, with the exact session-derived synchroniser token.
	for _, want := range []string{
		`method="post" action="/itineraries/draft/clear"`,
		`hx-post="/itineraries/draft/clear"`,
		`hx-target="#itinerary-tray"`,
		`hx-swap="outerHTML"`,
		`hx-select="unset"`,
		`<input type="hidden" name="_csrf" value="` + itineraryworkflow.CSRFToken(cookie.Value) + `">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("add tray CLEAR form missing %q", want)
		}
	}

	// The OOB builder refresh still targets the builder block.
	if !strings.Contains(body, `<section id="itinerary-builder" hx-swap-oob="true"`) {
		t.Error("add response must refresh the builder block OOB")
	}
}

// TestTrayTargetedClearReturnsTrayPrimary proves the tray's CLEAR action,
// which targets the always-mounted #itinerary-tray, receives the tray-primary
// response shape so clearing from a non-builder page empties the tray instead
// of targeting the absent #itinerary-builder.
func TestTrayTargetedClearReturnsTrayPrimary(t *testing.T) {
	_, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)

	add := hxPostForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	if add.Code != http.StatusOK {
		t.Fatalf("add status = %d, want 200", add.Code)
	}

	form := url.Values{"_csrf": {csrf}}
	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/clear", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "itinerary-tray")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want 200", response.Code)
	}
	body := response.Body.String()

	if !strings.Contains(body, `<div id="itinerary-tray"`) {
		t.Error("tray-targeted clear must render the tray as the primary target")
	}
	if strings.Contains(body, `id="itinerary-tray" hx-swap-oob`) {
		t.Error("tray-targeted clear tray must not be OOB")
	}
	if !strings.Contains(body, `<section id="itinerary-builder" hx-swap-oob="true"`) {
		t.Error("tray-targeted clear must refresh the builder OOB")
	}
	if strings.Contains(body, "ITINERARY DRAFT ·") {
		t.Error("cleared tray must not render the draft bar")
	}
}

func TestOrdinaryAddClearAndDuplicateDoNotCreateExtraStops(t *testing.T) {
	app, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)

	add := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	if add.Code != http.StatusSeeOther || add.Header().Get("Location") != "/itineraries/new" {
		t.Fatalf("ordinary add = %d Location %q, want 303 /itineraries/new", add.Code, add.Header().Get("Location"))
	}
	duplicate := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	if duplicate.Code != http.StatusSeeOther {
		t.Fatalf("duplicate ordinary add = %d, want 303", duplicate.Code)
	}
	if got := len(stopsForCookie(t, app, cookie)); got != 1 {
		t.Fatalf("duplicate add persisted %d stops, want 1", got)
	}

	clear := postForm(t, mux, "/itineraries/draft/clear", cookie, csrf, nil)
	if clear.Code != http.StatusSeeOther || clear.Header().Get("Location") != "/itineraries/new" {
		t.Fatalf("ordinary clear = %d Location %q, want 303 /itineraries/new", clear.Code, clear.Header().Get("Location"))
	}
	if got := len(stopsForCookie(t, app, cookie)); got != 0 {
		t.Fatalf("ordinary clear left %d stops, want 0", got)
	}
}

func TestBuilderGETDoesNotPersistAnonymousDraft(t *testing.T) {
	app, mux := newItineraryMux(t)
	for _, path := range []string{"/itineraries/new", "/itineraries/new?picker=1&pq=work"} {
		response := getPath(t, mux, path, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, response.Code)
		}
	}
	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'draft'", "", 0, 0)
	if err != nil {
		t.Fatalf("find drafts after GET: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("GET created %d draft records, want 0", len(records))
	}
}
