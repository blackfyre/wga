package guestbook

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/hooks"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestBuildFilters(t *testing.T) {
	request := httptest.NewRequest("GET", "/guestbook?q=%20%20Jane%20%20&year=all&sort=oldest&show=20", nil)
	got := buildFilters(&core.RequestEvent{Event: router.Event{Request: request}}, "2026")
	want := filters{Query: "Jane", Year: "all", Sort: "oldest", Show: 20}

	if got != want {
		t.Errorf("filters = %#v", got)
	}

	filter, params := got.buildFilter()
	if filter != "(name ~ {:query} || location ~ {:query} || message ~ {:query})" {
		t.Errorf("filter = %q", filter)
	}
	if params["query"] != "Jane" {
		t.Errorf("query = %q", params["query"])
	}
}

func TestBuildFiltersDefaults(t *testing.T) {
	request := httptest.NewRequest("GET", "/guestbook?sort=invalid&show=2", nil)
	got := buildFilters(&core.RequestEvent{Event: router.Event{Request: request}}, "2026")
	want := filters{Year: "2026", Sort: "newest", Show: guestbookPageSize}

	if got != want {
		t.Errorf("filters = %#v", got)
	}
	if got.sortExpression() != "-created" {
		t.Errorf("sort expression = %q", got.sortExpression())
	}
}

func TestYearOptionsIncludeCurrentYearWithDerivedYears(t *testing.T) {
	app := newGuestbookTestApp(t)

	saveGuestbookEntry(t, app, "2023-01-01 00:00:00.000Z")
	saveGuestbookEntry(t, app, "2025-01-01 00:00:00.000Z")
	saveGuestbookEntry(t, app, "2023-06-01 00:00:00.000Z")

	assertYearOptions(t, app, []string{"2026", "2025", "2023"})
}

func TestYearOptionsIncludeCurrentYearWithoutEntries(t *testing.T) {
	app := newGuestbookTestApp(t)

	assertYearOptions(t, app, []string{"2026"})
}

func TestYearOptionsRefreshAfterRecordChanges(t *testing.T) {
	app := newGuestbookTestApp(t)

	saveGuestbookEntry(t, app, "2025-01-01 00:00:00.000Z")
	assertYearOptions(t, app, []string{"2026", "2025"})

	entry := saveGuestbookEntry(t, app, "2023-01-01 00:00:00.000Z")
	assertYearOptions(t, app, []string{"2026", "2025", "2023"})

	entry.Set("created", "2022-01-01 00:00:00.000Z")
	if err := app.Save(entry); err != nil {
		t.Fatalf("update guestbook entry: %v", err)
	}
	assertYearOptions(t, app, []string{"2026", "2025", "2022"})

	if err := app.Delete(entry); err != nil {
		t.Fatalf("delete guestbook entry: %v", err)
	}
	assertYearOptions(t, app, []string{"2026", "2025"})
}

func newGuestbookTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	collection := core.NewBaseCollection(constants.CollectionGuestbook)
	collection.Fields.Add(
		&core.TextField{Name: "created"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("create guestbook collection: %v", err)
	}

	hooks.RegisterHooks(app)

	return app
}

func saveGuestbookEntry(t *testing.T, app core.App, created string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(constants.CollectionGuestbook)
	if err != nil {
		t.Fatalf("find guestbook collection: %v", err)
	}

	entry := core.NewRecord(collection)
	entry.Set("created", created)
	if err := app.Save(entry); err != nil {
		t.Fatalf("create guestbook entry: %v", err)
	}

	return entry
}

func assertYearOptions(t *testing.T, app core.App, want []string) {
	t.Helper()

	got, err := yearOptions(app, "2026")
	if err != nil {
		t.Fatalf("get year options: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("year options = %v, want %v", got, want)
	}
}
