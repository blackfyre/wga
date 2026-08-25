package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestGuestbookModerationBackfillsLegacyEmailsWithoutRevalidating(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("Guestbook")
	collection.Id = "guestbook"
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.EmailField{Name: "email"},
		&core.TextField{Name: "location"},
		&core.TextField{Name: "message"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save guestbook collection: %v", err)
	}

	emptyEmail := core.NewRecord(collection)
	emptyEmail.Set("name", "Empty email visitor")
	emptyEmail.Set("email", "")
	emptyEmail.Set("location", "Delft")
	emptyEmail.Set("message", "Empty email note")
	if err := app.Save(emptyEmail); err != nil {
		t.Fatalf("save empty-email legacy entry: %v", err)
	}

	invalidEmail := core.NewRecord(collection)
	invalidEmail.Set("name", "Invalid email visitor")
	invalidEmail.Set("email", "not-an-email")
	invalidEmail.Set("location", "Leiden")
	invalidEmail.Set("message", "Invalid email note")
	if err := app.SaveNoValidate(invalidEmail); err != nil {
		t.Fatalf("save invalid-email legacy entry: %v", err)
	}

	if err := addGuestbookModeration(app); err != nil {
		t.Fatalf("add guestbook moderation: %v", err)
	}

	for _, entry := range []*core.Record{emptyEmail, invalidEmail} {
		got, err := app.FindRecordById("guestbook", entry.Id)
		if err != nil {
			t.Fatalf("find legacy entry %s: %v", entry.Id, err)
		}
		if got.GetString("moderation_state") != "unreviewed" {
			t.Fatalf("legacy moderation state = %q, want unreviewed", got.GetString("moderation_state"))
		}
		if got.GetString("retention_until") == "" {
			t.Fatal("legacy entry has no retention deadline")
		}
	}
	emptyAfter, err := app.FindRecordById("guestbook", emptyEmail.Id)
	if err != nil {
		t.Fatalf("find empty-email entry: %v", err)
	}
	for field, want := range map[string]string{"name": "Empty email visitor", "email": "", "location": "Delft", "message": "Empty email note"} {
		if got := emptyAfter.GetString(field); got != want {
			t.Fatalf("empty-email %s = %q, want %q (bytes preserved)", field, got, want)
		}
	}
	invalidAfter, err := app.FindRecordById("guestbook", invalidEmail.Id)
	if err != nil {
		t.Fatalf("find invalid-email entry: %v", err)
	}
	for field, want := range map[string]string{"name": "Invalid email visitor", "email": "not-an-email", "location": "Leiden", "message": "Invalid email note"} {
		if got := invalidAfter.GetString(field); got != want {
			t.Fatalf("invalid-email %s = %q, want %q (bytes preserved)", field, got, want)
		}
	}
	approved, err := app.FindRecordsByFilter("guestbook", "moderation_state = 'approved'", "", 0, 0)
	if err != nil {
		t.Fatalf("query approved legacy entries: %v", err)
	}
	if len(approved) != 0 {
		t.Fatalf("migration exposed %d legacy entries as approved", len(approved))
	}

	// Re-running the migration must not change or re-expose anything.
	if err := addGuestbookModeration(app); err != nil {
		t.Fatalf("re-run guestbook moderation: %v", err)
	}
	reRun, err := app.FindRecordById("guestbook", emptyEmail.Id)
	if err != nil {
		t.Fatalf("find entry after re-run: %v", err)
	}
	if reRun.GetString("moderation_state") != "unreviewed" || reRun.GetString("email") != "" {
		t.Fatalf("re-run changed legacy row: state %q email %q", reRun.GetString("moderation_state"), reRun.GetString("email"))
	}

	// Rollback keeps moderation and retention evidence and the visitor bytes.
	if err := removeGuestbookModeration(app); err != nil {
		t.Fatalf("remove guestbook moderation: %v", err)
	}
	rolledBack, err := app.FindCollectionByNameOrId("guestbook")
	if err != nil {
		t.Fatalf("find rolled-back guestbook: %v", err)
	}
	for _, field := range []string{"moderation_state", "retention_until"} {
		if rolledBack.Fields.GetByName(field) == nil {
			t.Fatalf("authoritative field %q was removed by rollback", field)
		}
	}
	rolledBackRecord, err := app.FindRecordById("guestbook", invalidEmail.Id)
	if err != nil {
		t.Fatalf("find rolled-back invalid-email record: %v", err)
	}
	if rolledBackRecord.GetString("moderation_state") != "unreviewed" {
		t.Fatalf("rollback lost moderation evidence: %q", rolledBackRecord.GetString("moderation_state"))
	}
	if rolledBackRecord.GetString("email") != "not-an-email" {
		t.Fatalf("rollback changed visitor email bytes: %q", rolledBackRecord.GetString("email"))
	}
}

