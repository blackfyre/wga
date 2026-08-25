package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestItinerariesMigrationCreatesCollectionsAndIndexes(t *testing.T) {
	dataDir := t.TempDir()
	app := newMigrationTestApp(t, dataDir)
	defer func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	installMigrationArtworks(t, app)

	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries: %v", err)
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}
	for _, field := range []string{"owner", "status", "token", "title", "intro", "creator", "listed", "published", "expires_at"} {
		if itineraries.Fields.GetByName(field) == nil {
			t.Errorf("itineraries is missing field %q", field)
		}
	}
	assertIndexContains(t, itineraries.Indexes, "pbx_itinerary_token", "WHERE token != ''")
	assertIndexContains(t, itineraries.Indexes, "pbx_itinerary_draft_owner", "WHERE status = 'draft'")
	assertIndexContains(t, itineraries.Indexes, "pbx_itinerary_expiry", "expires_at")

	stops, err := app.FindCollectionByNameOrId("itinerary_stops")
	if err != nil {
		t.Fatalf("find itinerary_stops: %v", err)
	}
	for _, field := range []string{"itinerary", "artwork", "title", "position", "narration"} {
		if stops.Fields.GetByName(field) == nil {
			t.Errorf("itinerary_stops is missing field %q", field)
		}
	}
	assertIndexContains(t, stops.Indexes, "pbx_itinerary_stop_artwork", "artwork")
	assertUniqueIndexContains(t, stops.Indexes, "pbx_itinerary_stop_order", "position")

	assertClosedAPIRules(t, itineraries)
	assertClosedAPIRules(t, stops)

	// Idempotent: a second application is a no-op.
	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries (again): %v", err)
	}
}

func TestItinerariesDraftOwnerUniqueIndex(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	defer func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	installMigrationArtworks(t, app)
	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries: %v", err)
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}

	first := core.NewRecord(itineraries)
	first.Set("owner", "same-owner-digest")
	first.Set("status", "draft")
	if err := app.Save(first); err != nil {
		t.Fatalf("save first draft: %v", err)
	}

	second := core.NewRecord(itineraries)
	second.Set("owner", "same-owner-digest")
	second.Set("status", "draft")
	if err := app.Save(second); err == nil {
		t.Fatal("second draft for the same owner must violate the unique index")
	}
}

func TestItinerariesStopArtworkUniqueIndex(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	defer func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	installMigrationArtworks(t, app)
	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries: %v", err)
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}
	stopsCollection, err := app.FindCollectionByNameOrId("itinerary_stops")
	if err != nil {
		t.Fatalf("find itinerary_stops: %v", err)
	}

	itinerary := core.NewRecord(itineraries)
	itinerary.Set("owner", "owner-digest-0001")
	itinerary.Set("status", "draft")
	if err := app.Save(itinerary); err != nil {
		t.Fatalf("save itinerary: %v", err)
	}

	newStop := func() *core.Record {
		stop := core.NewRecord(stopsCollection)
		stop.Set("itinerary", itinerary.Id)
		stop.Set("artwork", "aw0000000000001")
		stop.Set("position", 0)
		return stop
	}

	if err := app.Save(newStop()); err != nil {
		t.Fatalf("save first stop: %v", err)
	}
	if err := app.Save(newStop()); err == nil {
		t.Fatal("duplicate stop for the same itinerary and artwork must violate the unique index")
	}
}

func TestItinerariesStopPositionUniqueIndex(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	defer func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	installMigrationArtworks(t, app)
	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries: %v", err)
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}
	stopsCollection, err := app.FindCollectionByNameOrId("itinerary_stops")
	if err != nil {
		t.Fatalf("find itinerary_stops: %v", err)
	}

	itinerary := core.NewRecord(itineraries)
	itinerary.Set("owner", "owner-digest-0002")
	itinerary.Set("status", "draft")
	if err := app.Save(itinerary); err != nil {
		t.Fatalf("save itinerary: %v", err)
	}

	newStop := func(position int) *core.Record {
		stop := core.NewRecord(stopsCollection)
		stop.Set("itinerary", itinerary.Id)
		stop.Set("artwork", "aw0000000000001")
		stop.Set("position", position)
		return stop
	}

	if err := app.Save(newStop(0)); err != nil {
		t.Fatalf("save first stop: %v", err)
	}
	// A second stop at the same position for the same itinerary must be
	// rejected by the unique (itinerary, position) index.
	duplicate := core.NewRecord(stopsCollection)
	duplicate.Set("itinerary", itinerary.Id)
	duplicate.Set("artwork", "aw0000000000002")
	duplicate.Set("position", 0)
	if err := app.Save(duplicate); err == nil {
		t.Fatal("second stop at the same position must violate the unique index")
	}
}

func TestItinerariesMigrationRollbackPreservesData(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	defer func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	installMigrationArtworks(t, app)

	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries: %v", err)
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}
	draft := core.NewRecord(itineraries)
	draft.Set("owner", "owner-digest-0003")
	draft.Set("status", "draft")
	if err := app.Save(draft); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	// Rollback refuses to run and preserves both collections and their data.
	if err := removeItineraries(app); err == nil {
		t.Fatal("removeItineraries must refuse to run")
	}

	if _, err := app.FindCollectionByNameOrId("itineraries"); err != nil {
		t.Error("itineraries collection must be preserved after refused rollback")
	}
	if _, err := app.FindCollectionByNameOrId("itinerary_stops"); err != nil {
		t.Error("itinerary_stops collection must be preserved after refused rollback")
	}
	if _, err := app.FindRecordById("itineraries", draft.Id); err != nil {
		t.Error("itinerary records must be preserved after refused rollback")
	}
}

func installMigrationArtworks(t *testing.T, app *core.BaseApp) {
	t.Helper()

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks collection: %v", err)
	}

	artwork := core.NewRecord(artworks)
	artwork.Id = "aw0000000000001"
	artwork.Set("title", "Work")
	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("create artwork: %v", err)
	}

	second := core.NewRecord(artworks)
	second.Id = "aw0000000000002"
	second.Set("title", "Work Two")
	second.Set("published", true)
	if err := app.Save(second); err != nil {
		t.Fatalf("create second artwork: %v", err)
	}
}

func assertIndexContains(t *testing.T, indexes []string, name string, fragment string) {
	t.Helper()
	for _, index := range indexes {
		if strings.Contains(index, name) && strings.Contains(index, fragment) {
			return
		}
	}
	t.Errorf("missing index %q containing %q", name, fragment)
}

func assertUniqueIndexContains(t *testing.T, indexes []string, name string, fragment string) {
	t.Helper()
	for _, index := range indexes {
		if strings.Contains(index, name) && strings.Contains(index, fragment) {
			if !strings.Contains(strings.ToUpper(index), "UNIQUE") {
				t.Errorf("index %q must be unique", name)
			}
			return
		}
	}
	t.Errorf("missing unique index %q containing %q", name, fragment)
}

func assertClosedAPIRules(t *testing.T, collection *core.Collection) {
	t.Helper()
	if collection.ListRule != nil {
		t.Errorf("%s ListRule must be closed (nil)", collection.Name)
	}
	if collection.ViewRule != nil {
		t.Errorf("%s ViewRule must be closed (nil)", collection.Name)
	}
	if collection.CreateRule != nil {
		t.Errorf("%s CreateRule must be closed (nil)", collection.Name)
	}
	if collection.UpdateRule != nil {
		t.Errorf("%s UpdateRule must be closed (nil)", collection.Name)
	}
	if collection.DeleteRule != nil {
		t.Errorf("%s DeleteRule must be closed (nil)", collection.Name)
	}
}
