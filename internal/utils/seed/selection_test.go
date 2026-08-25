package seed

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

// validSelectionContentHash is a well-formed 64-character SHA-256 hex digest
// used by validation fixtures.
const validSelectionContentHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// validSelectionFixture builds a selection with correct deterministic
// provenance for the given canonical source path, so a test can mutate one
// field to isolate a specific rejection.
func validSelectionFixture(sourcePath string, artistID string) sourceSelection {
	sourceHash, _ := selectionSourceHashFromPath(sourcePath)
	id, _ := selectionIDFromSourcePath(sourcePath)
	return sourceSelection{
		ID:           id,
		ArtistID:     artistID,
		Title:        "Paintings",
		Context:      "",
		DisplayTitle: "Paintings",
		Commentary:   "",
		SourcePath:   sourcePath,
		SourceHash:   sourceHash,
		ContentHash:  validSelectionContentHash,
		Published:    true,
	}
}

func openSelectionsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func createSelectionsTable(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		CREATE TABLE artwork_selections (
			id TEXT PRIMARY KEY,
			artist_id TEXT NOT NULL,
			title TEXT NOT NULL,
			context TEXT NOT NULL DEFAULT '',
			display_title TEXT NOT NULL,
			commentary_html TEXT NOT NULL DEFAULT '',
			artwork_ids TEXT NOT NULL,
			source_path TEXT NOT NULL UNIQUE,
			source_hash TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			published INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		t.Fatalf("create artwork_selections: %v", err)
	}
}

// TestSelectionSourceHashFromPathMatchesProducer pins the reimplemented
// SHA-256 derivation to the producer's algorithm with an independently
// computed golden digest.
func TestSelectionSourceHashFromPathMatchesProducer(t *testing.T) {
	got, err := selectionSourceHashFromPath("html/a/artist/paintings/index.html")
	if err != nil {
		t.Fatalf("derive source hash: %v", err)
	}
	if want := "2633f9d80c78f05c96c3996731070684cd2c3663846597353cf7345369f862f8"; got != want {
		t.Fatalf("selectionSourceHashFromPath = %q, want %q", got, want)
	}
}

// TestSelectionIDFromSourcePathMatchesProducer pins the deterministic record ID
// ("r" + first 14 hex of the SHA-256 canonical path) to the producer algorithm.
func TestSelectionIDFromSourcePathMatchesProducer(t *testing.T) {
	got, err := selectionIDFromSourcePath("html/a/artist/paintings/index.html")
	if err != nil {
		t.Fatalf("derive selection ID: %v", err)
	}
	if want := "r2633f9d80c78f0"; got != want {
		t.Fatalf("selectionIDFromSourcePath = %q, want %q", got, want)
	}
}

func TestCanonicalSelectionSourcePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "canonical", input: "html/a/artist/paintings/index.html", want: "html/a/artist/paintings/index.html"},
		{name: "physical in prefix", input: "in/html/a/artist/paintings/index.html", want: "html/a/artist/paintings/index.html"},
		{name: "backslashes", input: `html\a\artist\paintings\index.html`, want: "html/a/artist/paintings/index.html"},
		{name: "leading slash", input: "/html/a/artist/paintings/index.html", want: "html/a/artist/paintings/index.html"},
		{name: "dot segments", input: "html/a/artist/./paintings/index.html", want: "html/a/artist/paintings/index.html"},
		{name: "traversal", input: "html/a/../index.html", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "non-html", input: "bio/a/artist/index.html", wantErr: true},
		{name: "non-index basename", input: "html/a/artist/paintings.html", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := canonicalSelectionSourcePath(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("canonicalSelectionSourcePath(%q) = %q, want error", test.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("canonicalSelectionSourcePath(%q): %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("canonicalSelectionSourcePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestLoadSelectionsPreservesCommentaryBytesAndOrder(t *testing.T) {
	db := openSelectionsTestDB(t)
	createSelectionsTable(t, db)

	const commentary = `<p>Ledé with <em>emphasis</em> and a “quote”.</p>`
	if _, err := db.Exec(`
		INSERT INTO artwork_selections (
			id, artist_id, title, context, display_title, commentary_html,
			artwork_ids, source_path, source_hash, content_hash, published
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"rselect00000001", "rartist00000001", "Paintings", "Dürer", "Dürer: Paintings",
		commentary, `["rwork000000003","rwork000000001","rwork000000002"]`,
		"html/a/artist/paintings/index.html", "source-hash", "content-hash", 1,
	); err != nil {
		t.Fatalf("insert selection: %v", err)
	}

	selections, err := loadSelections(db)
	if err != nil {
		t.Fatalf("load selections: %v", err)
	}
	if len(selections) != 1 {
		t.Fatalf("selection count = %d, want 1", len(selections))
	}

	selection := selections[0]
	if selection.ID != "rselect00000001" {
		t.Errorf("ID = %q", selection.ID)
	}
	if selection.ArtistID != "rartist00000001" {
		t.Errorf("artist = %q", selection.ArtistID)
	}
	if selection.Title != "Paintings" {
		t.Errorf("title = %q", selection.Title)
	}
	if selection.Context != "Dürer" {
		t.Errorf("context = %q", selection.Context)
	}
	if selection.DisplayTitle != "Dürer: Paintings" {
		t.Errorf("display title = %q", selection.DisplayTitle)
	}
	if selection.Commentary != commentary {
		t.Errorf("commentary = %q, want byte-identical %q", selection.Commentary, commentary)
	}
	if !selection.Published {
		t.Error("published = false, want true")
	}
	if want := []string{"rwork000000003", "rwork000000001", "rwork000000002"}; !reflect.DeepEqual(selection.ArtworkIDs, want) {
		t.Errorf("artwork IDs = %v, want ordered %v", selection.ArtworkIDs, want)
	}
}

func TestLoadSelectionsMissingTableIsEmpty(t *testing.T) {
	db := openSelectionsTestDB(t)

	selections, err := loadSelections(db)
	if err != nil {
		t.Fatalf("load selections: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("selection count = %d, want 0", len(selections))
	}
}

func TestLoadSelectionsIsDeterministic(t *testing.T) {
	db := openSelectionsTestDB(t)
	createSelectionsTable(t, db)

	for _, id := range []string{"rselect00000002", "rselect00000001"} {
		sourcePath := "html/a/artist/" + id + "/index.html"
		if _, err := db.Exec(`
			INSERT INTO artwork_selections (
				id, artist_id, title, context, display_title, commentary_html,
				artwork_ids, source_path, source_hash, content_hash, published
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, "rartist00000001", "Paintings", "", "Paintings", "",
			`["rwork000000001"]`, sourcePath, "source-hash", "content-hash", 1,
		); err != nil {
			t.Fatalf("insert selection %q: %v", id, err)
		}
	}

	first, err := loadSelections(db)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	second, err := loadSelections(db)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("loadSelections is not deterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if len(first) != 2 || first[0].SourcePath > first[1].SourcePath {
		t.Fatalf("selections should be source-path ordered, got %+v", first)
	}
}

func TestParseArtworkIDsRejectsInvalidPayload(t *testing.T) {
	for _, value := range []string{"", "[]", `["rwork000000001"`, "not-json"} {
		if _, err := parseArtworkIDs(value); err == nil {
			t.Errorf("parseArtworkIDs(%q) = nil error, want rejection", value)
		}
	}
}

func TestParseArtworkIDsRejectsDuplicateMembership(t *testing.T) {
	if _, err := parseArtworkIDs(`["rwork0000000001","rwork0000000001"]`); err == nil {
		t.Fatal("parseArtworkIDs accepted duplicate artwork IDs, want rejection")
	}
}

func TestLoadSelectionsRejectsDuplicateMembership(t *testing.T) {
	db := openSelectionsTestDB(t)
	createSelectionsTable(t, db)

	if _, err := db.Exec(`
		INSERT INTO artwork_selections (
			id, artist_id, title, context, display_title, commentary_html,
			artwork_ids, source_path, source_hash, content_hash, published
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"rselect00000001", "rartist00000001", "Paintings", "", "Paintings", "",
		`["rwork0000000001","rwork0000000001"]`,
		"html/a/artist/paintings/index.html", "source-hash", "content-hash", 1,
	); err != nil {
		t.Fatalf("insert selection: %v", err)
	}

	if _, err := loadSelections(db); err == nil {
		t.Fatal("loadSelections accepted duplicate artwork membership, want rejection")
	}
}

func TestValidateSourceSelectionsRejectsUnpublished(t *testing.T) {
	data := sourceData{
		artists:    []sourceArtist{{ID: "rartist00000001"}},
		selections: []sourceSelection{{ID: "rselect00000001", ArtistID: "rartist00000001", Published: false}},
	}
	artistIDs := makeIDSet(data.artists, func(item sourceArtist) string { return item.ID })

	if err := validateSourceSelections(data, artistIDs); err == nil {
		t.Fatal("expected unpublished selection to be rejected")
	}
}

func TestValidateSourceSelectionsAcceptsValidProvenance(t *testing.T) {
	selection := validSelectionFixture("html/a/artist/paintings/index.html", "rartist00000001")
	selection.ArtworkIDs = []string{"rwork0000000001"}
	data := sourceData{
		artists:  []sourceArtist{{ID: "rartist00000001"}},
		artworks: []sourceArtwork{{ID: "rwork0000000001", AuthorID: "rartist00000001"}},
		selections: []sourceSelection{
			selection,
		},
	}
	artistIDs := makeIDSet(data.artists, func(item sourceArtist) string { return item.ID })

	if err := validateSourceSelections(data, artistIDs); err != nil {
		t.Fatalf("expected valid provenance to pass, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsNonCanonicalPath(t *testing.T) {
	artistIDs := map[string]struct{}{"rartist00000001": {}}

	backslash := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	backslash.SourcePath = `html\a\artist\index.html`
	if err := validateSourceSelections(sourceData{selections: []sourceSelection{backslash}}, artistIDs); err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("expected non-canonical path error, got %v", err)
	}

	traversal := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	traversal.SourcePath = "html/a/../index.html"
	if err := validateSourceSelections(sourceData{selections: []sourceSelection{traversal}}, artistIDs); err == nil || !strings.Contains(err.Error(), "source root") {
		t.Fatalf("expected traversal path error, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsMismatchedID(t *testing.T) {
	selection := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	selection.ID = "r00000000000000"
	artistIDs := map[string]struct{}{"rartist00000001": {}}

	err := validateSourceSelections(sourceData{selections: []sourceSelection{selection}}, artistIDs)
	if err == nil || !strings.Contains(err.Error(), "does not match deterministic source ID") {
		t.Fatalf("expected mismatched-ID error, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsMismatchedSourceHash(t *testing.T) {
	selection := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	selection.SourceHash = strings.Repeat("0", 64)
	artistIDs := map[string]struct{}{"rartist00000001": {}}

	err := validateSourceSelections(sourceData{selections: []sourceSelection{selection}}, artistIDs)
	if err == nil || !strings.Contains(err.Error(), "does not match deterministic hash") {
		t.Fatalf("expected mismatched-source-hash error, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsMalformedContentHash(t *testing.T) {
	selection := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	selection.ContentHash = "not-a-sha256"
	artistIDs := map[string]struct{}{"rartist00000001": {}}

	err := validateSourceSelections(sourceData{selections: []sourceSelection{selection}}, artistIDs)
	if err == nil || !strings.Contains(err.Error(), "SHA-256 digest") {
		t.Fatalf("expected malformed content-hash error, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsDanglingArtist(t *testing.T) {
	selection := validSelectionFixture("html/a/artist/index.html", "rmissing0000001")
	artistIDs := map[string]struct{}{"rartist00000001": {}}

	err := validateSourceSelections(sourceData{selections: []sourceSelection{selection}}, artistIDs)
	if err == nil || !strings.Contains(err.Error(), "unknown artist") {
		t.Fatalf("expected unknown-artist error, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsDanglingAndCrossArtistArtwork(t *testing.T) {
	artistIDs := map[string]struct{}{"rartist00000001": {}, "rartist00000002": {}}
	artworks := []sourceArtwork{{ID: "rwork0000000001", AuthorID: "rartist00000002"}}

	missing := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	missing.ArtworkIDs = []string{"rwork0000000009"}
	if err := validateSourceSelections(sourceData{artworks: artworks, selections: []sourceSelection{missing}}, artistIDs); err == nil || !strings.Contains(err.Error(), "unknown artwork") {
		t.Fatalf("expected dangling-artwork error, got %v", err)
	}

	crossArtist := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	crossArtist.ArtworkIDs = []string{"rwork0000000001"}
	if err := validateSourceSelections(sourceData{artworks: artworks, selections: []sourceSelection{crossArtist}}, artistIDs); err == nil || !strings.Contains(err.Error(), "belongs to artist") {
		t.Fatalf("expected cross-artist error, got %v", err)
	}
}

func TestValidateSourceSelectionsRejectsDuplicateSelection(t *testing.T) {
	selection := validSelectionFixture("html/a/artist/index.html", "rartist00000001")
	artistIDs := map[string]struct{}{"rartist00000001": {}}

	data := sourceData{
		selections: []sourceSelection{selection, selection},
	}
	if err := validateSourceSelections(data, artistIDs); err == nil || !strings.Contains(err.Error(), "duplicate selection ID") {
		t.Fatalf("expected duplicate selection ID error, got %v", err)
	}
}

func TestImportSyntheticSelectionsPreservesOrderAndBytes(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "test-encryption-key",
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	createSelectionAppCollections(t, app)

	items := []sourceSelection{{
		ID:           "rselect00000001",
		ArtistID:     "rartist00000001",
		Title:        "Paintings",
		Context:      "Dürer",
		DisplayTitle: "Dürer: Paintings",
		Commentary:   `<p>Ledé with <em>emphasis</em>.</p>`,
		ArtworkIDs:   []string{"rwork0000000003", "rwork0000000001", "rwork0000000002"},
		SourcePath:   "html/a/artist/paintings/index.html",
		SourceHash:   "source-hash",
		ContentHash:  validSelectionContentHash,
		Published:    true,
	}}

	if err := importSyntheticSelections(app, items); err != nil {
		t.Fatalf("import selections: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionSelections, "rselect00000001")
	if err != nil {
		t.Fatalf("find imported selection: %v", err)
	}
	if got := record.GetString("commentary"); got != items[0].Commentary {
		t.Errorf("commentary = %q, want byte-identical %q", got, items[0].Commentary)
	}
	if got := record.GetStringSlice("artworks"); !reflect.DeepEqual(got, items[0].ArtworkIDs) {
		t.Errorf("artworks order = %v, want %v", got, items[0].ArtworkIDs)
	}
	if got := record.GetString("artist"); got == "" {
		t.Error("artist relation not set")
	}
	if got := record.GetString("source_path"); got != items[0].SourcePath {
		t.Errorf("source path = %q, want %q", got, items[0].SourcePath)
	}
	if got := record.GetString("source_hash"); got != items[0].SourceHash {
		t.Errorf("source hash = %q, want %q", got, items[0].SourceHash)
	}
	if got := record.GetString("content_hash"); got != items[0].ContentHash {
		t.Errorf("content hash = %q, want byte-identical %q", got, items[0].ContentHash)
	}
	if !record.GetBool("published") {
		t.Error("published = false, want true")
	}
}

func createSelectionAppCollections(t *testing.T, app core.App) {
	t.Helper()

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.RelationField{Name: "author", CollectionId: "artists", MinSelect: 1, MaxSelect: 10},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	selections := core.NewBaseCollection("Art_selections")
	selections.Id = "art_selections"
	selections.MarkAsNew()
	selections.Fields.Add(
		&core.RelationField{Name: "artist", CollectionId: "artists", MinSelect: 1, MaxSelect: 1, Required: true},
		&core.TextField{Name: "title", Required: true},
		&core.TextField{Name: "context"},
		&core.TextField{Name: "display_title", Required: true},
		&core.EditorField{Name: "commentary"},
		&core.RelationField{Name: "artworks", CollectionId: "artworks", MinSelect: 1, MaxSelect: 1000, Required: true},
		&core.TextField{Name: "source_path", Required: true, Hidden: true},
		&core.TextField{Name: "source_hash", Required: true, Hidden: true},
		&core.TextField{Name: "content_hash", Required: true, Hidden: true},
		&core.BoolField{Name: "published", Required: true},
	)
	if err := app.Save(selections); err != nil {
		t.Fatalf("save selections: %v", err)
	}

	saveRecord := func(collection, id string, fields map[string]any) {
		t.Helper()
		coll, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatalf("find %s: %v", collection, err)
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

	saveRecord("artists", "rartist00000001", map[string]any{"name": "Dürer", "slug": "durer", "published": true})
	for _, id := range []string{"rwork0000000001", "rwork0000000002", "rwork0000000003"} {
		saveRecord("artworks", id, map[string]any{
			"title": "Work " + id, "author": []string{"rartist00000001"}, "published": true,
		})
	}
}
