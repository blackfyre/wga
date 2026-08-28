package static

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestVisualOverhaulReferenceIsEmbedded(t *testing.T) {
	content, err := assets.ReferenceFiles.ReadFile("reference/visual-overhaul.html")
	if err != nil {
		t.Fatalf("read visual overhaul reference: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("visual overhaul reference is empty")
	}
	if string(content[:15]) != "<!DOCTYPE html>" {
		t.Fatal("visual overhaul reference must remain a standalone HTML document")
	}
}

func TestVisualOverhaulRouteExcludesProduction(t *testing.T) {
	if shouldRegisterVisualOverhaul(config.EnvironmentProduction) {
		t.Fatal("visual overhaul route must remain excluded from production")
	}
	if !shouldRegisterVisualOverhaul(config.EnvironmentDevelopment) {
		t.Fatal("visual overhaul route must be available in development")
	}
}

func TestVisualOverhaulFooterFixtureServesDocumentAndHTMXFragment(t *testing.T) {
	app := newStaticTestApp(t)
	RegisterHandlers(app, config.EnvironmentDevelopment)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(se *core.ServeEvent) error {
		mux, err := se.Router.BuildMux()
		if err != nil {
			return err
		}

		page := httptest.NewRecorder()
		mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/tmp/visual-overhaul/footer", nil))
		if page.Code != http.StatusOK {
			t.Fatalf("page status = %d, want %d", page.Code, http.StatusOK)
		}
		for _, expected := range []string{
			`<footer`,
			`hx-get="/tmp/visual-overhaul/footer"`,
			`hx-target="footer"`,
			`src="/assets/js/app.js"`,
		} {
			if !strings.Contains(page.Body.String(), expected) {
				t.Errorf("fixture page must contain %q", expected)
			}
		}

		fragmentRequest := httptest.NewRequest(http.MethodGet, "/tmp/visual-overhaul/footer", nil)
		fragmentRequest.Header.Set("HX-Request", "true")
		fragment := httptest.NewRecorder()
		mux.ServeHTTP(fragment, fragmentRequest)
		if fragment.Code != http.StatusOK {
			t.Fatalf("fragment status = %d, want %d", fragment.Code, http.StatusOK)
		}
		if !strings.HasPrefix(fragment.Body.String(), "<footer") {
			t.Error("HTMX response must contain only the server-rendered footer fragment")
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger routes: %v", err)
	}
}
