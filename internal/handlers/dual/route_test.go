package dual

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// TestDualModeRouteRendersFullPageAndLocalBlock pins the shell-versus-feature
// response contract: ordinary and boosted shell navigation return the selectable
// full #mc-area document (exactly once), while a feature-local pane/bar request
// returns only the dual block.
func TestDualModeRouteRendersFullPageAndLocalBlock(t *testing.T) {
	app := newDualTestApp(t)
	RegisterHandlers(app)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		serve := func(target string) string {
			t.Helper()
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/dual-mode", nil)
			if target != "" {
				request.Header.Set("HX-Request", "true")
				request.Header.Set("HX-Target", target)
			}
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status for target %q = %d, want 200", target, recorder.Code)
			}
			return recorder.Body.String()
		}

		ordinary := serve("")
		if !strings.Contains(ordinary, "<html") {
			t.Error("ordinary GET must render the full document")
		}
		if strings.Count(ordinary, `id="mc-area"`) != 1 {
			t.Error("ordinary full document must carry exactly one #mc-area")
		}
		if !strings.Contains(ordinary, `id="dual-area"`) {
			t.Error("ordinary full document must render the dual surface")
		}

		shell := serve("mc-area")
		if !strings.Contains(shell, "<html") {
			t.Error("boosted shell navigation must render the full document")
		}
		if strings.Count(shell, `id="mc-area"`) != 1 {
			t.Error("shell navigation document must carry exactly one #mc-area")
		}
		if !strings.Contains(shell, `id="dual-area"`) {
			t.Error("shell navigation document must render the dual surface")
		}

		local := serve("dual-area")
		if strings.Contains(local, "<html") {
			t.Error("feature-local dual response must not render the full document")
		}
		if !strings.Contains(local, `id="dual-area"`) {
			t.Error("feature-local dual response must render the dual block")
		}
		if strings.Contains(local, `id="mc-area"`) {
			t.Error("feature-local dual response must not carry #mc-area")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

// TestDualModePreservesOnlyTheIssuingPaneNameField pins that only a request
// issued by a pane's own filter form keeps that pane's in-progress name input;
// the other pane and every other request render server values.
func TestDualModePreservesOnlyTheIssuingPaneNameField(t *testing.T) {
	app := newDualTestApp(t)
	RegisterHandlers(app)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		tests := []struct {
			name      string
			trigger   string
			preserved string
		}{
			{name: "left form", trigger: "dual-filters-left", preserved: "l-name"},
			{name: "right form", trigger: "dual-filters-right", preserved: "r-name"},
			{name: "reset or pane link", trigger: "", preserved: ""},
		}
		for _, test := range tests {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/dual-mode?l_q=van&r_q=van", nil)
			request.Header.Set("HX-Request", "true")
			request.Header.Set("HX-Target", "dual-area")
			if test.trigger != "" {
				request.Header.Set("HX-Trigger", test.trigger)
			}
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("%s: status = %d, want 200", test.name, recorder.Code)
			}
			body := recorder.Body.String()
			for _, id := range []string{"l-name", "r-name"} {
				if !strings.Contains(body, `id="`+id+`"`) {
					t.Fatalf("%s: response missing name field %s", test.name, id)
				}
			}
			want := 0
			if test.preserved != "" {
				want = 1
				if !strings.Contains(body, `id="`+test.preserved+`" name=`) {
					t.Fatalf("%s: preserved field %s missing", test.name, test.preserved)
				}
			}
			if got := strings.Count(body, "hx-preserve"); got != want {
				t.Errorf("%s: hx-preserve count = %d, want %d", test.name, got, want)
			}
			if test.preserved != "" {
				start := strings.Index(body, `id="`+test.preserved+`"`)
				end := start + strings.Index(body[start:], ">")
				if !strings.Contains(body[start:end], "hx-preserve") {
					t.Errorf("%s: hx-preserve not on %s", test.name, test.preserved)
				}
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
