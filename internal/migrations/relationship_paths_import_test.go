package migrations

import (
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/blackfyre/wga/resources/synthetic"
)

// TestRelationshipPathsImportContract imports an external producer export that
// carries the relationship-path contract: primary_author_id, raw location text,
// canonical techniques and assignments, reviewed location aliases, and a
// private-collection classification. It proves the importer preserves the raw
// location, keeps the producer's private classification non-public, retains the
// reviewed aliases, and populates the replacement artwork relations.
func TestRelationshipPathsImportContract(t *testing.T) {
	data, err := synthetic.Files.ReadFile("wga-test.sqlite")
	if err != nil {
		t.Fatalf("read synthetic source: %v", err)
	}
	sourcePath := filepath.Join(t.TempDir(), "wga-src.sqlite")
	if err := os.WriteFile(sourcePath, data, 0o600); err != nil {
		t.Fatalf("write external source: %v", err)
	}
	materializeEmbeddedStorage(t, filepath.Join(filepath.Dir(sourcePath), "storage"))

	const (
		privateArtworkID  = "2225c982be1af02"
		privateLocationID = "privatecollect0"
		privateRaw        = "Private collection, Test City"
		museumLocationID  = "da1cec57c285f0d"
		museumAliasKey    = "synthetic museum of fine arts test city"
		privateAliasKey   = "private collection"
		secondTechniqueID = "tempera0000000a"
		thirdTechniqueID  = "gilding0000000b"
	)
	source, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		t.Fatalf("open external source: %v", err)
	}
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO locations (id, name, city, country, museum, is_public) VALUES (?, 'Private collection', NULL, NULL, 0, 0)", []any{privateLocationID}},
		{"UPDATE artworks SET location = ?, current_location_id = ? WHERE id = ?", []any{privateRaw, privateLocationID, privateArtworkID}},
		{"INSERT INTO location_aliases (comparison_key, location_id) VALUES (?, ?), (?, ?)", []any{museumAliasKey, museumLocationID, privateAliasKey, privateLocationID}},
		{"INSERT INTO techniques (id, name, comparison_key) VALUES (?, 'Tempera on panel', 'tempera on panel')", []any{secondTechniqueID}},
		{"INSERT INTO techniques (id, name, comparison_key) VALUES (?, 'Gilding', 'gilding')", []any{thirdTechniqueID}},
		{"DELETE FROM artwork_techniques WHERE artwork_id = ?", []any{privateArtworkID}},
		{"INSERT INTO artwork_techniques (artwork_id, technique_id) VALUES (?, ?), (?, ?)", []any{privateArtworkID, secondTechniqueID, privateArtworkID, thirdTechniqueID}},
	} {
		if _, err := source.Exec(statement.sql, statement.args...); err != nil {
			t.Fatalf("add relationship-path source data %q: %v", statement.sql, err)
		}
	}
	if err := source.Close(); err != nil {
		t.Fatalf("close external source: %v", err)
	}

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ClearBootstrap(); err != nil {
			t.Error(err)
		}
	})
	steps := []struct {
		name string
		run  func() error
	}{
		{"baseline schema", func() error { return createCurrentSchema(app) }},
		{"artist selections", func() error { return addArtistSelections(app) }},
		{"artist identity", func() error { return addArtistIdentityFields(app) }},
		{"collection data", func() error { return addCollectionData(app) }},
		{"artwork source fields", func() error { return addArtworkSourceFields(app) }},
		{"artwork byte size", func() error { return addArtworkFileByteSize(app) }},
		{"relationship paths", func() error { return addRelationshipPaths(app) }},
		{"import", func() error { return seed.Import(app, sourcePath) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}

	artworks, err := app.FindAllRecords("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	if len(artworks) == 0 {
		t.Fatal("expected imported artworks")
	}
	for _, artwork := range artworks {
		legacy := artwork.GetStringSlice("author")
		if len(legacy) != 1 || artwork.GetString("primary_author") != legacy[0] {
			t.Fatalf("artwork %s primary_author = %q, legacy author = %v; want the producer primary author", artwork.Id, artwork.GetString("primary_author"), legacy)
		}
		if artwork.GetString("source_location") == "" {
			t.Fatalf("artwork %s lost its raw source location", artwork.Id)
		}
	}

	private, err := app.FindRecordById("artworks", privateArtworkID)
	if err != nil {
		t.Fatalf("find private artwork: %v", err)
	}
	if got := private.GetString("source_location"); got != privateRaw {
		t.Fatalf("private source_location = %q, want raw %q", got, privateRaw)
	}
	location, err := app.FindRecordById("locations", private.GetString("current_location_id"))
	if err != nil {
		t.Fatalf("find private location: %v", err)
	}
	if location.GetBool("is_public") || location.GetBool("museum") {
		t.Fatalf("private collection location is_public=%v museum=%v, want both false", location.GetBool("is_public"), location.GetBool("museum"))
	}
	// Both producer assignments survive the save: techniques is multi-valued.
	if techniques := private.GetStringSlice("techniques"); !slices.Equal(techniques, []string{thirdTechniqueID, secondTechniqueID}) {
		t.Fatalf("private artwork techniques = %v, want both %s and %s", techniques, thirdTechniqueID, secondTechniqueID)
	}

	aliases, err := app.FindAllRecords("location_aliases")
	if err != nil {
		t.Fatalf("find location aliases: %v", err)
	}
	resolved := map[string]string{}
	for _, alias := range aliases {
		resolved[alias.GetString("comparison_key")] = alias.GetString("location")
	}
	if resolved[museumAliasKey] != museumLocationID || resolved[privateAliasKey] != privateLocationID || len(resolved) != 2 {
		t.Fatalf("location aliases = %v, want the two reviewed source aliases", resolved)
	}
}
