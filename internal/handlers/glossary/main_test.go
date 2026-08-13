package glossary

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestFindGlossaryTermsFiltersAndOrdersTerms(t *testing.T) {
	app := newGlossaryTestApp(t)
	saveGlossaryTerm(t, app, "Brushwork", "The handling of paint.")
	saveGlossaryTerm(t, app, "Acanthus", "An ornamental leaf.")
	saveGlossaryTerm(t, app, "Aerial perspective", "Creates the impression of distance.")
	saveGlossaryTerm(t, app, "Allegory", "A symbolic narrative about <em>distance</em>.")

	terms, err := findGlossaryTerms(app, glossaryQuery{Letter: "A", Text: "distance"})
	if err != nil {
		t.Fatalf("find glossary terms: %v", err)
	}

	if len(terms) != 2 {
		t.Fatalf("expected 2 matching terms, got %d", len(terms))
	}
	if terms[0].Expression != "Aerial perspective" || terms[1].Expression != "Allegory" {
		t.Errorf("unexpected term ordering: %#v", terms)
	}
	if terms[1].Definition != "A symbolic narrative about <em>distance</em>." {
		t.Errorf("definition was not sanitised: %q", terms[1].Definition)
	}
}

func TestGlossaryLettersUsesAvailableInitials(t *testing.T) {
	letters := glossaryLetters([]pages.GlossaryTerm{
		{Expression: "Brushwork"},
		{Expression: "Acanthus"},
		{Expression: "Allegory"},
		{Expression: "1-point perspective"},
	})

	if got, want := strings.Join(letters, ","), "A,B"; got != want {
		t.Errorf("letters = %q, want %q", got, want)
	}
}

func TestGlossaryRouteRendersFullAndHTMXResponses(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test application: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset test application: %v", err)
		}
	})
	createGlossaryCollection(t, app)
	saveGlossaryTerm(t, app, "Acanthus", "An ornamental leaf.")
	saveGlossaryTerm(t, app, "Brushwork", "The handling of paint.")
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

		full := httptest.NewRecorder()
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/glossary?letter=A", nil))
		if full.Code != http.StatusOK {
			t.Errorf("full response status = %d, want %d", full.Code, http.StatusOK)
		}
		if !strings.Contains(full.Body.String(), "<html") || !strings.Contains(full.Body.String(), "Acanthus") || strings.Contains(full.Body.String(), "Brushwork") {
			t.Error("full response did not render the selected glossary terms")
		}
		if got := full.Header().Get("HX-Push-Url"); got != "/glossary?letter=A" {
			t.Errorf("HX-Push-Url = %q, want /glossary?letter=A", got)
		}

		partial := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/glossary?q=paint", nil)
		request.Header.Set("HX-Request", "true")
		mux.ServeHTTP(partial, request)
		if partial.Code != http.StatusOK {
			t.Errorf("partial response status = %d, want %d", partial.Code, http.StatusOK)
		}
		if strings.Contains(partial.Body.String(), "<html") || !strings.Contains(partial.Body.String(), "Brushwork") {
			t.Error("HTMX response did not render only the glossary block")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func newGlossaryTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	createGlossaryCollection(t, app)

	return app
}

func createGlossaryCollection(t *testing.T, app core.App) {
	t.Helper()

	collection := core.NewBaseCollection(constants.CollectionGlossary)
	collection.Fields.Add(
		&core.TextField{Name: "expression", Required: true},
		&core.TextField{Name: "definition", Required: true},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("create glossary collection: %v", err)
	}
}

func saveGlossaryTerm(t *testing.T, app core.App, expression string, definition string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(constants.CollectionGlossary)
	if err != nil {
		t.Fatalf("find glossary collection: %v", err)
	}

	term := core.NewRecord(collection)
	term.Set("expression", expression)
	term.Set("definition", definition)
	if err := app.Save(term); err != nil {
		t.Fatalf("save glossary term: %v", err)
	}
}
