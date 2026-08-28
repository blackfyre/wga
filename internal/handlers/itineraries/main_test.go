package itineraries

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const testArtworkID = "aw0000000000001"

const (
	testProductionCookieName  = itineraryworkflow.ProductionSessionCookieName
	testDevelopmentCookieName = "wga_itinerary_dev"
	testClientIdentity        = "198.51.100.7"
)

func TestBuilderSetsSessionCookieAndRenders(t *testing.T) {
	app, mux := newItineraryMux(t)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/itineraries/new", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Build an itinerary") {
		t.Error("response does not render the builder")
	}

	cookie := findSessionCookie(t, response)
	if cookie == nil {
		t.Fatal("builder must set the session cookie")
	}
	if !cookie.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}

	_ = app
}

func TestMutationRejectsMissingCSRF(t *testing.T) {
	_, mux := newItineraryMux(t)

	// A session is issued, but no synchroniser token is submitted.
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/add", strings.NewReader("artwork_id="+testArtworkID))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a mutation without CSRF", response.Code)
	}
}

func TestMutationRejectsCrossOrigin(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	form := url.Values{"artwork_id": {testArtworkID}, "_csrf": {csrf}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/add", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://evil.example")
	request.AddCookie(cookie)
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a cross-origin mutation", response.Code)
	}
}

func TestMutationRejectsMismatchedHost(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	form := url.Values{"artwork_id": {testArtworkID}, "_csrf": {csrf}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://evil.example/itineraries/draft/add", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookie)
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a mismatched Host", response.Code)
	}
}

func TestAddStopAndReloadRecovery(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	// The stop is server-persisted and restored across a reload.
	stops := stopsForCookie(t, app, cookie)
	if len(stops) != 1 {
		t.Fatalf("persisted stops = %d, want 1", len(stops))
	}
	if stops[0].GetInt("position") != 0 {
		t.Errorf("position = %d, want 0", stops[0].GetInt("position"))
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	request.AddCookie(cookie)
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if strings.Contains(response.Body.String(), "No stops yet") {
		t.Error("builder must restore the persisted stop after reload")
	}
}

func TestHtmxMutationsReturnFragmentAndOobTargets(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	// Add returns the tray as the primary target and the builder block OOB.
	add := hxPostForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	if add.Code != http.StatusOK {
		t.Fatalf("add status = %d, want 200", add.Code)
	}
	body := add.Body.String()
	if !strings.Contains(body, `<div id="itinerary-tray"`) {
		t.Error("add response must target the tray as the primary swap")
	}
	if strings.Contains(body, `id="itinerary-tray" hx-swap-oob`) {
		t.Error("add tray must not be OOB; it is the primary target")
	}
	if !strings.Contains(body, `<section id="itinerary-builder" hx-swap-oob="true"`) {
		t.Error("add response must refresh the builder block OOB")
	}

	// Clear returns the builder block as the primary target and an empty tray OOB.
	clear := hxPostForm(t, mux, "/itineraries/draft/clear", cookie, csrf, url.Values{})
	if clear.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want 200", clear.Code)
	}
	body = clear.Body.String()
	if !strings.Contains(body, `<section id="itinerary-builder"`) {
		t.Error("clear response must target the builder as the primary swap")
	}
	if strings.Contains(body, `id="itinerary-builder" hx-swap-oob`) {
		t.Error("clear builder must not be OOB; it is the primary target")
	}
	if !strings.Contains(body, `<div id="itinerary-tray" hx-swap-oob="true"`) {
		t.Error("clear response must empty the tray OOB")
	}
}

func TestConsecutiveDistinctAddsThroughRenderedBuilder(t *testing.T) {
	app, mux := newItineraryMux(t)
	createArtworks(t, app, 2, 3)

	cookie, csrf := sessionForMux(t, mux)

	ids := []string{"aw0000000000001", "aw0000000000002", "aw0000000000003"}
	for index, artworkID := range ids {
		add := hxPostForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {artworkID}})
		if add.Code != http.StatusOK {
			t.Fatalf("add %d status = %d, want 200: %s", index+1, add.Code, add.Body.String())
		}
		if !strings.Contains(add.Body.String(), `id="itinerary-builder" hx-swap-oob="true"`) {
			t.Fatalf("add %d response must refresh the builder OOB", index+1)
		}

		// The OOB builder swap is what the browser installs next, so it must
		// carry a valid synchroniser token for the following mutation.
		next := csrfFromBuilder(t, add.Body.String())
		if next != itineraryworkflow.CSRFToken(cookie.Value) {
			t.Fatalf("add %d OOB builder CSRF = %q, want %q", index+1, next, itineraryworkflow.CSRFToken(cookie.Value))
		}
		csrf = next
	}

	stops := stopsForCookie(t, app, cookie)
	if len(stops) != len(ids) {
		t.Fatalf("persisted stops = %d, want %d", len(stops), len(ids))
	}
	for index, stop := range stops {
		if stop.GetInt("position") != index {
			t.Errorf("stop %d position = %d, want %d", index, stop.GetInt("position"), index)
		}
	}
}

