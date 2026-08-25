package itineraries

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func newProjectionEvent(app core.App, request *http.Request) *core.RequestEvent {
	return &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
	}
}

func devCookiePolicy() CookiePolicy {
	return CookiePolicy{Name: testDevelopmentCookieName, Secure: false}
}

func TestProjectSessionIssuesCookieAndCSRFWhenAbsent(t *testing.T) {
	app, _ := newItineraryMux(t)

	event := newProjectionEvent(app, httptest.NewRequest(http.MethodGet, "/", nil))
	session, err := ProjectSession(app, event, devCookiePolicy())
	if err != nil {
		t.Fatalf("ProjectSession: %v", err)
	}

	if session.CSRF == "" {
		t.Fatal("first request must carry a synchroniser token")
	}
	if session.Tray.Count != 0 {
		t.Errorf("first request tray count = %d, want 0", session.Tray.Count)
	}
	if session.IsAdded(testArtworkID) {
		t.Error("first request must report no added artworks")
	}

	cookie := findSessionCookie(t, event.Response.(*httptest.ResponseRecorder))
	if cookie == nil {
		t.Fatal("first request must set the session cookie")
	}
	if !itineraryworkflow.ValidToken(cookie.Value) {
		t.Errorf("issued cookie token %q is not valid", cookie.Value)
	}
	if got := itineraryworkflow.CSRFToken(cookie.Value); got != session.CSRF {
		t.Errorf("projection CSRF %q does not match issued token CSRF %q", session.CSRF, got)
	}
	if got := itineraryworkflow.TokenFromRequest(event.Request, testDevelopmentCookieName); got != cookie.Value {
		t.Error("freshly issued token must be visible on the same request for downstream handlers")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Errorf("issued cookie flags = %+v", cookie)
	}
	if cookie.Secure {
		t.Error("development cookie must not be Secure")
	}
}

func TestProjectSessionRotatesInvalidAndDuplicateCookies(t *testing.T) {
	app, _ := newItineraryMux(t)

	// Invalid token is rotated.
	invalid := newProjectionEvent(app, httptest.NewRequest(http.MethodGet, "/", nil))
	invalid.Request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: "not-a-valid-token"})
	if _, err := ProjectSession(app, invalid, devCookiePolicy()); err != nil {
		t.Fatalf("ProjectSession: %v", err)
	}
	issued := findSessionCookie(t, invalid.Response.(*httptest.ResponseRecorder))
	if issued == nil || !itineraryworkflow.ValidToken(issued.Value) || issued.Value == "not-a-valid-token" {
		t.Errorf("invalid token was not rotated to a fresh valid token, got %+v", issued)
	}
	if got := itineraryworkflow.TokenFromRequest(invalid.Request, testDevelopmentCookieName); got != issued.Value {
		t.Errorf("request token = %q, want issued %q", got, issued.Value)
	}

	// Duplicate same-name cookies are treated as absent and rotated.
	first, _ := itineraryworkflow.NewToken()
	second, _ := itineraryworkflow.NewToken()
	dup := newProjectionEvent(app, httptest.NewRequest(http.MethodGet, "/", nil))
	dup.Request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: first})
	dup.Request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: second})
	dup.Request.AddCookie(&http.Cookie{Name: "wga_theme", Value: "dark"})
	if _, err := ProjectSession(app, dup, devCookiePolicy()); err != nil {
		t.Fatalf("ProjectSession: %v", err)
	}
	dupIssued := findSessionCookie(t, dup.Response.(*httptest.ResponseRecorder))
	if dupIssued == nil || !itineraryworkflow.ValidToken(dupIssued.Value) {
		t.Errorf("duplicate cookies were not rotated to a fresh valid token, got %+v", dupIssued)
	}
	if got := itineraryworkflow.TokenFromRequest(dup.Request, testDevelopmentCookieName); got != dupIssued.Value {
		t.Errorf("request token = %q, want issued %q", got, dupIssued.Value)
	}

	// Unrelated cookies are preserved after replacement.
	var themeSeen bool
	for _, cookie := range dup.Request.Cookies() {
		if cookie.Name == "wga_theme" && cookie.Value == "dark" {
			themeSeen = true
		}
	}
	if !themeSeen {
		t.Error("replacement dropped an unrelated cookie")
	}

	// The request carries exactly one itinerary cookie after rotation.
	var itineraryCookies int
	for _, cookie := range dup.Request.Cookies() {
		if cookie.Name == testDevelopmentCookieName {
			itineraryCookies++
		}
	}
	if itineraryCookies != 1 {
		t.Errorf("request itinerary cookies = %d, want exactly 1", itineraryCookies)
	}
}

func TestProjectSessionIgnoresLegacyCookie(t *testing.T) {
	app, _ := newItineraryMux(t)

	token, err := itineraryworkflow.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	event := newProjectionEvent(app, httptest.NewRequest(http.MethodGet, "/", nil))
	event.Request.AddCookie(&http.Cookie{Name: itineraryworkflow.LegacySessionCookieName, Value: token})

	session, err := ProjectSession(app, event, devCookiePolicy())
	if err != nil {
		t.Fatalf("ProjectSession: %v", err)
	}

	issued := findSessionCookie(t, event.Response.(*httptest.ResponseRecorder))
	if issued == nil || issued.Name != testDevelopmentCookieName || issued.Value == token {
		t.Errorf("legacy cookie must be ignored and a fresh token issued, got %+v", issued)
	}
	if got := itineraryworkflow.CSRFToken(issued.Value); got != session.CSRF {
		t.Error("projection CSRF must derive from the freshly issued token")
	}
}

func TestProjectSessionReflectsDraft(t *testing.T) {
	app, mux := newItineraryMux(t)

	artwork, err := app.FindRecordById("artworks", testArtworkID)
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("image", "work.jpg")
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork image: %v", err)
	}

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	event := newProjectionEvent(app, httptest.NewRequest(http.MethodGet, "/", nil))
	event.Request.AddCookie(cookie)

	session, err := ProjectSession(app, event, devCookiePolicy())
	if err != nil {
		t.Fatalf("ProjectSession: %v", err)
	}

	if want := itineraryworkflow.CSRFToken(cookie.Value); session.CSRF != want {
		t.Errorf("CSRF = %q, want %q", session.CSRF, want)
	}
	if session.Tray.Count != 1 {
		t.Errorf("tray count = %d, want 1", session.Tray.Count)
	}
	if !session.IsAdded(testArtworkID) {
		t.Error("projection must report the added artwork")
	}
	if len(session.Tray.Thumbs) != 1 {
		t.Errorf("tray thumbnails = %d, want 1", len(session.Tray.Thumbs))
	}
}
