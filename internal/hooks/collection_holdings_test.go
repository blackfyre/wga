package hooks

import (
	"testing"

	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestCollectionHoldingsCacheHooksInvalidateAfterMutations(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection("artists")
	artists.Id = "holding_artists"
	artists.MarkAsNew()
	artists.Fields.Add(&core.TextField{Id: "holding_artist_name", Name: "name", Required: true})
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	locations := core.NewBaseCollection("locations")
	locations.Id = "holding_locations"
	locations.MarkAsNew()
	locations.Fields.Add(&core.TextField{Id: "holding_location_name", Name: "name", Required: true})
	if err := app.Save(locations); err != nil {
		t.Fatalf("save locations collection: %v", err)
	}

	artworks := core.NewBaseCollection("artworks")
	artworks.Id = "holding_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "holding_artwork_title", Name: "title", Required: true},
		&core.RelationField{Id: "holding_artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Id: "holding_artwork_location", Name: "current_location_id", CollectionId: locations.Id, MaxSelect: 1},
		&core.BoolField{Id: "holding_artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	artworkCatalogueCacheHook(app)
	collectionHoldingsCacheHook(app)

	artist := core.NewRecord(artists)
	artist.Id = "holdingartist01"
	artist.Set("name", "Holding Artist")
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist: %v", err)
	}
	location := core.NewRecord(locations)
	location.Id = "holdingloc00001"
	location.Set("name", "Original Museum")
	if err := app.Save(location); err != nil {
		t.Fatalf("save location: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Original Museum", 0)

	artwork := core.NewRecord(artworks)
	artwork.Id = "holdingwork0001"
	artwork.Set("title", "Holding Work")
	artwork.Set("author", []string{artist.Id})
	artwork.Set("current_location_id", location.Id)
	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Original Museum", 1)

	location.Set("name", "Renamed Museum")
	if err := app.Save(location); err != nil {
		t.Fatalf("rename location: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Renamed Museum", 1)

	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("unpublish artwork: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Renamed Museum", 0)
	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("republish artwork: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Renamed Museum", 1)

	if err := app.Delete(artist); err != nil {
		t.Fatalf("delete artist: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Renamed Museum", 0)
	if _, err := app.DB().NewQuery("UPDATE artworks SET author = {:author} WHERE id = {:id}").Bind(dbx.Params{
		"author": `["` + artist.Id + `"]`,
		"id":     artwork.Id,
	}).Execute(); err != nil {
		t.Fatalf("restore persisted artwork relation: %v", err)
	}

	restoredArtist := core.NewRecord(artists)
	restoredArtist.Id = artist.Id
	restoredArtist.Set("name", "Restored Artist")
	if err := app.Save(restoredArtist); err != nil {
		t.Fatalf("restore artist: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Renamed Museum", 1)

	if err := app.Delete(artwork); err != nil {
		t.Fatalf("delete artwork: %v", err)
	}
	assertCollectionHolding(t, app, location.Id, "Renamed Museum", 0)
	if err := app.Delete(location); err != nil {
		t.Fatalf("delete location: %v", err)
	}
	assertCollectionHoldingMissing(t, app, location.Id)
}

func assertCollectionHolding(t *testing.T, app core.App, id string, label string, count int) {
	t.Helper()
	holdings, err := repositories.LoadCollectionHoldings(app)
	if err != nil {
		t.Fatalf("load collection holdings: %v", err)
	}
	for _, holding := range holdings {
		if holding.Value == id {
			if holding.Label != label || holding.Count != count {
				t.Fatalf("holding = %#v, want label %q and count %d", holding, label, count)
			}
			return
		}
	}
	t.Fatalf("holding %q missing from %#v", id, holdings)
}

func assertCollectionHoldingMissing(t *testing.T, app core.App, id string) {
	t.Helper()
	holdings, err := repositories.LoadCollectionHoldings(app)
	if err != nil {
		t.Fatalf("load collection holdings: %v", err)
	}
	for _, holding := range holdings {
		if holding.Value == id {
			t.Fatalf("holding %q remains after delete: %#v", id, holdings)
		}
	}
}
