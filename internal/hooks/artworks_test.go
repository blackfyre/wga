package hooks

import (
	"testing"

	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestArtworkAvailabilityCacheHookInvalidatesAfterMutations(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection("artists")
	artists.Id = "hook_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_filing", Name: "filing_name"},
		&core.TextField{Id: "artist_short", Name: "short_name"},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection("artworks")
	artworks.Id = "hook_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Id: "artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	artworkAvailabilityCacheHook(app)

	artist := core.NewRecord(artists)
	artist.Id = "hookartist00001"
	artist.Set("name", "Hook Artist")
	artist.Set("filing_name", "Hook Artist")
	artist.Set("short_name", "Hook Artist")
	artist.Set("published", true)
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist: %v", err)
	}

	repo := repositories.NewArtistIndexRepository(app)
	assertHookArtistAvailability(t, repo, false)

	artwork := core.NewRecord(artworks)
	artwork.Id = "hookartwork0001"
	artwork.Set("title", "Hook Work")
	artwork.Set("author", []string{artist.Id})
	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
	assertHookArtistAvailability(t, repo, true)

	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("unpublish artwork: %v", err)
	}
	assertHookArtistAvailability(t, repo, false)

	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("republish artwork: %v", err)
	}
	assertHookArtistAvailability(t, repo, true)

	if err := app.Delete(artwork); err != nil {
		t.Fatalf("delete artwork: %v", err)
	}
	assertHookArtistAvailability(t, repo, false)
}

func assertHookArtistAvailability(t *testing.T, repo *repositories.ArtistIndexRepository, want bool) {
	t.Helper()
	artists, err := repo.ListArtists(repositories.ArtistIndexFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("artists = %d, want 1", len(artists))
	}
	if artists[0].Available != want {
		t.Fatalf("availability = %t, want %t", artists[0].Available, want)
	}
}
