package sitemap

import (
	"fmt"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestFetchArtworkAuthorsBatchesUniqueAuthorIDs(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(&core.TextField{Name: "name", Required: true})
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}
	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(&core.RelationField{Name: "author", CollectionId: artists.Id, MaxSelect: 1, Required: true})
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	artworkRecords := make([]*core.Record, 0, 101)
	for index := range 101 {
		author := core.NewRecord(artists)
		author.Set("name", fmt.Sprintf("Artist %d", index))
		if err := app.Save(author); err != nil {
			t.Fatalf("save author: %v", err)
		}
		artwork := core.NewRecord(artworks)
		artwork.Set("author", author.Id)
		if err := app.Save(artwork); err != nil {
			t.Fatalf("save artwork: %v", err)
		}
		artworkRecords = append(artworkRecords, artwork)
	}

	authors, err := fetchArtworkAuthors(app, artworkRecords)
	if err != nil {
		t.Fatalf("fetch artwork authors: %v", err)
	}
	if len(authors) != len(artworkRecords) {
		t.Fatalf("expected %d authors, got %d", len(artworkRecords), len(authors))
	}
}
