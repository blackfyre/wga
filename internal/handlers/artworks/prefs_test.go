package artworks

import (
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const rememberedDateListCookie = "%7B%22sort%22%3A%22date%22%2C%22dir%22%3A%22desc%22%2C%22view%22%3A%22list%22%2C%22actions%22%3Atrue%7D"

const preferenceConsentCookie = "%7B%22categories%22%3A%5B%22necessary%22%2C%22preferences%22%5D%2C%22revision%22%3A0%7D"

// searchPrefsRequest carries the remembered cookie with preference consent.
func searchPrefsRequest(target string, cookie string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.AddCookie(&http.Cookie{Name: consentCookieName, Value: preferenceConsentCookie})
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: searchPrefsCookieName, Value: cookie})
	}
	return request
}

func TestStoredSearchPrefsRequirePreferenceConsent(t *testing.T) {
	for _, test := range []struct {
		name    string
		consent string
	}{
		{name: "no consent record"},
		{name: "essential only", consent: neturl.PathEscape(`{"categories":["necessary"]}`)},
		{name: "malformed record", consent: "%7Bbroken"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/artworks", nil)
			if test.consent != "" {
				request.AddCookie(&http.Cookie{Name: consentCookieName, Value: test.consent})
			}
			request.AddCookie(&http.Cookie{Name: searchPrefsCookieName, Value: rememberedDateListCookie})
			if got := storedSearchPrefs(request); got != (searchPrefs{}) {
				t.Fatalf("storedSearchPrefs() = %#v, want stale preferences ignored", got)
			}
		})
	}
}

