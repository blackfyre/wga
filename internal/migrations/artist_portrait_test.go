package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestAddArtistPortraitAddsFieldToExistingCollection(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	if err := addArtistPortrait(app); err != nil {
		t.Fatalf("add artist portrait: %v", err)
	}
	updatedArtists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find updated artists collection: %v", err)
	}
	if _, ok := updatedArtists.Fields.GetByName("portrait").(*core.FileField); !ok {
		t.Fatal("expected artist portrait file field")
	}
	if err := addArtistPortrait(app); err != nil {
		t.Fatalf("repeat artist portrait migration: %v", err)
	}
}
