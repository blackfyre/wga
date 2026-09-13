package itineraries

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
)

func TestBoardImportDisclosesAndConfirmsDraftReplacement(t *testing.T) {
	app, mux := newItineraryMux(t)
	createArtworks(t, app, 2, 3)
	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {"aw0000000000002"}})
	stops := stopsForCookie(t, app, cookie)
	if err := itineraryworkflow.SetNarration(app, itineraryworkflow.OwnerDigest(cookie.Value), stops[0].Id, "Existing narration"); err != nil {
		t.Fatalf("SetNarration: %v", err)
	}

	review := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/itineraries/draft/from-board?board=aw0000000000003,aw0000000000002", nil)
	request.AddCookie(cookie)
	mux.ServeHTTP(review, request)
	if review.Code != http.StatusOK {
		t.Fatalf("review status = %d, want 200", review.Code)
	}
	body := review.Body.String()
	for _, expected := range []string{
		"2 ordered Study Board works",
		"replace existing works (2) and discard narrated stops (1)",
		"The Study Board remains unchanged",
		`name="confirm" value="replace"`,
		"REPLACE DRAFT",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("review missing %q", expected)
		}
	}
	expectation := hiddenFormValue(body, "expectation")
	if expectation == "" {
		t.Fatal("review must bind replacement to an expectation")
	}

	unconfirmed := postForm(t, mux, "/itineraries/draft/from-board", cookie, csrf, url.Values{
		"board":       {"aw0000000000003,aw0000000000002"},
		"expectation": {expectation},
	})
	if unconfirmed.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed status = %d, want 400", unconfirmed.Code)
	}
	if got := stopsForCookie(t, app, cookie); len(got) != 2 || got[0].GetString("narration") == "" {
		t.Fatalf("unconfirmed request mutated draft: %#v", got)
	}

	confirmed := postForm(t, mux, "/itineraries/draft/from-board", cookie, csrf, url.Values{
		"board":       {"aw0000000000003,aw0000000000002"},
		"expectation": {expectation},
		"confirm":     {"replace"},
	})
	if confirmed.Code != http.StatusSeeOther || confirmed.Header().Get("Location") != "/itineraries/new" {
		t.Fatalf("confirmed response = %d Location %q", confirmed.Code, confirmed.Header().Get("Location"))
	}
	replaced := stopsForCookie(t, app, cookie)
	if len(replaced) != 2 || replaced[0].GetString("artwork") != "aw0000000000003" || replaced[1].GetString("artwork") != "aw0000000000002" {
		t.Fatalf("replacement order = %#v", replaced)
	}
	if replaced[0].GetString("narration") != "" || replaced[1].GetString("narration") != "" {
		t.Fatal("replacement must not retain narration from deleted stops")
	}
}

func TestBoardImportCreatesDraftWithoutDestructiveConfirmation(t *testing.T) {
	app, mux := newItineraryMux(t)
	createArtworks(t, app, 2, 2)
	cookie, csrf := sessionForMux(t, mux)

	review := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/itineraries/draft/from-board?board=aw0000000000002,aw0000000000001", nil)
	request.AddCookie(cookie)
	mux.ServeHTTP(review, request)
	body := review.Body.String()
	if !strings.Contains(body, "CREATE ITINERARY") || strings.Contains(body, `name="confirm"`) {
		t.Fatalf("empty draft review has wrong confirmation contract: %s", body)
	}

	response := postForm(t, mux, "/itineraries/draft/from-board", cookie, csrf, url.Values{
		"board":       {"aw0000000000002,aw0000000000001"},
		"expectation": {hiddenFormValue(body, "expectation")},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", response.Code)
	}
	stops := stopsForCookie(t, app, cookie)
	if len(stops) != 2 || stops[0].GetString("artwork") != "aw0000000000002" {
		t.Fatalf("created draft order = %#v", stops)
	}
}

func hiddenFormValue(body string, name string) string {
	marker := `name="` + name + `" value="`
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