func TestAddStopEnforcesLimit(t *testing.T) {
	app, mux := newItineraryMux(t)
	createArtworks(t, app, 2, 16)

	cookie, csrf := sessionForMux(t, mux)

	for index := 1; index <= itineraryworkflow.MaxStops; index++ {
		postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {fmt.Sprintf("aw%013d", index)}})
	}

	response := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {fmt.Sprintf("aw%013d", 16)}})
	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when exceeding the stop limit", response.Code)
	}
}

func TestOwnershipIsolation(t *testing.T) {
	app, mux := newItineraryMux(t)

	firstCookie, firstCSRF := sessionForMux(t, mux)
	secondCookie, secondCSRF := sessionForMux(t, mux)

	postForm(t, mux, "/itineraries/draft/add", firstCookie, firstCSRF, url.Values{"artwork_id": {testArtworkID}})

	firstStops := stopsForCookie(t, app, firstCookie)
	if len(firstStops) != 1 {
		t.Fatalf("first session stops = %d, want 1", len(firstStops))
	}

	// The second session cannot mutate the first session's stop.
	response := postForm(t, mux, "/itineraries/draft/remove", secondCookie, secondCSRF, url.Values{"stop_id": {firstStops[0].Id}})
	if response.Code != http.StatusBadRequest {
		t.Errorf("cross-session removal status = %d, want 400", response.Code)
	}

	if len(stopsForCookie(t, app, firstCookie)) != 1 {
		t.Error("a different session must not be able to modify another session's draft")
	}
}

func TestPublishFlow(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"My Journey"}})

	response := postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("publish status = %d, want 303", response.Code)
	}
	if got := response.Header().Get("Location"); got != "/itineraries/published" {
		t.Errorf("Location = %q, want /itineraries/published", got)
	}

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil {
		t.Fatalf("find published: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("published records = %d, want 1", len(records))
	}
	if records[0].GetString("token") == "" {
		t.Error("publish must issue an immutable token")
	}

	// The token is immediately readable cookie-less.
	token := records[0].GetString("token")
	view := getPath(t, mux, "/itineraries/"+token, nil)
	if view.Code != http.StatusOK {
		t.Fatalf("published token view status = %d, want 200 (immediate publication)", view.Code)
	}
}

func TestPublishHtmxReturnsRedirectHeader(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"My Journey"}})

	// An HTMX publish must become a real page navigation to the session-owned
	// confirmation, not an in-place body swap that leaves the URL on the builder.
	response := hxPostForm(t, mux, "/itineraries", cookie, csrf, url.Values{})
	if response.Code != http.StatusNoContent {
		t.Fatalf("htmx publish status = %d, want 204", response.Code)
	}
	if got := response.Header().Get("HX-Redirect"); got != "/itineraries/published" {
		t.Errorf("HX-Redirect = %q, want /itineraries/published", got)
	}

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("find published: %v %d", err, len(records))
	}
	if records[0].GetString("token") == "" {
		t.Error("htmx publish must still issue an immutable token")
	}
}

func TestImmediatePublicationTokenReadableCookieLess(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"My Journey"}})
	postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("find published: %v %d", err, len(records))
	}
	token := records[0].GetString("token")

	// No cookie: the public token must render without any session.
	response := getPath(t, mux, "/itineraries/"+token, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("cookie-less published view status = %d, want 200", response.Code)
	}
}

