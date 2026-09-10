package migrations

import (
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestArtistQueryIndexesMigrationLifecycle(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtistIdentityFields(app); err != nil {
		t.Fatalf("add artist identity fields: %v", err)
	}

	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(artists)
	record.Id = "indexartist0001"
	record.Set("name", "Index Artist")
	record.Set("slug", "index-artist")
	record.Set("filing_name", "Artist, Index")
	record.Set("short_name", "Index Artist")
	record.Set("known_place_of_birth", "n/a")
	record.Set("known_place_of_death", "n/a")
	record.Set("published", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("save existing artist: %v", err)
	}

	if err := addArtistQueryIndexes(app); err != nil {
		t.Fatalf("add artist query indexes: %v", err)
	}
	if err := addArtistQueryIndexes(app); err != nil {
		t.Fatalf("add artist query indexes again: %v", err)
	}

	artists, err = app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("reload artists: %v", err)
	}
	for _, expected := range artistQueryIndexes {
		if countExact(artists.Indexes, expected) != 1 {
			t.Fatalf("index definition count for %q = %d, want 1 in %#v", expected, countExact(artists.Indexes, expected), artists.Indexes)
		}
	}
	assertPhysicalArtistIndexes(t, app, true)
	assertArtistQueryPlanUsesIndex(t, app,
		"filing_name ASC, id ASC",
		artistIndexPublishedFiling,
	)
	assertArtistQueryPlanUsesIndex(t, app,
		"filing_name DESC, id ASC",
		artistIndexPublishedFilingDesc,
	)
	assertArtistQueryPlanUsesIndex(t, app,
		"(year_of_birth = 0) ASC, year_of_birth ASC, filing_name ASC, id ASC",
		artistIndexPublishedBirth,
	)
	assertIndexArtistPreserved(t, app, record.Id)

	if err := removeArtistQueryIndexes(app); err != nil {
		t.Fatalf("remove artist query indexes: %v", err)
	}
	artists, err = app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("reload rolled-back artists: %v", err)
	}
	for _, removed := range artistQueryIndexes {
		if slices.Contains(artists.Indexes, removed) {
			t.Fatalf("rollback retained index %q", removed)
		}
	}
	for _, baseline := range []string{"pbx_artist_slug", "pbx_artist_published_name"} {
		if !containsIndexName(artists.Indexes, baseline) {
			t.Fatalf("rollback removed baseline index %q from %#v", baseline, artists.Indexes)
		}
	}
	assertPhysicalArtistIndexes(t, app, false)
	assertIndexArtistPreserved(t, app, record.Id)
}

func countExact(values []string, wanted string) int {
	count := 0
	for _, value := range values {
		if value == wanted {
			count++
		}
	}
	return count
}

func containsIndexName(indexes []string, name string) bool {
	for _, index := range indexes {
		if strings.Contains(index, "`"+name+"`") {
			return true
		}
	}
	return false
}

func assertPhysicalArtistIndexes(t *testing.T, app core.App, want bool) {
	t.Helper()
	rows := []struct {
		Name string `db:"name"`
	}{}
	if err := app.DB().NewQuery("SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'Artists'").All(&rows); err != nil {
		t.Fatalf("list physical artist indexes: %v", err)
	}
	names := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		names[row.Name] = struct{}{}
	}
	for _, name := range []string{artistIndexPublishedFiling, artistIndexPublishedFilingDesc, artistIndexPublishedBirth} {
		_, found := names[name]
		if found != want {
			t.Fatalf("physical index %q present = %t, want %t; indexes = %#v", name, found, want, names)
		}
	}
}

func assertArtistQueryPlanUsesIndex(t *testing.T, app core.App, orderBy string, index string) {
	t.Helper()
	rows := []struct {
		Detail string `db:"detail"`
	}{}
	query := `EXPLAIN QUERY PLAN
		SELECT Artists.*
		FROM Artists
		WHERE published = true
			AND filing_name IS NOT NULL AND TRIM(filing_name) != ''
			AND short_name IS NOT NULL AND TRIM(short_name) != ''
		ORDER BY ` + orderBy + `
		LIMIT 60 OFFSET 0`
	if err := app.DB().NewQuery(query).All(&rows); err != nil {
		t.Fatalf("explain artist order %q: %v", orderBy, err)
	}
	details := make([]string, 0, len(rows))
	for _, row := range rows {
		details = append(details, row.Detail)
	}
	plan := strings.Join(details, "\n")
	if !strings.Contains(plan, "USING INDEX "+index) {
		t.Fatalf("artist order %q plan = %q, want index %q", orderBy, plan, index)
	}
	if strings.Contains(plan, "TEMP B-TREE") {
		t.Fatalf("artist order %q retained temporary sort: %q", orderBy, plan)
	}
}

func assertIndexArtistPreserved(t *testing.T, app core.App, id string) {
	t.Helper()
	record, err := app.FindRecordById("artists", id)
	if err != nil {
		t.Fatalf("find existing artist after migration transition: %v", err)
	}
	if record.GetString("filing_name") != "Artist, Index" || record.GetString("short_name") != "Index Artist" || !record.GetBool("published") {
		t.Fatalf("existing artist changed: %#v", record)
	}
}
