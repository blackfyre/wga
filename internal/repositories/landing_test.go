package repositories

import (
	"strconv"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestLandingRepositoryEligibleArtworkQueriesAreBoundedAndStable(t *testing.T) {
	app := newLandingRepositoryTestApp(t)
	createLandingRepositoryArtist(t, app, "artistpublic001", true)
	createLandingRepositoryArtist(t, app, "artistprivate01", false)

	for index := 1; index <= 6; index++ {
		value := strconv.Itoa(index)
		createdValue := value
		if index == 6 {
			createdValue = "5"
		}
		id := "eligiblework00" + value
		createLandingRepositoryArtwork(t, app, id, "artistpublic001", true, "2026-01-0"+createdValue+" 00:00:00.000Z")
	}
	createLandingRepositoryArtwork(t, app, "hiddenwork00001", "artistpublic001", false, "2026-01-07 00:00:00.000Z")
	createLandingRepositoryArtwork(t, app, "privatework0000", "artistprivate01", true, "2026-01-08 00:00:00.000Z")

	repo := NewLandingRepository(app)
	count, err := repo.CountEligibleArtworks()
	if err != nil {
		t.Fatalf("count eligible artworks: %v", err)
	}
	if count != 6 {
		t.Fatalf("eligible artwork count = %d, want 6", count)
	}

	selected, err := repo.FindEligibleArtworkByOffset(2)
	if err != nil {
		t.Fatalf("find eligible artwork by offset: %v", err)
	}
	if selected == nil || selected.Artwork.Id != "eligiblework003" {
		t.Fatalf("artwork at offset 2 = %#v, want eligiblework003", selected)
	}

	for _, offset := range []int{-1, 6} {
		artwork, err := repo.FindEligibleArtworkByOffset(offset)
		if err != nil {
			t.Fatalf("find artwork at invalid offset %d: %v", offset, err)
		}
		if artwork != nil {
			t.Errorf("artwork at invalid offset %d = %#v, want nil", offset, artwork)
		}
	}

	recent, err := repo.ListRecentEligibleArtworks()
	if err != nil {
		t.Fatalf("list recent eligible artworks: %v", err)
	}
	if len(recent) != recentEligibleArtworkLimit {
		t.Fatalf("recent eligible artworks = %d, want %d", len(recent), recentEligibleArtworkLimit)
	}
	if recent[0].Artwork.Id != "eligiblework005" || recent[1].Artwork.Id != "eligiblework006" || recent[3].Artwork.Id != "eligiblework003" {
		t.Errorf("recent eligible artwork IDs = %q, %q, %q; want eligiblework005, eligiblework006, eligiblework003", recent[0].Artwork.Id, recent[1].Artwork.Id, recent[3].Artwork.Id)
	}
}

func TestLandingRepositoryExcludesBlankIdentityArtworks(t *testing.T) {
	app := newLandingRepositoryTestApp(t)
	createLandingRepositoryArtistWithIdentity(t, app, "artistpublic001", "FILING, Public", true)
	createLandingRepositoryArtistWithIdentity(t, app, "artistblank0001", "", true)

	createLandingRepositoryArtwork(t, app, "eligiblework001", "artistpublic001", true, "2026-01-01 00:00:00.000Z")
	createLandingRepositoryArtwork(t, app, "blankwork000001", "artistblank0001", true, "2026-01-02 00:00:00.000Z")

	repo := NewLandingRepository(app)

	count, err := repo.CountEligibleArtworks()
	if err != nil {
		t.Fatalf("count eligible artworks: %v", err)
	}
	if count != 1 {
		t.Fatalf("eligible artwork count = %d, want 1 (blank identity excluded)", count)
	}

	selected, err := repo.FindEligibleArtworkByOffset(0)
	if err != nil {
		t.Fatalf("find eligible artwork by offset: %v", err)
	}
	if selected == nil || selected.Artwork.Id != "eligiblework001" {
		t.Fatalf("artwork at offset 0 = %#v, want eligiblework001", selected)
	}
	if selected != nil && selected.Artist.GetString("filing_name") != "FILING, Public" {
		t.Errorf("artist filing name = %q, want %q", selected.Artist.GetString("filing_name"), "FILING, Public")
	}

	recent, err := repo.ListRecentEligibleArtworks()
	if err != nil {
		t.Fatalf("list recent eligible artworks: %v", err)
	}
	if len(recent) != 1 || recent[0].Artwork.Id != "eligiblework001" {
		t.Errorf("recent eligible artworks = %v, want only eligiblework001", recent)
	}
}

func newLandingRepositoryTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset test app: %v", err)
		}
	})

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_filing_name", Name: "filing_name"},
		&core.TextField{Id: "artist_short_name", Name: "short_name"},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "test_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.TextField{Id: "artwork_year", Name: "year"},
		&core.NumberField{Id: "artwork_date_end", Name: "date_end"},
		&core.TextField{Id: "artwork_image", Name: "image"},
		&core.NumberField{Id: "artwork_image_width", Name: "image_width"},
		&core.DateField{Id: "artwork_created", Name: "created"},
		&core.BoolField{Id: "artwork_published", Name: "published"},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	return app
}

func createLandingRepositoryArtist(t *testing.T, app *pocketbase.PocketBase, id string, published bool) {
	t.Helper()
	createLandingRepositoryArtistWithIdentity(t, app, id, id, published)
}

func createLandingRepositoryArtistWithIdentity(t *testing.T, app *pocketbase.PocketBase, id string, filingName string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", id)
	record.Set("filing_name", filingName)
	record.Set("short_name", filingName)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist: %v", err)
	}
}

func createLandingRepositoryArtwork(t *testing.T, app *pocketbase.PocketBase, id string, author string, published bool, created string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", id)
	record.Set("author", []string{author})
	record.Set("published", published)
	record.Set("created", created)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
}