func TestListedAndLinkOnlyVisibility(t *testing.T) {
	app, mux := newItineraryMux(t)

	// Listed publication appears on the public index.
	listedCookie, listedCSRF := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", listedCookie, listedCSRF, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", listedCookie, listedCSRF, url.Values{"title": {"Listed Journey"}})
	postForm(t, mux, "/itineraries", listedCookie, listedCSRF, url.Values{"listed": {"1"}})

	// Link-only publication is readable but absent from the index.
	linkCookie, linkCSRF := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", linkCookie, linkCSRF, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", linkCookie, linkCSRF, url.Values{"title": {"Link Only Journey"}})
	postForm(t, mux, "/itineraries", linkCookie, linkCSRF, url.Values{"listed": {"0"}})

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 2 {
		t.Fatalf("find published: %v %d", err, len(records))
	}
	var listedToken, linkToken string
	for _, record := range records {
		if record.GetString("title") == "Listed Journey" {
			listedToken = record.GetString("token")
		} else {
			linkToken = record.GetString("token")
		}
	}

	index := getPath(t, mux, "/itineraries", nil)
	if index.Code != http.StatusOK {
		t.Fatalf("index status = %d, want 200", index.Code)
	}
	if !strings.Contains(index.Body.String(), "Listed Journey") {
		t.Error("listed itinerary must appear on the public index")
	}
	if strings.Contains(index.Body.String(), "Link Only Journey") {
		t.Error("link-only itinerary must stay off the public index")
	}

	// Both tokens are readable.
	if got := getPath(t, mux, "/itineraries/"+listedToken, nil).Code; got != http.StatusOK {
		t.Errorf("listed token view = %d, want 200", got)
	}
	if got := getPath(t, mux, "/itineraries/"+linkToken, nil).Code; got != http.StatusOK {
		t.Errorf("link-only token view = %d, want 200", got)
	}
}

func TestPublicViewRequiresApprovalAndReturnsGoneWhenExpired(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"My Journey"}})
	postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("find approved record: %v %d", err, len(records))
	}
	token := records[0].GetString("token")
	path := "/itineraries/" + token

	// Immediately approved: the public view renders.
	response := getPath(t, mux, path, nil)
	if response.Code != http.StatusOK {
		t.Errorf("published view status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Work 1") {
		t.Error("public view must render the artwork title")
	}
	if !strings.Contains(response.Body.String(), `data-itinerary-viewer`) {
		t.Error("public view must render the slideshow viewer with keyboard data")
	}
	if !strings.Contains(response.Body.String(), "STOP 01 OF 01") {
		t.Error("public view must render the stop counter")
	}

	// Rejected is denied.
	records[0].Set("status", "rejected")
	if err := app.Save(records[0]); err != nil {
		t.Fatalf("reject: %v", err)
	}
	response = getPath(t, mux, path, nil)
	if response.Code != http.StatusNotFound {
		t.Errorf("rejected view status = %d, want 404", response.Code)
	}
	records[0].Set("status", "approved")
	if err := app.Save(records[0]); err != nil {
		t.Fatalf("re-approve: %v", err)
	}

	// Expired returns 410. Age the record via raw SQL because the lifecycle
	// hook forbids changing publication fields on a non-draft record.
	if _, err := app.DB().NewQuery(
		"UPDATE itineraries SET expires_at = '2000-01-01 00:00:00.000Z' WHERE id = {:id}",
	).Bind(map[string]any{"id": records[0].Id}).Execute(); err != nil {
		t.Fatalf("expire: %v", err)
	}
	response = getPath(t, mux, path, nil)
	if response.Code != http.StatusGone {
		t.Errorf("expired view status = %d, want 410", response.Code)
	}
}

func TestPublicViewRejectedReturnsNotFound(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"Journey"}})
	postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("find approved record: %v %d", err, len(records))
	}
	token := records[0].GetString("token")

	records[0].Set("status", "rejected")
	if err := app.Save(records[0]); err != nil {
		t.Fatalf("reject: %v", err)
	}

	response := getPath(t, mux, "/itineraries/"+token, nil)
	if response.Code != http.StatusNotFound {
		t.Errorf("rejected view status = %d, want 404", response.Code)
	}
}

func TestPublicView404WhenNoAvailableStops(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"Journey"}})
	postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("find approved record: %v %d", err, len(records))
	}
	token := records[0].GetString("token")

	// Unpublish the only artwork, so no stops remain available.
	artwork, err := app.FindRecordById("artworks", testArtworkID)
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("unpublish artwork: %v", err)
	}

	response := getPath(t, mux, "/itineraries/"+token, nil)
	if response.Code != http.StatusNotFound {
		t.Errorf("view with no available stops status = %d, want 404", response.Code)
	}
}

