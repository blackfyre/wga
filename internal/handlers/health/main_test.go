package health

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestHealthEndpoint(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset bootstrap state: %v", err)
		}
	})

	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(*core.ServeEvent) error { return nil }); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatalf("build mux: %v", err)
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", contentType)
	}
	if got := recorder.Body.String(); got != "ok" {
		t.Errorf("body = %q, want %q", got, "ok")
	}
}
