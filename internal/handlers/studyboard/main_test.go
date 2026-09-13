package studyboard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func newStudyBoardRouteApp(t *testing.T) (*pocketbase.PocketBase, func(string, bool) *httptest.ResponseRecorder) {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	collection := core.NewBaseCollection("Artworks", constants.CollectionArtworks)
	collection.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save Artworks collection: %v", err)
	}

	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error { return nil }); err != nil {
		t.Fatalf("trigger serve: %v", err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatalf("build mux: %v", err)
	}

	request := func(path string, htmx bool) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if htmx {
			req.Header.Set("HX-Request", "true")
		}
		mux.ServeHTTP(recorder, req)
		return recorder
	}
	return app, request
}

func saveArtwork(t *testing.T, app *pocketbase.PocketBase, id, title string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatalf("find Artworks collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", title)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork %q: %v", id, err)
	}
}

func TestStudyBoardCanonicalisesPublishedUniqueWorksAtCapacity(t *testing.T) {
	app, request := newStudyBoardRouteApp(t)
	ids := make([]string, 14)
	for i := range ids {
		ids[i] = fmt.Sprintf("work%011d", i)
		saveArtwork(t, app, ids[i], fmt.Sprintf("Work %02d", i), true)
	}
	hiddenID := "hidden000000001"
	saveArtwork(t, app, hiddenID, "Secret title", false)

	raw := append([]string{ids[0], "invalid id", "missing00000001", hiddenID, ids[0]}, ids[1:]...)
	recorder := request(route+"?extra=ignored&board="+url.QueryEscape(strings.Join(raw, ",")), false)
	wantIDs := ids[:12]
	wantLocation := route + "?board=" + strings.Join(wantIDs, ",")
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}
	if got := recorder.Header().Get("Location"); got != wantLocation {
		t.Fatalf("Location = %q, want %q", got, wantLocation)
	}
	if strings.Contains(recorder.Body.String(), "Secret title") {
		t.Fatal("canonicalisation exposed unpublished metadata")
	}

	canonical := request(wantLocation, false)
	if canonical.Code != http.StatusOK {
		t.Fatalf("canonical status = %d, want 200", canonical.Code)
	}
	body := canonical.Body.String()
	for _, id := range wantIDs {
		if got := strings.Count(body, `data-study-board-work="`+id+`"`); got != 1 {
			t.Errorf("work %q rendered %d times, want once", id, got)
		}
	}
	if strings.Contains(body, ids[12]) || strings.Contains(body, "Secret title") {
		t.Fatal("canonical board exposed overflow or unpublished work")
	}
	if !strings.Contains(body, `data-study-board-capacity="true"`) {
		t.Error("capacity state was not projected")
	}
}

func TestStudyBoardUsesURLStateAndProjectsTransientActions(t *testing.T) {
	app, request := newStudyBoardRouteApp(t)
	id := "work00000000001"
	saveArtwork(t, app, id, "Shared work", true)

	recorder := request(route+"?board="+id, false)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != route+"?board="+id {
		t.Errorf("HX-Push-Url = %q", got)
	}
	for _, expected := range []string{
		`data-study-board-url-state="true"`,
		`data-study-board-ids="` + id + `"`,
		"COPY LINK",
		"CLEAR",
		"no account, title, narration, publication state, archive entry, server record, or scheduled expiry",
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}
	if _, err := app.FindCollectionByNameOrId("study_boards"); err == nil {
		t.Fatal("Study Board unexpectedly created a persistence collection")
	}
}

func TestStudyBoardBaseRouteAllowsBrowserLocalRestoration(t *testing.T) {
	_, request := newStudyBoardRouteApp(t)
	recorder := request(route, false)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `data-study-board-url-state="false"`) {
		t.Fatal("base route did not advertise absent URL state")
	}
}

func TestStudyBoardHTMXCanonicalisationPushesWithoutRedirect(t *testing.T) {
	app, request := newStudyBoardRouteApp(t)
	id := "work00000000001"
	saveArtwork(t, app, id, "Shared work", true)

	recorder := request(route+"?board=missing00000001,"+id+","+id, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != route+"?board="+id {
		t.Errorf("HX-Push-Url = %q, want canonical board", got)
	}
	if got := strings.Count(recorder.Body.String(), `data-study-board-work="`+id+`"`); got != 1 {
		t.Errorf("work rendered %d times, want once", got)
	}
}

func TestStudyBoardCollapsesRepeatedBoardValuesIntoOneCanonicalValue(t *testing.T) {
	app, request := newStudyBoardRouteApp(t)
	first := "work00000000001"
	second := "work00000000002"
	saveArtwork(t, app, first, "First work", true)
	saveArtwork(t, app, second, "Second work", true)

	recorder := request(route+"?board="+first+"&board="+second, false)
	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}
	if got := recorder.Header().Get("Location"); got != route+"?board="+first+","+second {
		t.Errorf("Location = %q, want one ordered board value", got)
	}
}