func TestPublishRequiresTitleAndStop(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	response := postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})
	if response.Code != http.StatusBadRequest {
		t.Errorf("publish without title/stop status = %d, want 400", response.Code)
	}
}

func TestAddPreservesPickerStateThroughOobBuilder(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	// An add performed while the picker is open and filtered must preserve the
	// disclosure, query, and selected stop in the out-of-band builder refresh.
	add := hxPostForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{
		"artwork_id": {testArtworkID},
		"picker":     {"1"},
		"pq":         {"sun"},
		"stop":       {"0"},
	})
	if add.Code != http.StatusOK {
		t.Fatalf("add status = %d, want 200: %s", add.Code, add.Body.String())
	}
	body := add.Body.String()
	if !strings.Contains(body, `id="itinerary-builder" hx-swap-oob="true"`) {
		t.Error("add response must refresh the builder OOB")
	}
	if !strings.Contains(body, `name="picker" value="1"`) {
		t.Error("add response must preserve the open picker disclosure")
	}
	if !strings.Contains(body, `name="pq" value="sun"`) {
		t.Error("add response must preserve the picker query")
	}
}

func TestMutationPreservesVolatileBuilderState(t *testing.T) {
	app, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	stopID := stopsForCookie(t, app, cookie)[0].Id

	// A narration save submitted with picker/selected state must render the
	// builder back with the same state embedded for the next mutation.
	narration := hxPostForm(t, mux, "/itineraries/draft/narration", cookie, csrf, url.Values{
		"stop_id":   {stopID},
		"narration": {"A note"},
		"picker":    {"1"},
		"pq":        {"sun"},
		"stop":      {"0"},
	})
	if narration.Code != http.StatusOK {
		t.Fatalf("narration status = %d, want 200", narration.Code)
	}
	if !strings.Contains(narration.Body.String(), `name="picker" value="1"`) {
		t.Error("narration response must preserve the picker disclosure")
	}
	if !strings.Contains(narration.Body.String(), `name="pq" value="sun"`) {
		t.Error("narration response must preserve the picker query")
	}
}

func TestCookieLessGETsCreateNoRecords(t *testing.T) {
	app, mux := newItineraryMux(t)

	for index := 0; index < 3; index++ {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/itineraries/new", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET status = %d, want 200", response.Code)
		}
	}

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "", "", 0, 0)
	if err != nil {
		t.Fatalf("find records: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("cookie-less GETs allocated %d records, want 0", len(records))
	}
}

func TestSessionCookieRotationStrictDuplicateLegacy(t *testing.T) {
	_, mux := newItineraryMux(t)

	first, _ := itineraryworkflow.NewToken()
	second, _ := itineraryworkflow.NewToken()

	// Strict: an invalid token is rotated to a fresh valid token.
	rotated := rotateSessionForCookie(t, mux, &http.Cookie{Name: testDevelopmentCookieName, Value: "invalid-token"})
	if rotated == "" || !itineraryworkflow.ValidToken(rotated) || rotated == "invalid-token" {
		t.Errorf("invalid token must be rotated to a fresh valid token, got %q", rotated)
	}

	// Duplicate: two same-name cookies are treated as absent.
	dup := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: first})
	request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: second})
	mux.ServeHTTP(dup, request)
	dupCookie := findSessionCookie(t, dup)
	if dupCookie == nil || !itineraryworkflow.ValidToken(dupCookie.Value) {
		t.Error("duplicate same-name cookies must yield a fresh valid token")
	}

	// Legacy: the pre-remediation cookie name is ignored.
	legacy := httptest.NewRecorder()
	legacyRequest := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	legacyRequest.AddCookie(&http.Cookie{Name: itineraryworkflow.LegacySessionCookieName, Value: first})
	mux.ServeHTTP(legacy, legacyRequest)
	legacyCookie := findSessionCookie(t, legacy)
	if legacyCookie == nil || !itineraryworkflow.ValidToken(legacyCookie.Value) {
		t.Error("a legacy cookie must be ignored and a fresh token issued")
	}
}

