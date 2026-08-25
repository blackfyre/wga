package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestRetireArtworkToneKeywordsMigrationUpDownUp(t *testing.T) {
	app := newToneKeywordsApp(t)

	if err := retireArtworkToneKeywords(app); err != nil {
		t.Fatalf("retire tone keywords: %v", err)
	}
	assertToneKeywordsField(t, app, false)

	if err := restoreArtworkToneKeywords(app); err != nil {
		t.Fatalf("restore tone keywords: %v", err)
	}
	assertToneKeywordsField(t, app, true)

	if err := retireArtworkToneKeywords(app); err != nil {
		t.Fatalf("retire tone keywords again: %v", err)
	}
	assertToneKeywordsField(t, app, false)
}

func TestRetireArtworkToneKeywordsRemovesEmptyValues(t *testing.T) {
	cases := []struct {
		name     string
		sqlValue string
	}{
		{"sql null", "NULL"},
		{"json null", "'null'"},
		{"json empty string", "'\"\"'"},
		{"empty array", "'[]'"},
		{"empty object", "'{}'"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newToneKeywordsApp(t)
			id := insertArtwork(t, app)
			setArtworkToneKeywords(t, app, id, tc.sqlValue)

			if err := retireArtworkToneKeywords(app); err != nil {
				t.Fatalf("retire tone keywords with %s: %v", tc.name, err)
			}
			assertToneKeywordsField(t, app, false)
		})
	}
}

func TestRetireArtworkToneKeywordsRefusesStoredValues(t *testing.T) {
	app := newToneKeywordsApp(t)

	values := []string{
		`{"warm":true}`,
		`["bright"]`,
		`"cool"`,
	}
	ids := make([]string, 0, len(values))
	for _, v := range values {
		id := insertArtwork(t, app)
		setArtworkToneKeywords(t, app, id, "'"+v+"'")
		ids = append(ids, id)
	}

	err := retireArtworkToneKeywords(app)
	if err == nil {
		t.Fatal("retire tone keywords: expected refusal")
	}
	if want := fmt.Sprintf("%d artwork records contain authoritative values", len(values)); !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal %q does not report count %q", err.Error(), want)
	}
	for _, id := range ids {
		if !strings.Contains(err.Error(), id) {
			t.Fatalf("refusal %q does not list affected id %q", err.Error(), id)
		}
	}
	for _, v := range values {
		if strings.Contains(err.Error(), v) {
			t.Fatalf("refusal %q leaks source content %q", err.Error(), v)
		}
	}
	assertToneKeywordsField(t, app, true)
}

func TestRetireArtworkToneKeywordsRefusalCapsReportedIDs(t *testing.T) {
	app := newToneKeywordsApp(t)

	const total = maxReportedToneKeywordIDs + 2
	ids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		id := insertArtwork(t, app)
		setArtworkToneKeywords(t, app, id, fmt.Sprintf(`'{"i":%d}'`, i))
		ids = append(ids, id)
	}

	err := retireArtworkToneKeywords(app)
	if err == nil {
		t.Fatal("retire tone keywords: expected refusal")
	}
	if want := fmt.Sprintf("%d artwork records contain authoritative values", total); !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal %q does not report full count %q", err.Error(), want)
	}

	reported := 0
	for _, id := range ids {
		if strings.Contains(err.Error(), id) {
			reported++
		}
	}
	if reported != maxReportedToneKeywordIDs {
		t.Fatalf("refusal listed %d ids, want capped %d", reported, maxReportedToneKeywordIDs)
	}
	if !strings.Contains(err.Error(), "...") {
		t.Fatalf("refusal %q does not mark truncated id list", err.Error())
	}
	assertToneKeywordsField(t, app, true)
}

// newToneKeywordsApp creates a fresh app whose schema carries the legacy
// tone_keywords field, matching the state a deployed instance sees before the
// retirement migration runs.
func newToneKeywordsApp(t *testing.T) *core.BaseApp {
	t.Helper()
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addCollectionData(app); err != nil {
		t.Fatalf("add collection data: %v", err)
	}
	addLegacyToneKeywords(t, app)
	return app
}

// insertArtwork stores a minimal artwork record (validation skipped, mirroring
// the migration's tolerance for legacy optional data) and returns its id.
func insertArtwork(t *testing.T, app core.App) string {
	t.Helper()
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(artworks)
	record.Set("title", "tone keyword record")
	if err := app.SaveNoValidate(record); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
	return record.Id
}

// setArtworkToneKeywords assigns a raw SQL expression (for example NULL or a
// quoted JSON literal) to the tone_keywords column of the given artwork row.
func setArtworkToneKeywords(t *testing.T, app core.App, id, expr string) {
	t.Helper()
	if _, err := app.DB().NewQuery(
		"UPDATE Artworks SET tone_keywords = " + expr + " WHERE id = {:id}",
	).Bind(dbx.Params{"id": id}).Execute(); err != nil {
		t.Fatalf("set tone_keywords to %s: %v", expr, err)
	}
}

func addLegacyToneKeywords(t *testing.T, app core.App) {
	t.Helper()
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	artworks.Fields.Add(jsonField("tone_keywords", false))
	if err := app.Save(artworks); err != nil {
		t.Fatalf("add legacy tone_keywords field: %v", err)
	}
}

func assertToneKeywordsField(t *testing.T, app core.App, want bool) {
	t.Helper()
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	field := artworks.Fields.GetByName("tone_keywords")
	if (field != nil) != want {
		t.Fatalf("tone_keywords field present = %v, want %v", field != nil, want)
	}
	if field != nil && field.Type() != core.FieldTypeJSON {
		t.Fatalf("tone_keywords field type = %q, want %q", field.Type(), core.FieldTypeJSON)
	}
}
