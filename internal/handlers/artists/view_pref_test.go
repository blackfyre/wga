package artists

import (
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/visitorprefs"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var (
	viewPrefConsent    = neturl.PathEscape(`{"categories":["necessary","preferences"]}`)
	viewPrefListCookie = neturl.PathEscape(`{"sort":"date","dir":"desc","view":"list","actions":true}`)
)

func TestArtistIndexInheritsRememberedView(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ClearBootstrap(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	createArtistIndexCollections(t, app)
	saveIndexRecord(t, app, "artists", "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "published": true,
	})
	RegisterHandlers(app, config.EnvironmentTest)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		built, err := event.Router.BuildMux()
		mux = built
		return err
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}

	for _, test := range []struct {
		name        string
		target      string
		consent     string
		prefs       string
		wantList    bool
		wantPush    string
		wantGridURL string
		wantPrivate bool
	}{
		{name: "no cookie renders the grid", target: "/artists", wantPush: "/artists", wantGridURL: `href="/artists"`},
		{name: "remembered list without a view parameter", target: "/artists?letter=S", consent: viewPrefConsent, prefs: viewPrefListCookie, wantList: true, wantPush: "/artists?letter=S&view=list", wantGridURL: `href="/artists?letter=S&amp;view=grid"`, wantPrivate: true},
		{name: "explicit grid wins and stays explicit", target: "/artists?view=grid", consent: viewPrefConsent, prefs: viewPrefListCookie, wantPush: "/artists?view=grid", wantGridURL: `href="/artists?view=grid"`, wantPrivate: true},
		{name: "no consent ignores the remembered list", target: "/artists", prefs: viewPrefListCookie, wantPush: "/artists", wantGridURL: `href="/artists"`, wantPrivate: true},
		{name: "invalid remembered view renders the grid", target: "/artists", consent: viewPrefConsent, prefs: neturl.PathEscape(`{"view":"table"}`), wantPush: "/artists", wantGridURL: `href="/artists"`, wantPrivate: true},
		{name: "explicit list wins over a remembered grid", target: "/artists?view=list", consent: viewPrefConsent, prefs: neturl.PathEscape(`{"view":"grid"}`), wantList: true, wantPush: "/artists?view=list", wantGridURL: `href="/artists"`, wantPrivate: true},
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
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d", recorder.Code)
			}
			if got := recorder.Header().Get("HX-Push-Url"); got != test.wantPush {
				t.Errorf("HX-Push-Url = %q, want %q", got, test.wantPush)
			}
			body := recorder.Body.String()
			if listShown := strings.Contains(body, `<table class="wga-table`); listShown != test.wantList {
				t.Errorf("list rendered = %v, want %v", listShown, test.wantList)
			}
			if !strings.Contains(body, test.wantGridURL) {
				t.Errorf("grid toggle missing %s", test.wantGridURL)
			}
			for _, marker := range []string{`data-wga-view-pref="grid"`, `data-wga-view-pref="list"`} {
				if !strings.Contains(body, marker) {
					t.Errorf("toggle missing %s", marker)
				}
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