func TestSessionCookieExactFlagsAndNames(t *testing.T) {
	_, mux := newItineraryMux(t)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/itineraries/new", nil))

	cookie := findSessionCookie(t, response)
	if cookie == nil {
		t.Fatal("no session cookie issued")
	}
	if cookie.Name != testDevelopmentCookieName {
		t.Errorf("cookie name = %q, want %q (development policy)", cookie.Name, testDevelopmentCookieName)
	}
	if !cookie.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
	if cookie.Domain != "" {
		t.Error("cookie must be host-only (no Domain)")
	}
	if cookie.Secure {
		t.Error("development cookie must not be Secure")
	}
}

func TestSecureCookiesMarkSessionCookieAndRequireHTTPS(t *testing.T) {
	_, mux := newItineraryMuxSecure(t, true)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/itineraries/new", nil))

	cookie := findSessionCookie(t, response)
	if cookie == nil {
		t.Fatal("builder must set the session cookie")
	}
	if cookie.Name != itineraryworkflow.ProductionSessionCookieName {
		t.Errorf("cookie name = %q, want %q", cookie.Name, itineraryworkflow.ProductionSessionCookieName)
	}
	if !cookie.Secure {
		t.Error("secure registration must mark the session cookie Secure")
	}
	if cookie.Domain != "" {
		t.Error("production cookie must be host-only (no Domain)")
	}

	// A mutation with an http Origin must be rejected in secure mode.
	form := url.Values{"artwork_id": {testArtworkID}, "_csrf": {itineraryworkflow.CSRFToken(cookie.Value)}}
	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/add", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.com")
	request.AddCookie(cookie)
	rejected := httptest.NewRecorder()
	mux.ServeHTTP(rejected, request)
	if rejected.Code != http.StatusBadRequest {
		t.Errorf("http Origin in secure mode status = %d, want 400", rejected.Code)
	}
}

func TestMissingTrustedIdentityFailsClosed(t *testing.T) {
	_, mux := newItineraryMuxWithResolver(t, func(*http.Request) (string, bool) {
		return "", false
	})

	cookie, csrf := sessionForMux(t, mux)

	// State creation and publication fail closed without a trusted identity.
	for _, path := range []string{"/itineraries/draft/meta", "/itineraries/draft/add", "/itineraries"} {
		response := postForm(t, mux, path, cookie, csrf, url.Values{"artwork_id": {testArtworkID}, "title": {"Journey"}})
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400 (fail closed)", path, response.Code)
		}
	}
}

func TestDraftLimitCannotBeBypassedByCookieRotation(t *testing.T) {
	_, mux := newItineraryMux(t)

	// Three distinct cookies (owners) share one trusted identity.
	for index := 0; index < 3; index++ {
		cookie, csrf := sessionForMux(t, mux)
		response := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("draft %d status = %d, want 303", index, response.Code)
		}
	}

	// A fourth cookie for the same identity is denied.
	cookie, csrf := sessionForMux(t, mux)
	response := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	if response.Code != http.StatusTooManyRequests {
		t.Errorf("fourth draft status = %d, want 429", response.Code)
	}
}

func TestExistingDraftMutationsDoNotConsumeBudget(t *testing.T) {
	_, mux := newItineraryMux(t)

	// First draft (charges one creation slot).
	cookie, csrf := sessionForMux(t, mux)
	if response := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}}); response.Code != http.StatusSeeOther {
		t.Fatalf("first add status = %d, want 303", response.Code)
	}

	// Many mutations of the existing draft must not consume creation budget.
	for index := 0; index < 5; index++ {
		postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {fmt.Sprintf("Journey %d", index)}})
		postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	}

	// Two more drafts for the same identity are allowed.
	for index := 0; index < 2; index++ {
		other, otherCSRF := sessionForMux(t, mux)
		if response := postForm(t, mux, "/itineraries/draft/add", other, otherCSRF, url.Values{"artwork_id": {testArtworkID}}); response.Code != http.StatusSeeOther {
			t.Fatalf("additional draft %d status = %d, want 303", index, response.Code)
		}
	}

	// The budget is spent: a fourth distinct draft is denied.
	fourth, fourthCSRF := sessionForMux(t, mux)
	response := postForm(t, mux, "/itineraries/draft/add", fourth, fourthCSRF, url.Values{"artwork_id": {testArtworkID}})
	if response.Code != http.StatusTooManyRequests {
		t.Errorf("fourth draft status = %d, want 429", response.Code)
	}
}

