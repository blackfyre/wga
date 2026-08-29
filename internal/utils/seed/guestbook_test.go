package seed

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestImportSyntheticGuestbookDoesNotRetainEmail(t *testing.T) {
	app := newGuestbookSeedTestApp(t)
	if err := importSyntheticGuestbook(app, []sourceGuestbookEntry{{
		ID:       "raaaaaaaaaaaaaa",
		Name:     "Archive visitor",
		Email:    "visitor@example.test",
		Location: "Delft",
		Message:  "A historical note.",
		Created:  "2002-11-24 00:00:00.000Z",
		Updated:  "2002-11-24 00:00:00.000Z",
	}}, false); err != nil {
		t.Fatalf("import guestbook: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionGuestbook, "raaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("find imported entry: %v", err)
	}
	if got := record.GetString("email"); got != "" {
		t.Fatalf("imported email = %q, want empty", got)
	}
}

func TestPromoteProductionGuestbookEntries(t *testing.T) {
	app := newGuestbookSeedTestApp(t)
	entries := []sourceGuestbookEntry{
		{ID: "raaaaaaaaaaaaaa", Name: "Archive visitor", Email: "visitor@example.test", Location: "Delft", Message: "A historical note.", Created: "2002-11-24 00:00:00.000Z", Updated: "2002-11-24 00:00:00.000Z"},
		{ID: "rbbbbbbbbbbbbbb", Name: "", Email: "private@example.test", Location: "Leiden", Message: "No name.", Created: "2002-11-25 00:00:00.000Z", Updated: "2002-11-25 00:00:00.000Z"},
		{ID: "rcccccccccccccc", Name: "No date", Email: "private2@example.test", Location: "Haarlem", Message: "No date.", Created: "", Updated: ""},
		{ID: "rdddddddddddddd", Name: "Bad date", Email: "private3@example.test", Location: "Utrecht", Message: "Malformed date.", Created: "2002-13-99 00:00:00.000Z", Updated: "2002-13-99 00:00:00.000Z"},
	}
	for _, entry := range entries {
		seedGuestbookEntry(t, app, entry)
	}
	local := sourceGuestbookEntry{ID: "leeeeeeeeeeeeee", Name: "Local visitor", Email: "local@example.test", Location: "Ghent", Message: "Local note."}
	seedGuestbookEntry(t, app, local)

	sourcePath := writeGuestbookSource(t, entries)
	if err := PromoteProductionGuestbookEntries(app, sourcePath); err != nil {
		t.Fatalf("promote production guestbook: %v", err)
	}

	approved, err := app.FindRecordById(constants.CollectionGuestbook, entries[0].ID)
	if err != nil {
		t.Fatalf("find approved entry: %v", err)
	}
	if got := approved.GetString("moderation_state"); got != guestbookStateApproved {
		t.Fatalf("approved moderation state = %q, want %q", got, guestbookStateApproved)
	}
	if got := approved.GetString("retention_until"); got != "" {
		t.Fatalf("approved retention_until = %q, want empty", got)
	}

	private, err := app.FindRecordsByFilter(constants.CollectionGuestbook, "moderation_state = {:state}", "", 0, 0, dbx.Params{"state": guestbookStateUnreviewed})
	if err != nil {
		t.Fatalf("find unreviewed entries: %v", err)
	}
	if len(private) != 4 {
		t.Fatalf("unreviewed entries = %d, want 4", len(private))
	}
	for _, entry := range entries {
		record, err := app.FindRecordById(constants.CollectionGuestbook, entry.ID)
		if err != nil {
			t.Fatalf("find entry %s: %v", entry.ID, err)
		}
		if got := record.GetString("email"); got != "" {
			t.Fatalf("entry %s email = %q, want empty", entry.ID, got)
		}
	}
	localRecord, err := app.FindRecordById(constants.CollectionGuestbook, local.ID)
	if err != nil {
		t.Fatalf("find local entry: %v", err)
	}
	if got := localRecord.GetString("moderation_state"); got != guestbookStateUnreviewed {
		t.Fatalf("local moderation state = %q, want %q", got, guestbookStateUnreviewed)
	}
	if got := localRecord.GetString("email"); got != local.Email {
		t.Fatalf("local email = %q, want %q", got, local.Email)
	}
}

func newGuestbookSeedTestApp(t *testing.T) core.App {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir(), EncryptionEnv: "test-encryption-key"})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("Guestbook")
	collection.Id = constants.CollectionGuestbook
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.EmailField{Name: "email"},
		&core.TextField{Name: "location"},
		&core.TextField{Name: "message"},
		&core.SelectField{Name: "moderation_state", MaxSelect: 1, Values: []string{guestbookStateUnreviewed, guestbookStateApproved}},
		&core.DateField{Name: "retention_until"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save guestbook collection: %v", err)
	}

	return app
}

func seedGuestbookEntry(t *testing.T, app core.App, entry sourceGuestbookEntry) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(constants.CollectionGuestbook)
	if err != nil {
		t.Fatalf("find guestbook collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("id", entry.ID)
	record.Set("name", entry.Name)
	record.Set("email", entry.Email)
	record.Set("location", entry.Location)
	record.Set("message", entry.Message)
	record.Set("moderation_state", []string{guestbookStateUnreviewed})
	record.Set("retention_until", "2027-01-01 00:00:00.000Z")
	if err := app.Save(record); err != nil {
		t.Fatalf("save guestbook entry: %v", err)
	}
}

func writeGuestbookSource(t *testing.T, entries []sourceGuestbookEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wga-src.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open source database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE guestbook_entries (id TEXT, name TEXT, email TEXT, location TEXT, message TEXT, created TEXT, updated TEXT)`); err != nil {
		t.Fatalf("create guestbook source: %v", err)
	}
	for _, entry := range entries {
		if _, err := db.Exec(`INSERT INTO guestbook_entries VALUES (?, ?, ?, ?, ?, ?, ?)`, entry.ID, entry.Name, entry.Email, entry.Location, entry.Message, entry.Created, entry.Updated); err != nil {
			t.Fatalf("insert guestbook source entry: %v", err)
		}
	}

	return path
}
