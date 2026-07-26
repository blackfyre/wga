package licences

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestOpenSourceLicencesPage(t *testing.T) {
	ctx := utils.DecorateContext(context.Background(), utils.TitleKey, "Open-source licences")
	var output bytes.Buffer
	if err := pages.LicencesPage(assets.OpenSourceLicencesHTML).Render(ctx, &output); err != nil {
		t.Fatalf("render licences page: %v", err)
	}

	for _, expected := range []string{"Open-source licences", "third-party components", "<footer"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("rendered page does not contain %q", expected)
		}
	}
}

func TestOpenSourceLicencesRoute(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test application: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset test application: %v", err)
		}
	})
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
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/open-source-licences", nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if !strings.Contains(recorder.Body.String(), "Open-source licences") {
			t.Error("route response does not contain licence notices")
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