func TestConcurrentPublishesRespectClientLimit(t *testing.T) {
	app, mux := newItineraryMux(t)

	const workers = 6
	cookies := make([]*http.Cookie, workers)
	csrfs := make([]string, workers)

	// Seed one ready draft per owner via the workflow directly, bypassing the
	// handler's draft admission so only the publication budget is exercised.
	for index := 0; index < workers; index++ {
		token, err := itineraryworkflow.NewToken()
		if err != nil {
			t.Fatalf("NewToken: %v", err)
		}
		owner := itineraryworkflow.OwnerDigest(token)
		if _, err := itineraryworkflow.AddStop(app, owner, testArtworkID); err != nil {
			t.Fatalf("seed stop: %v", err)
		}
		if err := itineraryworkflow.SetMeta(app, owner, itineraryworkflow.Meta{Title: "Journey"}); err != nil {
			t.Fatalf("seed meta: %v", err)
		}
		cookies[index] = &http.Cookie{Name: testDevelopmentCookieName, Value: token}
		csrfs[index] = itineraryworkflow.CSRFToken(token)
	}

	codes := make([]int, workers)
	var wg sync.WaitGroup
	for index := 0; index < workers; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			codes[index] = postForm(t, mux, "/itineraries", cookies[index], csrfs[index], url.Values{}).Code
		}(index)
	}
	wg.Wait()

	successes := 0
	for _, code := range codes {
		if code == http.StatusSeeOther {
			successes++
		}
	}
	if successes > 3 {
		t.Errorf("successful publishes = %d, want at most 3", successes)
	}
}

func TestPublicResponsesAreNoStore(t *testing.T) {
	app, mux := newItineraryMux(t)

	// Public index 200.
	assertCacheControl(t, getPath(t, mux, "/itineraries", nil), "private, no-store")

	// Publish so the slideshow 200 path is reachable.
	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"Journey"}})
	postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})

	records, err := app.FindRecordsByFilter(itineraryworkflow.CollectionItineraries, "status = 'approved'", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("find approved: %v %d", err, len(records))
	}
	token := records[0].GetString("token")

	// Slideshow 200.
	assertCacheControl(t, getPath(t, mux, "/itineraries/"+token, nil), "private, no-store")

	// Slideshow 404.
	missing := getPath(t, mux, "/itineraries/unknown-token", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing view status = %d, want 404", missing.Code)
	}
	assertCacheControl(t, missing, "private, no-store")

	// Slideshow 410.
	if _, err := app.DB().NewQuery(
		"UPDATE itineraries SET expires_at = '2000-01-01 00:00:00.000Z' WHERE id = {:id}",
	).Bind(map[string]any{"id": records[0].Id}).Execute(); err != nil {
		t.Fatalf("expire: %v", err)
	}
	gone := getPath(t, mux, "/itineraries/"+token, nil)
	if gone.Code != http.StatusGone {
		t.Fatalf("expired view status = %d, want 410", gone.Code)
	}
	assertCacheControl(t, gone, "private, no-store")
}

func TestSessionOwnedResponsesAreNoStore(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	// Builder page and block.
	assertCacheControl(t, getPath(t, mux, "/itineraries/new", cookie), "private, no-store")
	assertCacheControl(t, getPath(t, mux, "/itineraries/draft", cookie), "private, no-store")

	// Publication receipt.
	assertCacheControl(t, getPath(t, mux, "/itineraries/published", cookie), "private, no-store")

	// Mutations (HTMX and redirect paths).
	add := postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	assertCacheControl(t, add, "private, no-store")

	meta := postForm(t, mux, "/itineraries/draft/meta", cookie, csrf, url.Values{"title": {"Journey"}})
	assertCacheControl(t, meta, "private, no-store")

	published := postForm(t, mux, "/itineraries", cookie, csrf, url.Values{})
	assertCacheControl(t, published, "private, no-store")
}

func TestForbiddenMutationIsNoStore(t *testing.T) {
	_, mux := newItineraryMux(t)

	cookie, csrf := sessionForMux(t, mux)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/add", strings.NewReader("artwork_id="+testArtworkID+"&_csrf="+csrf))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://evil.example")
	request.AddCookie(cookie)
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	assertCacheControl(t, response, "private, no-store")
}

