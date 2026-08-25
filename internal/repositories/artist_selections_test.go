package repositories

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newArtistSelectionsTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection("Artists")
	artists.Id = constants.CollectionArtists
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = constants.CollectionArtworks
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	selections := core.NewBaseCollection("Art_selections")
	selections.Id = constants.CollectionSelections
	selections.MarkAsNew()
	selections.Fields.Add(
		&core.RelationField{Name: "artist", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 1, Required: true},
		&core.TextField{Name: "title", Required: true},
		&core.TextField{Name: "context"},
		&core.TextField{Name: "display_title", Required: true},
		&core.EditorField{Name: "commentary"},
		&core.RelationField{Name: "artworks", CollectionId: artworks.Id, MinSelect: 1, MaxSelect: 1000, Required: true},
		&core.TextField{Name: "source_path", Required: true, Hidden: true},
		&core.TextField{Name: "source_hash", Required: true, Hidden: true},
		&core.BoolField{Name: "published", Required: true},
	)
	if err := app.Save(selections); err != nil {
		t.Fatalf("save selections: %v", err)
	}

	return app
}

func saveSelectionArtist(t *testing.T, app *tests.TestApp, id, name string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtists)
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("slug", name)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist %s: %v", id, err)
	}
}

func saveSelectionWork(t *testing.T, app *tests.TestApp, id string, authors []string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", "Work "+id)
	record.Set("author", authors)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork %s: %v", id, err)
	}
}

func saveSelectionRecord(t *testing.T, app *tests.TestApp, id, artistID, displayTitle string, artworkIDs []string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(constants.CollectionSelections)
	if err != nil {
		t.Fatalf("find selections: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("artist", []string{artistID})
	record.Set("title", displayTitle)
	record.Set("display_title", displayTitle)
	record.Set("artworks", artworkIDs)
	record.Set("source_path", "html/a/artist/"+id+"/index.html")
	record.Set("source_hash", "source-hash")
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save selection %s: %v", id, err)
	}
}

func TestArtistSelectionsRepositoryFindsPublishedOwnedSelection(t *testing.T) {
	app := newArtistSelectionsTestApp(t)
	saveSelectionArtist(t, app, "artistone000001", "Dürer", true)
	saveSelectionArtist(t, app, "artisttwo000001", "Other", true)
	saveSelectionWork(t, app, "workone00000001", []string{"artistone000001"}, true)
	saveSelectionRecord(t, app, "rselect00000001", "artistone000001", "Paintings", []string{"workone00000001"}, true)
	saveSelectionRecord(t, app, "rselect00000003", "artisttwo000001", "Foreign", []string{"workone00000001"}, true)

	repo := NewArtistSelectionsRepository(app)

	owned, err := repo.FindPublishedSelection("artistone000001", "rselect00000001")
	if err != nil {
		t.Fatalf("find owned selection: %v", err)
	}
	if owned.GetString("display_title") != "Paintings" {
		t.Errorf("selection = %q, want Paintings", owned.GetString("display_title"))
	}

	for _, id := range []string{"rselect00000003", "rmissing0000001"} {
		if _, err := repo.FindPublishedSelection("artistone000001", id); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("selection %q error = %v, want sql.ErrNoRows", id, err)
		}
	}
}

func TestArtistSelectionsRepositoryListsPublishedArtistScopedBounded(t *testing.T) {
	app := newArtistSelectionsTestApp(t)
	saveSelectionArtist(t, app, "artistone000001", "Dürer", true)
	saveSelectionArtist(t, app, "artisttwo000001", "Other", true)
	saveSelectionWork(t, app, "workone00000001", []string{"artistone000001"}, true)

	for _, id := range []string{"rselect00000001", "rselect00000002", "rselect00000003"} {
		saveSelectionRecord(t, app, id, "artistone000001", "Selection "+id, []string{"workone00000001"}, true)
	}
	saveSelectionRecord(t, app, "rselect00000005", "artisttwo000001", "Foreign", []string{"workone00000001"}, true)

	repo := NewArtistSelectionsRepository(app)

	count, err := repo.CountPublishedSelections("artistone000001")
	if err != nil {
		t.Fatalf("count selections: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3 (foreign excluded)", count)
	}

	all, err := repo.ListPublishedSelections("artistone000001", 0)
	if err != nil {
		t.Fatalf("list selections: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("selections = %d, want 3", len(all))
	}

	bounded, err := repo.ListPublishedSelections("artistone000001", 2)
	if err != nil {
		t.Fatalf("list bounded selections: %v", err)
	}
	if len(bounded) != 2 {
		t.Errorf("bounded selections = %d, want 2", len(bounded))
	}
}

func TestArtistSelectionsRepositoryPreservesArtworkOrder(t *testing.T) {
	app := newArtistSelectionsTestApp(t)
	saveSelectionArtist(t, app, "artistone000001", "Dürer", true)
	for _, id := range []string{"workone00000001", "worktwo00000001", "workthree000001"} {
		saveSelectionWork(t, app, id, []string{"artistone000001"}, true)
	}
	saveSelectionRecord(t, app, "rselect00000001", "artistone000001", "Paintings",
		[]string{"workthree000001", "workone00000001", "worktwo00000001"}, true)

	repo := NewArtistSelectionsRepository(app)

	selection, err := repo.FindPublishedSelection("artistone000001", "rselect00000001")
	if err != nil {
		t.Fatalf("find selection: %v", err)
	}
	works, err := repo.ListSelectionArtworks("artistone000001", selection)
	if err != nil {
		t.Fatalf("list selection artworks: %v", err)
	}
	got := make([]string, len(works))
	for i, work := range works {
		got[i] = work.Id
	}
	if want := []string{"workthree000001", "workone00000001", "worktwo00000001"}; !reflect.DeepEqual(got, want) {
		t.Errorf("artwork order = %v, want %v", got, want)
	}
}

// TestArtistSelectionsRepositoryExcludesForeignArtwork proves that a malformed
// operational record listing another artist's published work alongside its own
// never renders the foreign work. The read-model requires both publication and
// ownership by the selection artist, so the foreign record is filtered out.
func TestArtistSelectionsRepositoryExcludesForeignArtwork(t *testing.T) {
	app := newArtistSelectionsTestApp(t)
	saveSelectionArtist(t, app, "artistone000001", "Dürer", true)
	saveSelectionArtist(t, app, "artisttwo000001", "Other", true)
	saveSelectionWork(t, app, "workowned000001", []string{"artistone000001"}, true)
	saveSelectionWork(t, app, "workforeign0001", []string{"artisttwo000001"}, true)
	saveSelectionRecord(t, app, "rselect00000001", "artistone000001", "Paintings",
		[]string{"workforeign0001", "workowned000001"}, true)

	repo := NewArtistSelectionsRepository(app)

	selection, err := repo.FindPublishedSelection("artistone000001", "rselect00000001")
	if err != nil {
		t.Fatalf("find selection: %v", err)
	}
	works, err := repo.ListSelectionArtworks("artistone000001", selection)
	if err != nil {
		t.Fatalf("list selection artworks: %v", err)
	}

	got := make([]string, len(works))
	for i, work := range works {
		got[i] = work.Id
	}
	if want := []string{"workowned000001"}; !reflect.DeepEqual(got, want) {
		t.Errorf("artworks = %v, want only owned %v (foreign published work excluded)", got, want)
	}
}
