package guestbook

import (
	"reflect"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/hooks"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestYearOptionsAreDerivedDistinctAndDescending(t *testing.T) {
	app := newGuestbookTestApp(t)

	saveGuestbookEntry(t, app, "2023-01-01 00:00:00.000Z")
	saveGuestbookEntry(t, app, "2025-01-01 00:00:00.000Z")
	saveGuestbookEntry(t, app, "2023-06-01 00:00:00.000Z")

	assertYearOptions(t, app, []string{"2025", "2023"})
}

func TestYearOptionsAreEmptyWithoutEntries(t *testing.T) {
	app := newGuestbookTestApp(t)

	assertYearOptions(t, app, []string{})
}

func TestYearOptionsRefreshAfterRecordChanges(t *testing.T) {
	app := newGuestbookTestApp(t)

	saveGuestbookEntry(t, app, "2025-01-01 00:00:00.000Z")
	assertYearOptions(t, app, []string{"2025"})

	entry := saveGuestbookEntry(t, app, "2023-01-01 00:00:00.000Z")
	assertYearOptions(t, app, []string{"2025", "2023"})

	entry.Set("created", "2022-01-01 00:00:00.000Z")
	if err := app.Save(entry); err != nil {
		t.Fatalf("update guestbook entry: %v", err)
	}
	assertYearOptions(t, app, []string{"2025", "2022"})

	if err := app.Delete(entry); err != nil {
		t.Fatalf("delete guestbook entry: %v", err)
	}
	assertYearOptions(t, app, []string{"2025"})
}

func newGuestbookTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
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

	got, err := yearOptions(app)
	if err != nil {
		t.Fatalf("get year options: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("year options = %v, want %v", got, want)
	}
}