func TestRegisterHandlersRejectsInvalidPolicy(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset app: %v", err)
		}
	})

	cases := []struct {
		name   string
		policy SecurityPolicy
	}{
		{"empty canonical origin", SecurityPolicy{Production: CookiePolicy{Name: testProductionCookieName, Secure: true}, TrustedClientID: testClientID}},
		{"bad production name", SecurityPolicy{CanonicalOrigin: "https://example.com", Production: CookiePolicy{Name: "wga_itinerary", Secure: true}, TrustedClientID: testClientID}},
		{"production not secure", SecurityPolicy{CanonicalOrigin: "https://example.com", Production: CookiePolicy{Name: testProductionCookieName, Secure: false}, TrustedClientID: testClientID}},
		{"http without development policy", SecurityPolicy{CanonicalOrigin: "http://example.com", Production: CookiePolicy{Name: testProductionCookieName, Secure: true}, TrustedClientID: testClientID}},
		{"development name collides with production", SecurityPolicy{CanonicalOrigin: "http://example.com", Production: CookiePolicy{Name: testProductionCookieName, Secure: true}, Development: CookiePolicy{Name: testProductionCookieName, Secure: false}, TrustedClientID: testClientID}},
		{"missing trusted client resolver", SecurityPolicy{CanonicalOrigin: "https://example.com", Production: CookiePolicy{Name: testProductionCookieName, Secure: true}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := RegisterHandlers(app, tc.policy); err == nil {
				t.Error("RegisterHandlers must reject the invalid policy")
			}
		})
	}
}

func assertCacheControl(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	if got := response.Header().Get("Cache-Control"); got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

// --- helpers ---

func newItineraryMux(t *testing.T) (*pocketbase.PocketBase, http.Handler) {
	t.Helper()
	return newItineraryMuxSecure(t, false)
}

func newItineraryMuxSecure(t *testing.T, secure bool) (*pocketbase.PocketBase, http.Handler) {
	t.Helper()
	return newItineraryMuxWithPolicy(t, testSecurityPolicy(secure))
}

func newItineraryMuxWithResolver(t *testing.T, resolver TrustedClientID) (*pocketbase.PocketBase, http.Handler) {
	t.Helper()
	policy := testSecurityPolicy(false)
	policy.TrustedClientID = resolver
	return newItineraryMuxWithPolicy(t, policy)
}

func newItineraryMuxWithPolicy(t *testing.T, policy SecurityPolicy) (*pocketbase.PocketBase, http.Handler) {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset app: %v", err)
		}
	})

	installSchema(t, app)
	if err := RegisterHandlers(app, policy); err != nil {
		t.Fatalf("register handlers: %v", err)
	}

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		built, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		mux = built
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}

	return app, mux
}

func testSecurityPolicy(secure bool) SecurityPolicy {
	if secure {
		return SecurityPolicy{
			CanonicalOrigin: "https://example.com",
			Production:      CookiePolicy{Name: testProductionCookieName, Secure: true},
			TrustedClientID: testClientID,
		}
	}

	return SecurityPolicy{
		CanonicalOrigin: "http://example.com",
		Production:      CookiePolicy{Name: testProductionCookieName, Secure: true},
		Development:     CookiePolicy{Name: testDevelopmentCookieName, Secure: false},
		TrustedClientID: testClientID,
	}
}

func testClientID(*http.Request) (string, bool) {
	return testClientIdentity, true
}

