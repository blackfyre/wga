package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestArtistSelectionsMigrationCreatesAndRollsBack(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	// The relation fields reference the artists and artworks collections, so
	// those must exist before the selection collection is created.
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}

	if err := addArtistSelections(app); err != nil {
		t.Fatalf("add artist selections: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("art_selections")
	if err != nil {
		t.Fatalf("find art_selections: %v", err)
	}
	if collection.Name != "Art_selections" {
		t.Fatalf("collection name = %q, want Art_selections", collection.Name)
	}
	if collection.Type != core.CollectionTypeBase {
		t.Fatalf("collection type = %q, want base", collection.Type)
	}

	checks := []struct {
		name     string
		typ      string
		required bool
		hidden   bool
	}{
		{"artist", core.FieldTypeRelation, true, false},
		{"title", core.FieldTypeText, true, false},
		{"context", core.FieldTypeText, false, false},
		{"display_title", core.FieldTypeText, true, false},
		{"commentary", core.FieldTypeEditor, false, false},
		{"artworks", core.FieldTypeRelation, true, false},
		{"source_path", core.FieldTypeText, true, true},
		{"source_hash", core.FieldTypeText, true, true},
		{"content_hash", core.FieldTypeText, true, true},
		{"published", core.FieldTypeBool, true, false},
	}
	for _, check := range checks {
		field := collection.Fields.GetByName(check.name)
		if field == nil {
			t.Fatalf("missing field %q", check.name)
		}
		if field.Type() != check.typ {
			t.Fatalf("field %q type = %q, want %q", check.name, field.Type(), check.typ)
		}
		required, hidden := fieldRequiredHidden(field)
		if required != check.required {
			t.Fatalf("field %q required = %v, want %v", check.name, required, check.required)
		}
		if hidden != check.hidden {
			t.Fatalf("field %q hidden = %v, want %v", check.name, hidden, check.hidden)
		}
	}

	artist, ok := collection.Fields.GetByName("artist").(*core.RelationField)
	if !ok {
		t.Fatal("artist is not a relation field")
	}
	if artist.CollectionId != "artists" || artist.MinSelect != 1 || artist.MaxSelect != 1 {
		t.Fatalf("artist relation = %+v, want required single artists relation", artist)
	}

	artworks, ok := collection.Fields.GetByName("artworks").(*core.RelationField)
	if !ok {
		t.Fatal("artworks is not a relation field")
	}
	if artworks.CollectionId != "artworks" || artworks.MinSelect != 1 || artworks.MaxSelect != 1000 {
		t.Fatalf("artworks relation = %+v, want ordered artworks relation [1,1000]", artworks)
	}

	for name, rule := range map[string]*string{
		"listRule":   collection.ListRule,
		"viewRule":   collection.ViewRule,
		"createRule": collection.CreateRule,
		"updateRule": collection.UpdateRule,
		"deleteRule": collection.DeleteRule,
	} {
		if rule != nil {
			t.Fatalf("art_selections %s = %q, want nil (operationally private)", name, *rule)
		}
	}

	assertSelectionIndexes(t, collection)

	// Seed an authoritative editorial record before rollback so data retention
	// can be proven.
	seedSelectionRecord(t, app)

	if err := removeArtistSelections(app); err != nil {
		t.Fatalf("remove artist selections: %v", err)
	}

	rolledBack, err := app.FindCollectionByNameOrId("art_selections")
	if err != nil {
		t.Fatalf("rollback deleted the art_selections collection: %v", err)
	}
	for _, fieldName := range []string{
		"artist", "title", "context", "display_title", "commentary", "artworks",
		"source_path", "source_hash", "content_hash", "published",
	} {
		if rolledBack.Fields.GetByName(fieldName) == nil {
			t.Fatalf("authoritative field %q was removed by rollback", fieldName)
		}
	}
	if _, err := app.FindRecordById("art_selections", "rselect00000001"); err != nil {
		t.Fatalf("rollback deleted a seeded editorial selection record: %v", err)
	}

	joinedIndexes := strings.Join(rolledBack.Indexes, "\n")
	for _, index := range []string{artistSelectionsIndexSourcePath, artistSelectionsIndexPublished} {
		if strings.Contains(joinedIndexes, index) {
			t.Fatalf("rollback retained feature index %q", index)
		}
	}
}

func TestArtistSelectionsMigrationReAddsIndexesAfterRollback(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtistSelections(app); err != nil {
		t.Fatalf("add artist selections: %v", err)
	}
	if err := removeArtistSelections(app); err != nil {
		t.Fatalf("remove artist selections: %v", err)
	}
	if err := addArtistSelections(app); err != nil {
		t.Fatalf("re-add artist selections after rollback: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("art_selections")
	if err != nil {
		t.Fatalf("find art_selections: %v", err)
	}
	assertSelectionIndexes(t, collection)
}

func TestArtistSelectionsMigrationIsIdempotent(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtistSelections(app); err != nil {
		t.Fatalf("add artist selections: %v", err)
	}
	if err := addArtistSelections(app); err != nil {
		t.Fatalf("add artist selections again: %v", err)
	}
}

func assertSelectionIndexes(t *testing.T, collection *core.Collection) {
	t.Helper()
	joinedIndexes := strings.Join(collection.Indexes, "\n")
	for _, index := range []string{artistSelectionsIndexSourcePath, artistSelectionsIndexPublished} {
		if !strings.Contains(joinedIndexes, index) {
			t.Fatalf("missing index %q in %q", index, joinedIndexes)
		}
	}
}

func seedSelectionRecord(t *testing.T, app core.App) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("art_selections")
	if err != nil {
		t.Fatalf("find art_selections: %v", err)
	}

	record := core.NewRecord(collection)
	record.Id = "rselect00000001"
	record.Set("artist", []string{"rartist00000001"})
	record.Set("title", "Paintings")
	record.Set("display_title", "Paintings")
	record.Set("artworks", []string{"rwork0000000001"})
	record.Set("source_path", "html/a/artist/paintings/index.html")
	record.Set("source_hash", "source-hash")
	record.Set("content_hash", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	record.Set("published", true)
	if err := app.SaveNoValidate(record); err != nil {
		t.Fatalf("save seeded selection record: %v", err)
	}
}

func fieldRequiredHidden(field core.Field) (bool, bool) {
	switch f := field.(type) {
	case *core.TextField:
		return f.Required, f.Hidden
	case *core.EditorField:
		return f.Required, f.Hidden
	case *core.BoolField:
		return f.Required, f.Hidden
	case *core.RelationField:
		return f.Required, f.Hidden
	default:
		return false, false
	}
}
