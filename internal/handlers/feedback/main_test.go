package feedback

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blackfyre/wga/internal/buildinfo"
	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestFeedbackFormView(t *testing.T) {
	originalVersion := buildinfo.Version
	buildinfo.Version = "2.0.0-rc4"
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
	})

	tests := []struct {
		name    string
		referer string
		context string
	}{
		{name: "home", referer: "https://wga.example/", context: "Home"},
		{name: "other page", referer: "https://wga.example/artworks", context: "Current page"},
		{name: "missing", context: "Current page"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := feedbackFormView(test.referer, config.EnvironmentStaging)
			if view.Context != test.context {
				t.Errorf("context = %q, want %q", view.Context, test.context)
			}
			if view.Build != "staging · 2.0.0-rc4" {
				t.Errorf("build = %q", view.Build)
			}
		})
	}
}

func TestFeedbackRouteRejectsNonHTMXHeadRequest(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test application: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset test application: %v", err)
		}
	})
	RegisterHandlers(app, config.EnvironmentTest)

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

		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/feedback", nil))
		if response.Code != http.StatusBadRequest {
			t.Errorf("response status = %d, want %d", response.Code, http.StatusBadRequest)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
