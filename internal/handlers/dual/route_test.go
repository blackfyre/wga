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