func installSchema(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("create artists collection: %v", err)
	}

	artist := core.NewRecord(artists)
	artist.Id = "ar0000000000001"
	artist.Set("name", "Test Artist")
	artist.Set("slug", "test-artist")
	artist.Set("filing_name", "ARTIST, Test")
	artist.Set("short_name", "Test Artist")
	if err := app.Save(artist); err != nil {
		t.Fatalf("create artist: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
		&core.RelationField{Name: "author", CollectionId: "artists", MaxSelect: 10},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks collection: %v", err)
	}
	createArtworks(t, app, 1, 1)

	itineraries := core.NewBaseCollection("Itineraries")
	itineraries.Id = itineraryworkflow.CollectionItineraries
	itineraries.MarkAsNew()
	itineraries.Fields.Add(
		&core.TextField{Name: "owner", Required: true},
		&core.SelectField{Name: "status", Values: []string{"draft", "pending", "approved", "rejected"}, MaxSelect: 1, Required: true},
		&core.TextField{Name: "token"},
		&core.TextField{Name: "title", Max: 80},
		&core.TextField{Name: "intro", Max: 400},
		&core.TextField{Name: "creator", Max: 40},
		&core.BoolField{Name: "listed"},
		&core.DateField{Name: "published"},
		&core.DateField{Name: "expires_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	itineraries.Indexes = append(itineraries.Indexes,
		"CREATE UNIQUE INDEX `pbx_itinerary_token` ON `itineraries` (token) WHERE token != ''",
		"CREATE UNIQUE INDEX `pbx_itinerary_draft_owner` ON `itineraries` (owner) WHERE status = 'draft'",
	)
	if err := app.Save(itineraries); err != nil {
		t.Fatalf("create itineraries collection: %v", err)
	}

	stops := core.NewBaseCollection("Itinerary_stops")
	stops.Id = itineraryworkflow.CollectionItineraryStops
	stops.MarkAsNew()
	stops.Fields.Add(
		&core.RelationField{Name: "itinerary", CollectionId: itineraryworkflow.CollectionItineraries, MinSelect: 1, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.RelationField{Name: "artwork", CollectionId: "artworks", MinSelect: 1, MaxSelect: 1, Required: true},
		&core.TextField{Name: "title"},
		&core.NumberField{Name: "position"},
		&core.TextField{Name: "narration", Max: 600},
	)
	stops.Indexes = append(stops.Indexes,
		"CREATE UNIQUE INDEX `pbx_itinerary_stop_artwork` ON `itinerary_stops` (itinerary, artwork)",
		"CREATE UNIQUE INDEX `pbx_itinerary_stop_order` ON `itinerary_stops` (itinerary, position)",
	)
	if err := app.Save(stops); err != nil {
		t.Fatalf("create itinerary_stops collection: %v", err)
	}
}

func createArtworks(t *testing.T, app *pocketbase.PocketBase, start int, end int) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	for index := start; index <= end; index++ {
		record := core.NewRecord(collection)
		record.Id = fmt.Sprintf("aw%013d", index)
		record.Set("title", "Work "+fmt.Sprintf("%d", index))
		record.Set("published", true)
		record.Set("author", []string{"ar0000000000001"})
		if err := app.Save(record); err != nil {
			t.Fatalf("create artwork %d: %v", index, err)
		}
	}
}

func sessionForMux(t *testing.T, mux http.Handler) (*http.Cookie, string) {
	t.Helper()
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/itineraries/new", nil))
	cookie := findSessionCookie(t, response)
	if cookie == nil {
		t.Fatal("no session cookie issued")
	}
	return cookie, itineraryworkflow.CSRFToken(cookie.Value)
}

// rotateSessionForCookie issues a GET carrying cookie and returns the session
// token set in response (possibly a rotated fresh token).
func rotateSessionForCookie(t *testing.T, mux http.Handler, cookie *http.Cookie) string {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	request.AddCookie(cookie)
	mux.ServeHTTP(response, request)
	issued := findSessionCookie(t, response)
	if issued == nil {
		return ""
	}
	return issued.Value
}

func postForm(t *testing.T, mux http.Handler, path string, cookie *http.Cookie, csrf string, extra url.Values) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"_csrf": {csrf}}
	for key, values := range extra {
		for _, value := range values {
			form.Add(key, value)
		}
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	mux.ServeHTTP(response, request)
	return response
}

func hxPostForm(t *testing.T, mux http.Handler, path string, cookie *http.Cookie, csrf string, extra url.Values) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"_csrf": {csrf}}
	for key, values := range extra {
		for _, value := range values {
			form.Add(key, value)
		}
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	mux.ServeHTTP(response, request)
	return response
}

func getPath(t *testing.T, mux http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	mux.ServeHTTP(response, request)
	return response
}

func csrfFromBuilder(t *testing.T, body string) string {
	t.Helper()
	const marker = `name="_csrf" value="`
	start := strings.Index(body, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}

func findSessionCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, raw := range response.Result().Cookies() {
		if raw.Name == testProductionCookieName || raw.Name == testDevelopmentCookieName {
			return raw
		}
	}
	return nil
}

func stopsForCookie(t *testing.T, app *pocketbase.PocketBase, cookie *http.Cookie) []*core.Record {
	t.Helper()
	owner := itineraryworkflow.OwnerDigest(cookie.Value)
	draft, err := itineraryworkflow.FindDraft(app, owner)
	if err != nil {
		return nil
	}
	stops, err := itineraryworkflow.LoadStops(app, draft.Id)
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	return stops
}