func TestStoredSearchPrefsDecodesEachFieldIndependently(t *testing.T) {
	for _, test := range []struct {
		name   string
		cookie string
		want   searchPrefs
	}{
		{name: "absent", want: searchPrefs{}},
		{name: "remembered", cookie: rememberedDateListCookie, want: searchPrefs{Present: true, Sort: "date", Dir: "desc", View: "list", Actions: true}},
		{name: "not encoded json", cookie: "%zz", want: searchPrefs{}},
		{name: "not an object", cookie: neturl.PathEscape(`["date"]`), want: searchPrefs{}},
		{name: "malformed json", cookie: neturl.PathEscape(`{"sort":`), want: searchPrefs{}},
		{
			name:   "invalid fields ignored",
			cookie: neturl.PathEscape(`{"sort":"catalogue","dir":"down","view":"list","actions":"yes"}`),
			want:   searchPrefs{Present: true, View: "list"},
		},
		{
			name:   "wrong types ignored",
			cookie: neturl.PathEscape(`{"sort":3,"dir":"asc","actions":false}`),
			want:   searchPrefs{Present: true, Dir: "asc"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := storedSearchPrefs(searchPrefsRequest("/artworks", test.cookie)); got != test.want {
				t.Fatalf("storedSearchPrefs() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestSearchPrefsActionsAttribute(t *testing.T) {
	if got := (searchPrefs{}).ActionsAttribute(); got != "" {
		t.Errorf("absent preferences attribute = %q, want empty", got)
	}
	if got := (searchPrefs{Present: true}).ActionsAttribute(); got != "off" {
		t.Errorf("stored off attribute = %q, want off", got)
	}
	if got := (searchPrefs{Present: true, Actions: true}).ActionsAttribute(); got != "on" {
		t.Errorf("stored on attribute = %q, want on", got)
	}
}

func TestSearchRequestValuesAppliesSearchPrefsOnlyToBareVisit(t *testing.T) {
	prefs := searchPrefs{Present: true, Sort: "date", Dir: "desc", View: "list"}
	for _, test := range []struct {
		name   string
		target string
		want   string
	}{
		{name: "bare visit uses remembered state", target: "/artworks", want: "dir=desc&sort=date&view=list"},
		{name: "explicit filter wins", target: "/artworks?q=mary", want: "q=mary"},
		{name: "explicit sort wins", target: "/artworks?sort=title", want: "sort=title"},
		{name: "explicit page wins", target: "/artworks?page=2", want: "page=2"},
		{name: "fragment requests are used as addressed", target: "/artworks/results", want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			u, err := neturl.Parse(test.target)
			if err != nil {
				t.Fatal(err)
			}
			if got := searchRequestValues(u, prefs).Encode(); got != test.want {
				t.Fatalf("searchRequestValues(%q) = %q, want %q", test.target, got, test.want)
			}
		})
	}

	u, _ := neturl.Parse("/artworks")
	if got := searchRequestValues(u, searchPrefs{Present: true, View: "list"}).Encode(); got != "view=list" {
		t.Errorf("partial preferences = %q, want only the valid view", got)
	}
}

func serveArtworkSearch(t *testing.T, app *pocketbase.PocketBase, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}
	if err := searchWithCheckpoint(app, event, requestprotection.Checkpoint); err != nil {
		t.Fatalf("searchWithCheckpoint(%s) error = %v", request.URL, err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("searchWithCheckpoint(%s) status = %d", request.URL, recorder.Code)
	}
	return recorder
}

func TestArtworkSearchRendersSearchPrefsFromCookie(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistprefs0001", "Artist Prefs")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workprefsearly1", title: "Alpha Early", authors: []string{"artistprefs0001"}, dateStart: 1500, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workprefslate01", title: "Bravo Late", authors: []string{"artistprefs0001"}, dateStart: 1600, published: true})

	t.Run("bare visit renders the remembered state", func(t *testing.T) {
		recorder := serveArtworkSearch(t, app, searchPrefsRequest("/artworks", rememberedDateListCookie))
		body := recorder.Body.String()
		if got := recorder.Header().Get("HX-Push-Url"); got != "/artworks?dir=desc&sort=date&view=list" {
			t.Errorf("HX-Push-Url = %q, want the remembered canonical state", got)
		}
		if got := recorder.Header().Values("Vary"); !containsValue(got, "Cookie") {
			t.Errorf("Vary = %v, want Cookie", got)
		}
		for _, expected := range []string{
			`<html lang="en" data-aw-actions="on">`,
			`data-view="list"`,
			"VIEW: LIST",
			"LATEST",
			`data-wga-aw-actions aria-pressed="true"`,
			"ACTIONS ✓",
			`type="hidden" name="sort" value="date"`,
			`type="hidden" name="dir" value="desc"`,
			`type="hidden" name="view" value="list"`,
		} {
			if !strings.Contains(body, expected) {
				t.Errorf("remembered page missing %q", expected)
			}
		}
		if late, early := strings.Index(body, "Bravo Late"), strings.Index(body, "Alpha Early"); late < 0 || early < 0 || late > early {
			t.Error("remembered date-descending sort must list the later work first")
		}
	})

	t.Run("explicit query parameters win", func(t *testing.T) {
		recorder := serveArtworkSearch(t, app, searchPrefsRequest("/artworks?q=a", rememberedDateListCookie))
		body := recorder.Body.String()
		if got := recorder.Header().Get("HX-Push-Url"); got != "/artworks?q=a" {
			t.Errorf("HX-Push-Url = %q, want the addressed state unchanged", got)
		}
		for _, expected := range []string{`data-view="grid"`, "VIEW: GRID", "A–Z", "ACTIONS ✓"} {
			if !strings.Contains(body, expected) {
				t.Errorf("explicit page missing %q", expected)
			}
		}
		if early, late := strings.Index(body, "Alpha Early"), strings.Index(body, "Bravo Late"); early < 0 || late < 0 || early > late {
			t.Error("explicit request must keep the default title order")
		}
	})

	t.Run("reset keeps the remembered state", func(t *testing.T) {
		filtered := serveArtworkSearch(t, app, searchPrefsRequest("/artworks?q=a&sort=date&dir=desc&view=list", rememberedDateListCookie))
		if !strings.Contains(filtered.Body.String(), `href="/artworks">RESET</a>`) {
			t.Fatal("RESET must link to the bare /artworks address")
		}
		reset := serveArtworkSearch(t, app, searchPrefsRequest("/artworks", rememberedDateListCookie))
		if got := reset.Header().Get("HX-Push-Url"); got != "/artworks?dir=desc&sort=date&view=list" {
			t.Errorf("reset HX-Push-Url = %q, want the remembered presentation retained", got)
		}
	})

	t.Run("invalid cookie is ignored", func(t *testing.T) {
		recorder := serveArtworkSearch(t, app, searchPrefsRequest("/artworks", neturl.PathEscape(`{"sort":"catalogue"`)))
		body := recorder.Body.String()
		if got := recorder.Header().Get("HX-Push-Url"); got != "/artworks" {
			t.Errorf("HX-Push-Url = %q, want defaults", got)
		}
		if strings.Contains(body, "data-aw-actions=") {
			t.Error("an unusable cookie must not set the actions attribute")
		}
		for _, expected := range []string{`data-view="grid"`, `data-wga-aw-actions aria-pressed="false"`, "ACTIONS +"} {
			if !strings.Contains(body, expected) {
				t.Errorf("default page missing %q", expected)
			}
		}
	})

	t.Run("fragment requests read only the actions setting", func(t *testing.T) {
		request := searchPrefsRequest("/artworks/results", rememberedDateListCookie)
		request.Header.Set("HX-Request", "true")
		body := serveArtworkSearch(t, app, request).Body.String()
		for _, expected := range []string{`data-view="grid"`, "ACTIONS ✓"} {
			if !strings.Contains(body, expected) {
				t.Errorf("fragment missing %q", expected)
			}
		}
	})
}

func containsValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