func TestGuestbookModerationMigrationBackfillsAndRollsBack(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("Guestbook")
	collection.Id = "guestbook"
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "location"},
		&core.TextField{Name: "message"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save guestbook collection: %v", err)
	}

	legacy := core.NewRecord(collection)
	legacy.Set("name", "Legacy visitor")
	legacy.Set("message", "Historical note")
	if err := app.Save(legacy); err != nil {
		t.Fatalf("save legacy guestbook entry: %v", err)
	}

	if err := addGuestbookModeration(app); err != nil {
		t.Fatalf("add guestbook moderation: %v", err)
	}
	updated, err := app.FindCollectionByNameOrId("guestbook")
	if err != nil {
		t.Fatalf("find updated guestbook: %v", err)
	}
	for _, field := range []string{"moderation_state", "retention_until"} {
		if updated.Fields.GetByName(field) == nil {
			t.Fatalf("missing field %q", field)
		}
	}
	got, err := app.FindRecordById("guestbook", legacy.Id)
	if err != nil {
		t.Fatalf("find legacy entry: %v", err)
	}
	if got.GetString("moderation_state") != "unreviewed" {
		t.Fatalf("legacy moderation state = %q", got.GetString("moderation_state"))
	}
	if got.GetString("retention_until") == "" {
		t.Fatal("legacy unreviewed entry has no retention deadline")
	}
	approved, err := app.FindRecordsByFilter("guestbook", "moderation_state = 'approved'", "", 0, 0)
	if err != nil {
		t.Fatalf("query approved legacy entries: %v", err)
	}
	if len(approved) != 0 {
		t.Fatalf("migration exposed %d unreviewed legacy entries", len(approved))
	}
	got.Set("moderation_state", "approved")
	got.Set("retention_until", "")
	if err := app.Save(got); err != nil {
		t.Fatalf("approve legacy entry: %v", err)
	}
	approved, err = app.FindRecordsByFilter("guestbook", "moderation_state = 'approved'", "", 0, 0)
	if err != nil || len(approved) != 1 {
		t.Fatalf("explicitly approved legacy entries = %d, err=%v", len(approved), err)
	}
	joinedIndexes := strings.Join(updated.Indexes, "\n")
	for _, index := range []string{guestbookModerationIndex, guestbookRetentionIndex} {
		if !strings.Contains(joinedIndexes, index) {
			t.Fatalf("missing index %q", index)
		}
	}

	if err := removeGuestbookModeration(app); err != nil {
		t.Fatalf("remove guestbook moderation: %v", err)
	}
	rolledBack, err := app.FindCollectionByNameOrId("guestbook")
	if err != nil {
		t.Fatalf("find rolled-back guestbook: %v", err)
	}
	for _, field := range []string{"moderation_state", "retention_until"} {
		if rolledBack.Fields.GetByName(field) == nil {
			t.Fatalf("authoritative field %q was removed by rollback", field)
		}
	}
	rolledBackRecord, err := app.FindRecordById("guestbook", legacy.Id)
	if err != nil {
		t.Fatalf("find rolled-back legacy record: %v", err)
	}
	if rolledBackRecord.GetString("moderation_state") != "approved" {
		t.Fatalf("rollback lost moderation evidence: %q", rolledBackRecord.GetString("moderation_state"))
	}
	joinedIndexes = strings.Join(rolledBack.Indexes, "\n")
	for _, index := range []string{guestbookModerationIndex, guestbookRetentionIndex} {
		if strings.Contains(joinedIndexes, index) {
			t.Fatalf("rollback retained feature index %q", index)
		}
	}
}
