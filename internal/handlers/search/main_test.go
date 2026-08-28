package search

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func newSearchTestApp(t *testing.T) *pocketbase.PocketBase {
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

	saveSearchCollection(t, app, "schools",
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
	)

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "profession"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.RelationField{Name: "school", CollectionId: "schools", MinSelect: 0, MaxSelect: 10},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: "artists", MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Name: "school", CollectionId: "schools", MinSelect: 0, MaxSelect: 10},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	return app
}

func saveSearchCollection(t *testing.T, app *pocketbase.PocketBase, id string, fields ...core.Field) {
	t.Helper()
	collection := core.NewBaseCollection(id)
	collection.Id = id
	collection.MarkAsNew()
	collection.Fields.Add(fields...)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save %s collection: %v", id, err)
	}
}

func saveSearchArtist(t *testing.T, app *pocketbase.PocketBase, id string, fields map[string]any) {
	t.Helper()
	saveSearchRecord(t, app, "artists", id, fields)
}

func saveSearchWork(t *testing.T, app *pocketbase.PocketBase, id string, fields map[string]any) {
	t.Helper()
	saveSearchRecord(t, app, "artworks", id, fields)
}

func saveSearchRecord(t *testing.T, app *pocketbase.PocketBase, collection string, id string, fields map[string]any) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s collection: %v", collection, err)
	}
	record := core.NewRecord(coll)
	record.Id = id
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

// TestSearchViewUsesFilingNameNotLegacyName proves the artist result set is
// filtered, ordered, and labelled by filing_name while its href is constructed
// from the legacy name. A divergent legacy name must never surface as identity.
func TestSearchViewUsesFilingNameNotLegacyName(t *testing.T) {
	app := newSearchTestApp(t)
	saveSearchArtist(t, app, "artdiverg100000", map[string]any{
		"name": "Legacy Name", "filing_name": "Filing, Name", "short_name": "Short Name",
		"year_of_birth": 1600, "year_of_death": 1670, "published": true,
	})
	saveSearchWork(t, app, "workdiverg10000", map[string]any{
		"title": "Divergent Work", "author": []string{"artdiverg100000"}, "published": true,
	})

	view, err := searchView(app, "filing")
	if err != nil {
		t.Fatalf("search by filing: %v", err)
	}
	if len(view.Artists) != 1 {
		t.Fatalf("artists = %d, want 1 (filing-name match)", len(view.Artists))
	}
	if got := view.Artists[0].Name; got != "Filing, Name" {
		t.Errorf("artist label = %q, want filing form %q", got, "Filing, Name")
	}
	if !strings.Contains(view.Artists[0].Href, "legacy-name-artdiverg100000") {
		t.Errorf("artist href = %q, want legacy-name-derived URL", view.Artists[0].Href)
	}

	// The legacy display name is not a search identity source: it must not match.
	legacy, err := searchView(app, "legacy")
	if err != nil {
		t.Fatalf("search by legacy name: %v", err)
	}
	if len(legacy.Artists) != 0 {
		t.Fatalf("legacy-name search artists = %d, want 0 (legacy name is not searchable)", len(legacy.Artists))
	}
}

// TestSearchViewWorkLabelUsesFilingNameAndLegacyURL proves work results label
// their author with filing_name while the work href keeps the legacy name.
func TestSearchViewWorkLabelUsesFilingNameAndLegacyURL(t *testing.T) {
	app := newSearchTestApp(t)
	saveSearchArtist(t, app, "artdiverg100000", map[string]any{
		"name": "Legacy Name", "filing_name": "Filing, Name", "short_name": "Short Name", "published": true,
	})
	saveSearchWork(t, app, "workdiverg10000", map[string]any{
		"title": "Divergent Work", "author": []string{"artdiverg100000"}, "published": true,
	})

	view, err := searchView(app, "divergent")
	if err != nil {
		t.Fatalf("search work title: %v", err)
	}
	if len(view.Works) != 1 {
		t.Fatalf("works = %d, want 1", len(view.Works))
	}
	if got := view.Works[0].Artist; got != "Filing, Name" {
		t.Errorf("work artist label = %q, want filing form %q", got, "Filing, Name")
	}
	if !strings.Contains(view.Works[0].Href, "legacy-name-artdiverg100000") {
		t.Errorf("work href = %q, want legacy-name-derived URL", view.Works[0].Href)
	}
}

// TestSearchViewExcludesBlankIdentity proves artists missing either
// authoritative identity field are denied from the global result set.
func TestSearchViewExcludesBlankIdentity(t *testing.T) {
	app := newSearchTestApp(t)
	saveSearchArtist(t, app, "artcomplete0000", map[string]any{
		"name": "Complete Legacy", "filing_name": "Complete, Filing", "short_name": "Complete", "published": true,
	})
	saveSearchArtist(t, app, "artblankfil0000", map[string]any{
		"name": "Blank Filing Legacy", "filing_name": "", "short_name": "Blank Filing", "published": true,
	})
	saveSearchArtist(t, app, "artblanksho0000", map[string]any{
		"name": "Blank Short Legacy", "filing_name": "Blank, Short", "short_name": "", "published": true,
	})

	view, err := searchView(app, "")
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if view.ArtistTotal != 1 {
		t.Errorf("artist total = %d, want 1 (blank identity excluded)", view.ArtistTotal)
	}
	if len(view.Artists) != 1 || view.Artists[0].Name != "Complete, Filing" {
		t.Fatalf("artists = %+v, want only the complete artist", view.Artists)
	}

	// A blank-identity artist must not match even its legacy name.
	blank, err := searchView(app, "blank")
	if err != nil {
		t.Fatalf("search blank: %v", err)
	}
	if len(blank.Artists) != 0 {
		t.Fatalf("blank-identity artists surfaced: %+v", blank.Artists)
	}
}

// TestSearchViewSortsArtistsByFilingName proves artist ordering follows
// filing_name, not the legacy name.
func TestSearchViewSortsArtistsByFilingName(t *testing.T) {
	app := newSearchTestApp(t)
	saveSearchArtist(t, app, "artzeta10000000", map[string]any{
		"name": "Alpha Legacy", "filing_name": "Zeta, Filing", "short_name": "Zeta", "published": true,
	})
	saveSearchArtist(t, app, "artalpha1000000", map[string]any{
		"name": "Zeta Legacy", "filing_name": "Alpha, Filing", "short_name": "Alpha", "published": true,
	})

	view, err := searchView(app, "")
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(view.Artists) != 2 {
		t.Fatalf("artists = %d, want 2", len(view.Artists))
	}
	if view.Artists[0].Name != "Alpha, Filing" || view.Artists[1].Name != "Zeta, Filing" {
		t.Errorf("filing-name order = [%q, %q], want [Alpha, Filing] then [Zeta, Filing]",
			view.Artists[0].Name, view.Artists[1].Name)
	}
}
