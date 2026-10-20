package itineraries

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
)

func postAdd(t *testing.T, mux http.Handler, cookie *http.Cookie, form url.Values, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/add", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func TestSessionTrayWithoutSessionIssuesNothing(t *testing.T) {
	_, mux := newItineraryMux(t)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/itineraries/session", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if cookie := findSessionCookie(t, response); cookie != nil {
		t.Fatalf("session fragment issued a cookie: %v", cookie)
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", got)
	}
	body := response.Body.String()
	if !strings.Contains(body, `id="itinerary-tray"`) || strings.Contains(body, "data-itinerary-csrf") {
		t.Errorf("anonymous session fragment = %s", body)
	}
}

func TestSessionTrayCarriesTokenAndAddedState(t *testing.T) {
	_, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	request := httptest.NewRequest(http.MethodGet, "/itineraries/session", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{
		`data-itinerary-csrf="` + csrf + `"`,
		`data-itinerary-added="` + testArtworkID + `"`,
		`data-itinerary-count="1"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("session fragment missing %q in %s", expected, body)
		}
	}
	if issued := findSessionCookie(t, response); issued != nil {
		t.Errorf("session fragment reissued the cookie: %v", issued)
	}
}

func TestFirstAdditionFromSameOriginCreatesSession(t *testing.T) {
	cases := map[string]map[string]string{
		"origin":         {"Origin": "http://example.com"},
		"referer":        {"Referer": "http://example.com/artworks"},
		"sec-fetch-site": {"Sec-Fetch-Site": "same-origin"},
	}
	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			app, mux := newItineraryMux(t)

			response := postAdd(t, mux, nil, url.Values{"artwork_id": {testArtworkID}}, headers)

			if response.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", response.Code)
			}
			cookie := findSessionCookie(t, response)
			if cookie == nil {
				t.Fatal("first addition did not issue a session cookie")
			}
			if stops := stopsForCookie(t, app, cookie); len(stops) != 1 {
				t.Fatalf("persisted stops = %d, want 1", len(stops))
			}
		})
	}
}

func TestFirstAdditionWithoutSameOriginAssertionIsRefused(t *testing.T) {
	cases := map[string]map[string]string{
		"no origin evidence": {},
		"cross-origin":       {"Origin": "https://evil.example"},
		"cross-site fetch":   {"Sec-Fetch-Site": "cross-site"},
	}
	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			_, mux := newItineraryMux(t)

			response := postAdd(t, mux, nil, url.Values{"artwork_id": {testArtworkID}}, headers)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", response.Code)
			}
			if cookie := findSessionCookie(t, response); cookie != nil {
				t.Errorf("refused addition issued a cookie: %v", cookie)
			}
		})
	}
}

func TestSessionAdditionWithoutTokenAsksForConfirmation(t *testing.T) {
	app, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)
	sameOrigin := map[string]string{"Origin": "http://example.com"}

	response := postAdd(t, mux, cookie, url.Values{"artwork_id": {testArtworkID}}, sameOrigin)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 confirmation", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{`id="itinerary-add-confirm"`, `value="` + csrf + `"`, `value="` + testArtworkID + `"`} {
		if !strings.Contains(body, expected) {
			t.Errorf("confirmation missing %q", expected)
		}
	}
	if stops := stopsForCookie(t, app, cookie); len(stops) != 0 {
		t.Fatalf("confirmation persisted %d stops, want 0", len(stops))
	}
	if issued := findSessionCookie(t, response); issued != nil && issued.Value != cookie.Value {
		t.Error("confirmation replaced the existing session")
	}
}

func TestSessionAdditionRejectsMissingOrInvalidToken(t *testing.T) {
	cases := []struct {
		name    string
		form    url.Values
		headers map[string]string
	}{
		{name: "invalid token", form: url.Values{"artwork_id": {testArtworkID}, "_csrf": {"forged"}}, headers: map[string]string{"Origin": "http://example.com"}},
		{name: "cross-origin without token", form: url.Values{"artwork_id": {testArtworkID}}, headers: map[string]string{"Origin": "https://evil.example"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			app, mux := newItineraryMux(t)
			cookie, _ := sessionForMux(t, mux)

			response := postAdd(t, mux, cookie, test.form, test.headers)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", response.Code)
			}
			if stops := stopsForCookie(t, app, cookie); len(stops) != 0 {
				t.Errorf("rejected addition persisted %d stops", len(stops))
			}
		})
	}
}

func TestAssertedSameOriginRequiresPositiveEvidence(t *testing.T) {
	canonical, _ := url.Parse("https://example.com")
	cases := []struct {
		name    string
		host    string
		headers map[string]string
		want    bool
	}{
		{name: "matching origin", host: "example.com", headers: map[string]string{"Origin": "https://example.com"}, want: true},
		{name: "matching referer", host: "example.com", headers: map[string]string{"Referer": "https://example.com/a"}, want: true},
		{name: "fetch metadata", host: "example.com", headers: map[string]string{"Sec-Fetch-Site": "same-origin"}, want: true},
		{name: "no evidence", host: "example.com", headers: map[string]string{}, want: false},
		{name: "same-site fetch", host: "example.com", headers: map[string]string{"Sec-Fetch-Site": "same-site"}, want: false},
		{name: "foreign origin", host: "example.com", headers: map[string]string{"Origin": "https://evil.example"}, want: false},
		{name: "foreign host", host: "evil.example", headers: map[string]string{"Origin": "https://example.com"}, want: false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "https://"+test.host+"/itineraries/draft/add", nil)
			for name, value := range test.headers {
				request.Header.Set(name, value)
			}
			if got := itineraryworkflow.AssertedSameOrigin(request, canonical); got != test.want {
				t.Errorf("AssertedSameOrigin = %t, want %t", got, test.want)
			}
		})
	}
}

func TestSessionHTMXAdditionWithoutTokenReturnsSessionTray(t *testing.T) {
	app, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)

	response := postAdd(t, mux, cookie, url.Values{"artwork_id": {testArtworkID}}, map[string]string{
		"Origin":     "http://example.com",
		"HX-Request": "true",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 session tray", response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, `data-itinerary-csrf="`+csrf+`"`) {
		t.Errorf("refresh response lacks the session token: %s", body)
	}
	if !strings.Contains(response.Header().Get("HX-Trigger"), "Add the work again") {
		t.Errorf("HX-Trigger = %q, want a retry toast", response.Header().Get("HX-Trigger"))
	}
	if stops := stopsForCookie(t, app, cookie); len(stops) != 0 {
		t.Fatalf("refresh persisted %d stops, want 0", len(stops))
	}
}

func TestRejectedFirstAdditionCreatesNoSession(t *testing.T) {
	_, mux := newItineraryMux(t)

	response := postAdd(t, mux, nil, url.Values{"artwork_id": {"missing-artwork"}}, map[string]string{"Origin": "http://example.com"})

	if response.Code < http.StatusBadRequest {
		t.Fatalf("status = %d, want a rejection", response.Code)
	}
	if cookie := findSessionCookie(t, response); cookie != nil {
		t.Errorf("rejected first addition issued a session cookie: %v", cookie)
	}
}

func TestBuilderMutationTrayKeepsAddedState(t *testing.T) {
	_, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	request := httptest.NewRequest(http.MethodPost, "/itineraries/draft/meta", strings.NewReader(url.Values{"_csrf": {csrf}, "title": {"A walk"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, `data-itinerary-added="`+testArtworkID+`"`) {
		t.Errorf("builder mutation tray lost the added state: %s", body)
	}
}

func TestConfirmationPageRendersTheVisitorTray(t *testing.T) {
	_, mux := newItineraryMux(t)
	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	response := postAdd(t, mux, cookie, url.Values{"artwork_id": {testArtworkID}}, map[string]string{"Origin": "http://example.com"})

	body := response.Body.String()
	if !strings.Contains(body, `id="itinerary-add-confirm"`) {
		t.Fatalf("expected the confirmation page, got %d", response.Code)
	}
	if !strings.Contains(body, `data-itinerary-count="1"`) || strings.Contains(body, `hx-get="/itineraries/session"`) {
		t.Errorf("confirmation page must render the visitor tray on the server")
	}
}
