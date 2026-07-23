package migrations

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSyntheticSourceFieldsCanBeReappliedAfterRollback(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	for _, item := range []struct {
		name string
		id   string
	}{
		{name: "Artists", id: "artists"},
		{name: "Artworks", id: "artworks"},
		{name: "Glossary", id: "glossary"},
		{name: "Music_song", id: "music_song"},
	} {
		collection := core.NewBaseCollection(item.name)
		collection.Id = item.id
		collection.MarkAsNew()
		if err := app.Save(collection); err != nil {
			t.Fatalf("create %s collection: %v", item.id, err)
		}
	}

	if err := addSyntheticSourceFields(app); err != nil {
		t.Fatalf("add synthetic source fields: %v", err)
	}
	if err := removeSyntheticSourceFields(app); err != nil {
		t.Fatalf("remove synthetic source fields: %v", err)
	}

	for _, name := range []string{"biographies", "biography_links", "professions", "source_attributions"} {
		_, err := app.FindCollectionByNameOrId(name)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected %s collection to be removed, got %v", name, err)
		}
	}

	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	if artists.Fields.GetByName("source_path") != nil {
		t.Fatal("expected source_path field to be removed")
	}

	if err := addSyntheticSourceFields(app); err != nil {
		t.Fatalf("reapply synthetic source fields: %v", err)
	}
}
