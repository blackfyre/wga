package itineraries

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// newProjectionRouteMux bootstraps a real itineraries app with the projection
// glue applied before RegisterHandlers, then returns the built router. It
// mirrors the central middleware's GET/HEAD projection so the end-to-end route
// exercises ProjectSession against the real /itineraries handlers.
func newProjectionRouteMux(t *testing.T) (*pocketbase.PocketBase, http.Handler) {
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

	cookie := CookiePolicy{Name: testDevelopmentCookieName, Secure: false}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
				return e.Next()
			}
			session, err := ProjectSession(app, e, cookie)
			if err != nil {
				return e.Next()
			}
			e.Request = e.Request.WithContext(
				tmplUtils.WithItineraryProjection(e.Request.Context(), session.CSRF, session.Tray, session.AddedIDs()),
			)
			e.Response.Header().Set("Cache-Control", "private, no-store")
			return e.Next()
		})
		return se.Next()
	})

	if err := RegisterHandlers(app, testSecurityPolicy(false)); err != nil {
		t.Fatalf("register handlers: %v", err)
	}

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(se *core.ServeEvent) error {
		built, err := se.Router.BuildMux()
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

func TestProjectionRouteRotatesMalformedCookieOnce(t *testing.T) {
	_, mux := newProjectionRouteMux(t)

	request := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	request.Header.Set("Accept", "text/html")
	request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: "not-a-valid-token"})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	cookies := response.Result().Cookies()
	var itineraryCookies []*http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == testDevelopmentCookieName {
			itineraryCookies = append(itineraryCookies, cookie)
		}
	}
	if len(itineraryCookies) != 1 {
		t.Fatalf("itinerary Set-Cookie count = %d, want exactly 1", len(itineraryCookies))
	}
	if !itineraryworkflow.ValidToken(itineraryCookies[0].Value) {
		t.Fatalf("issued token %q is not valid", itineraryCookies[0].Value)
	}

	wantCSRF := itineraryworkflow.CSRFToken(itineraryCookies[0].Value)
	if !strings.Contains(response.Body.String(), `value="`+wantCSRF+`"`) {
		t.Error("rendered form CSRF does not match the issued session cookie")
	}
}

func TestProjectionRouteRotatesDuplicateCookiesOnce(t *testing.T) {
	_, mux := newProjectionRouteMux(t)

	first, _ := itineraryworkflow.NewToken()
	second, _ := itineraryworkflow.NewToken()

	request := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	request.Header.Set("Accept", "text/html")
	request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: first})
	request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: second})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	var itineraryCookies []*http.Cookie
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == testDevelopmentCookieName {
			itineraryCookies = append(itineraryCookies, cookie)
		}
	}
	if len(itineraryCookies) != 1 {
		t.Fatalf("itinerary Set-Cookie count = %d, want exactly 1", len(itineraryCookies))
	}
	wantCSRF := itineraryworkflow.CSRFToken(itineraryCookies[0].Value)
	if !strings.Contains(response.Body.String(), `value="`+wantCSRF+`"`) {
		t.Error("rendered form CSRF does not match the issued session cookie")
	}
}

func TestProjectionRouteDoesNotReissueValidCookie(t *testing.T) {
	_, mux := newProjectionRouteMux(t)

	token, err := itineraryworkflow.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	request.Header.Set("Accept", "text/html")
	request.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: token})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == testDevelopmentCookieName {
			t.Fatalf("valid cookie request reissued a session cookie: %+v", cookie)
		}
	}
	if !strings.Contains(response.Body.String(), `value="`+itineraryworkflow.CSRFToken(token)+`"`) {
		t.Error("valid cookie request did not reuse the existing session CSRF")
	}
}

func TestProjectionRouteSameSessionMutationAfterRotation(t *testing.T) {
	app, mux := newProjectionRouteMux(t)

	// First: rotate a malformed cookie and capture the single issued cookie.
	firstRequest := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
	firstRequest.Header.Set("Accept", "text/html")
	firstRequest.AddCookie(&http.Cookie{Name: testDevelopmentCookieName, Value: "not-a-valid-token"})
	firstResponse := httptest.NewRecorder()
	mux.ServeHTTP(firstResponse, firstRequest)

	var sessionCookie *http.Cookie
	for _, cookie := range firstResponse.Result().Cookies() {
		if cookie.Name == testDevelopmentCookieName {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("no session cookie issued")
	}

	// Then: mutate with the issued cookie and its CSRF token.
	form := url.Values{
		"artwork_id": {testArtworkID},
		"_csrf":      {itineraryworkflow.CSRFToken(sessionCookie.Value)},
	}
	mutation := httptest.NewRequest(http.MethodPost, "/itineraries/draft/add", strings.NewReader(form.Encode()))
	mutation.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mutation.AddCookie(sessionCookie)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, mutation)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("mutation status = %d, want 303", response.Code)
	}

	owner := itineraryworkflow.OwnerDigest(sessionCookie.Value)
	draft, err := itineraryworkflow.FindDraft(app, owner)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	stops, err := itineraryworkflow.LoadStops(app, draft.Id)
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if len(stops) != 1 {
		t.Fatalf("persisted stops = %d, want 1", len(stops))
	}
}
