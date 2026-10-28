package dual

import (
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/visitorprefs"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestParseDualStateWithFallbackResolvesPaneViews(t *testing.T) {
	for _, test := range []struct {
		name     string
		raw      string
		fallback string
		want     string
		path     string
	}{
		{name: "default is grid and canonical omits it", raw: "", fallback: viewGrid, want: viewGrid, path: "/dual-mode"},
		{name: "remembered list without parameters", raw: "", fallback: viewList, want: viewList, path: "/dual-mode?l_view=list&r_view=list"},
		{name: "explicit grid wins and stays explicit", raw: "l_view=grid&r_view=grid", fallback: viewList, want: viewGrid, path: "/dual-mode?l_view=grid&r_view=grid"},
		{name: "explicit grid is redundant without a remembered list", raw: "l_view=grid&r_view=grid", fallback: viewGrid, want: viewGrid, path: "/dual-mode"},
		{name: "explicit list wins over the grid fallback", raw: "l_view=list&r_view=list", fallback: viewGrid, want: viewList, path: "/dual-mode?l_view=list&r_view=list"},
		{name: "unknown explicit value is the grid", raw: "l_view=table&r_view=table", fallback: viewList, want: viewGrid, path: "/dual-mode?l_view=grid&r_view=grid"},
		{name: "unknown fallback is the grid", raw: "", fallback: "table", want: viewGrid, path: "/dual-mode"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := parseDualStateWithFallback(mustParseQuery(t, test.raw), test.fallback)
			if state.left.index.view != test.want || state.right.index.view != test.want {
				t.Fatalf("views = %q/%q, want %q", state.left.index.view, state.right.index.view, test.want)
			}
			assertDualPath(t, state.path(), test.path)
		})
	}
}

func TestDualResetKeepsFallbackView(t *testing.T) {
	state := parseDualStateWithFallback(mustParseQuery(t, "l_view=grid&l_letter=A"), viewList)
	reset := state.reset()
	if reset.left.index.view != viewList || reset.right.index.view != viewList {
		t.Fatalf("reset views = %q/%q, want the remembered list", reset.left.index.view, reset.right.index.view)
	}
	assertDualPath(t, reset.path(), "/dual-mode?l_view=list&r_view=list")
}

func TestDualModeInheritsRememberedView(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	consent := neturl.PathEscape(`{"categories":["necessary","preferences"]}`)
	list := neturl.PathEscape(`{"sort":"title","dir":"asc","view":"list","actions":false}`)
	for _, test := range []struct {
		name        string
		target      string
		consent     string
		prefs       string
		wantPush    string
		wantLists   int
		wantPrivate bool
	}{
		{name: "no cookie renders grids", target: "/dual-mode", wantPush: "/dual-mode", wantLists: 0},
		{name: "remembered list", target: "/dual-mode", consent: consent, prefs: list, wantPush: "/dual-mode?l_view=list&r_view=list", wantLists: 2, wantPrivate: true},
		{name: "explicit pane view wins", target: "/dual-mode?l_view=grid", consent: consent, prefs: list, wantPush: "/dual-mode?l_view=grid&r_view=list", wantLists: 1, wantPrivate: true},
		{name: "no consent ignores the remembered list", target: "/dual-mode", prefs: list, wantPush: "/dual-mode", wantLists: 0, wantPrivate: true},
		{name: "invalid remembered view renders grids", target: "/dual-mode", consent: consent, prefs: neturl.PathEscape(`{"view":"table"}`), wantPush: "/dual-mode", wantLists: 0, wantPrivate: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			if test.consent != "" {
				request.AddCookie(&http.Cookie{Name: visitorprefs.ConsentCookieName, Value: test.consent})
			}
			if test.prefs != "" {
				request.AddCookie(&http.Cookie{Name: visitorprefs.CookieName, Value: test.prefs})
			}
			recorder := httptest.NewRecorder()
			event := &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}
			if err := renderDualModePageWithCheckpoint(app, event, requestprotection.Checkpoint); err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := recorder.Header().Get("HX-Push-Url"); got != test.wantPush {
				t.Errorf("HX-Push-Url = %q, want %q", got, test.wantPush)
			}
			body := recorder.Body.String()
			if got := strings.Count(body, `aria-current="true">LIST</a>`); got != test.wantLists {
				t.Errorf("panes reporting LIST as current = %d, want %d", got, test.wantLists)
			}
			if got := strings.Count(body, "data-wga-view-pref="); got != 4 {
				t.Errorf("view toggle markers = %d, want 4", got)
			}
			if !strings.Contains(strings.Join(recorder.Header().Values("Vary"), ","), "Cookie") {
				t.Error("response must vary on Cookie")
			}
			if private := recorder.Header().Get("Cache-Control") == "private, no-store"; private != test.wantPrivate {
				t.Errorf("Cache-Control = %q, want private %v", recorder.Header().Get("Cache-Control"), test.wantPrivate)
			}
		})
	}
}

func TestDualModeLegacyRedirectIsPrivateWithRememberedView(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	request := httptest.NewRequest(http.MethodGet, "/dual-mode?l_size=1100", nil)
	request.AddCookie(&http.Cookie{Name: visitorprefs.ConsentCookieName, Value: neturl.PathEscape(`{"categories":["preferences"]}`)})
	request.AddCookie(&http.Cookie{Name: visitorprefs.CookieName, Value: neturl.PathEscape(`{"view":"list"}`)})
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}
	if err := renderDualModePageWithCheckpoint(app, event, requestprotection.Checkpoint); err != nil {
		t.Fatalf("render: %v", err)
	}
	if recorder.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/dual-mode?l_view=list&r_view=list" {
		t.Errorf("Location = %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", got)
	}
	if !strings.Contains(strings.Join(recorder.Header().Values("Vary"), ","), "Cookie") {
		t.Error("redirect must vary on Cookie")
	}
}
