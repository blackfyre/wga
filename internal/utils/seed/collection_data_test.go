package seed

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

func openCollectionDataTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestLoadLocations(t *testing.T) {
	db := openCollectionDataTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE locations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			city TEXT,
			country TEXT,
			museum INTEGER NOT NULL DEFAULT 0,
			is_public INTEGER NOT NULL DEFAULT 1
		);
		INSERT INTO locations VALUES ('loc1', 'Musée du Louvre', 'Paris', 'FR', 1, 1);
		INSERT INTO locations VALUES ('loc2', 'Private Collection', NULL, NULL, 0, 0);
	`); err != nil {
		t.Fatalf("create locations: %v", err)
	}

	locations, err := loadLocations(db)
	if err != nil {
		t.Fatalf("load locations: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("location count = %d, want 2", len(locations))
	}
	first := locations[0]
	if first.ID != "loc1" || first.Name != "Musée du Louvre" || first.City != "Paris" || first.Country != "FR" || !first.Museum || !first.IsPublic {
		t.Fatalf("first location = %+v", first)
	}
	second := locations[1]
	if second.City != "" || second.Country != "" || second.Museum || second.IsPublic {
		t.Fatalf("second location should have empty city/country and false flags, got %+v", second)
	}
}

func TestLoadLocationsMissingTableIsEmpty(t *testing.T) {
	db := openCollectionDataTestDB(t)

	locations, err := loadLocations(db)
	if err != nil {
		t.Fatalf("load locations: %v", err)
	}
	if len(locations) != 0 {
		t.Fatalf("location count = %d, want 0", len(locations))
	}
}

const collectionDataArtworksSchema = `
	CREATE TABLE artworks (
		id TEXT PRIMARY KEY,
		author_id TEXT,
		title TEXT,
		date_text TEXT,
		technique TEXT,
		dimensions TEXT,
		location TEXT,
		output_image_path TEXT,
		image_width INTEGER,
		image_height INTEGER,
		school_id TEXT,
		form_id TEXT,
		type_id TEXT,
		source_row INTEGER,
		date_start INTEGER,
		date_end INTEGER,
		is_circa INTEGER,
		date_qualifier TEXT,
		timeframe_text TEXT,
		current_location_id TEXT,
		art_period_id TEXT,
		tone_keywords TEXT
	)
`

func TestLoadArtworksCarriesCollectionData(t *testing.T) {
	db := openCollectionDataTestDB(t)
	if _, err := db.Exec(collectionDataArtworksSchema); err != nil {
		t.Fatalf("create artworks: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO artworks (
			id, author_id, title, date_text, technique, dimensions, location,
			output_image_path, image_width, image_height, school_id, form_id, type_id,
			source_row, date_start, date_end, is_circa, date_qualifier, timeframe_text,
			current_location_id, art_period_id, tone_keywords
		) VALUES (
			'artwork', 'artist', 'Artwork', 'before 1574', 'Oil', NULL, 'Paris',
			'artwork.jpg', 1200, 800, NULL, 'form', NULL,
			42, 1574, NULL, 0, 'before', '1601-1650',
			'loc1', 'period1', '["warm","cool"]'
		);
	`); err != nil {
		t.Fatalf("insert artwork: %v", err)
	}

	artworks, err := loadArtworks(db)
	if err != nil {
		t.Fatalf("load artworks: %v", err)
	}
	if len(artworks) != 1 {
		t.Fatalf("artwork count = %d, want 1", len(artworks))
	}
	item := artworks[0]
	if item.SourceRow != 42 {
		t.Errorf("source_row = %d, want 42", item.SourceRow)
	}
	if item.DateStart != 1574 {
		t.Errorf("date_start = %d, want 1574", item.DateStart)
	}
	if item.DateEnd != 0 {
		t.Errorf("date_end = %d, want 0 for null", item.DateEnd)
	}
	if item.IsCirca {
		t.Error("is_circa = true, want false")
	}
	if item.DateQualifier != "before" {
		t.Errorf("date_qualifier = %q, want before", item.DateQualifier)
	}
	if item.TimeframeText != "1601-1650" {
		t.Errorf("timeframe_text = %q, want 1601-1650", item.TimeframeText)
	}
	if item.CurrentLocationID != "loc1" {
		t.Errorf("current_location_id = %q, want loc1", item.CurrentLocationID)
	}
	if item.ArtPeriodID != "period1" {
		t.Errorf("art_period_id = %q, want period1", item.ArtPeriodID)
	}
}

func TestLoadArtworksAbsentOptionalColumnsAreZero(t *testing.T) {
	db := openCollectionDataTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE artworks (
			id TEXT PRIMARY KEY,
			author_id TEXT,
			title TEXT,
			date_text TEXT,
			technique TEXT,
			dimensions TEXT,
			location TEXT,
			output_image_path TEXT,
			image_width INTEGER,
			image_height INTEGER,
			school_id TEXT,
			form_id TEXT,
			type_id TEXT
		);
		INSERT INTO artworks VALUES ('artwork', 'artist', 'Artwork', '1900', 'Oil', NULL, 'Gallery', 'artwork.jpg', 1200, 800, NULL, 'form', NULL);
	`); err != nil {
		t.Fatalf("create artwork: %v", err)
	}

	artworks, err := loadArtworks(db)
	if err != nil {
		t.Fatalf("load artworks: %v", err)
	}
	if len(artworks) != 1 {
		t.Fatalf("artwork count = %d, want 1", len(artworks))
	}
	item := artworks[0]
	if item.SourceRow != 0 || item.DateStart != 0 || item.DateEnd != 0 || item.IsCirca ||
		item.DateQualifier != "" || item.TimeframeText != "" ||
		item.CurrentLocationID != "" || item.ArtPeriodID != "" {
		t.Fatalf("optional collection-data fields should be zero-valued, got %+v", item)
	}
	if item.ImageWidth != 1200 || item.ImageHeight != 800 {
		t.Fatalf("base fields altered: %+v", item)
	}
}

func TestValidateSourceRelationsRejectsUnknownLocationAndPeriod(t *testing.T) {
	base := sourceArtwork{ID: "rwork000000001", AuthorID: "rartist00000001", FormID: "rform000000001"}
	artists := []sourceArtist{{ID: "rartist00000001"}}
	forms := []sourceTaxonomy{{ID: "rform000000001"}}
	locations := []sourceLocation{{ID: "rloc0000000001"}}
	artPeriods := []sourceArtPeriod{{ID: "rperiod0000001"}}

	data := sourceData{
		artists:    artists,
		forms:      forms,
		locations:  locations,
		artPeriods: artPeriods,
		artworks:   []sourceArtwork{{ID: base.ID, AuthorID: base.AuthorID, FormID: base.FormID, CurrentLocationID: "rmissinglocation"}},
	}
	if err := validateSourceRelations(data); err == nil {
		t.Fatal("expected unknown-location error")
	}

	data.artworks = []sourceArtwork{{ID: base.ID, AuthorID: base.AuthorID, FormID: base.FormID, ArtPeriodID: "rmissingperiod"}}
	if err := validateSourceRelations(data); err == nil {
		t.Fatal("expected unknown-art-period error")
	}

	data.artworks = []sourceArtwork{{ID: base.ID, AuthorID: base.AuthorID, FormID: base.FormID, CurrentLocationID: "rloc0000000001", ArtPeriodID: "rperiod0000001"}}
	if err := validateSourceRelations(data); err != nil {
		t.Fatalf("valid location/period references should pass: %v", err)
	}
}

func createCollectionDataAppCollections(t *testing.T, app core.App) {
	t.Helper()

	saveCollection := func(name, id string, fields ...core.Field) {
		t.Helper()
		collection := core.NewBaseCollection(name)
		collection.Id = id
		collection.MarkAsNew()
		collection.Fields.Add(fields...)
		if err := app.Save(collection); err != nil {
			t.Fatalf("save %s collection: %v", id, err)
		}
	}

	saveCollection("Artists", "artists",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.BoolField{Name: "published"},
	)
	saveCollection("Art_forms", "art_forms",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	saveCollection("Art_types", "art_types",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	saveCollection("Schools", "schools",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	saveCollection("Art_periods", "art_periods",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.NumberField{Name: "start"},
		&core.NumberField{Name: "end"},
		&core.TextField{Name: "description"},
	)
	saveCollection("Locations", constants.CollectionLocations,
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "city"},
		&core.TextField{Name: "country"},
		&core.BoolField{Name: "museum"},
		&core.BoolField{Name: "is_public"},
	)
	saveCollection("Artworks", constants.CollectionArtworks,
		&core.TextField{Name: "title"},
		&core.RelationField{Name: "author", CollectionId: "artists", MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Name: "form", CollectionId: "art_forms", MinSelect: 1, MaxSelect: 20},
		&core.RelationField{Name: "type", CollectionId: "art_types", MinSelect: 0, MaxSelect: 20},
		&core.RelationField{Name: "school", CollectionId: "schools", MinSelect: 0, MaxSelect: 10},
		&core.TextField{Name: "technique"},
		&core.EditorField{Name: "comment"},
		&core.BoolField{Name: "published"},
		&core.FileField{Name: "image", MaxSelect: 1},
		&core.NumberField{Name: "image_width"},
		&core.NumberField{Name: "image_height"},
		&core.NumberField{Name: "source_row"},
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "date_end"},
		&core.BoolField{Name: "is_circa"},
		&core.TextField{Name: "date_qualifier"},
		&core.TextField{Name: "timeframe_text"},
		&core.RelationField{Name: "current_location_id", CollectionId: constants.CollectionLocations, MinSelect: 0, MaxSelect: 1},
		&core.RelationField{Name: "art_period_id", CollectionId: "art_periods", MinSelect: 0, MaxSelect: 1},
		&core.TextField{Name: "source_url"},
		&core.TextField{Name: "source_path"},
		&core.TextField{Name: "source_comment", Max: 10000},
		&core.JSONField{Name: "colour_palette"},
		&core.JSONField{Name: "colour_signature"},
		&core.TextField{Name: "colour_profile_version"},
		&core.TextField{Name: "colour_image_hash"},
		&core.NumberField{Name: "image_size_bytes"},
	)
}

func TestImportSyntheticLocations(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	items := []sourceLocation{
		{ID: "rlocation000001", Name: "Musée du Louvre", City: "Paris", Country: "FR", Museum: true, IsPublic: true},
		{ID: "rlocation000002", Name: "Private Collection", Museum: false, IsPublic: false},
	}
	if err := importSyntheticLocations(app, items); err != nil {
		t.Fatalf("import locations: %v", err)
	}

	museum, err := app.FindRecordById(constants.CollectionLocations, "rlocation000001")
	if err != nil {
		t.Fatalf("find imported museum location: %v", err)
	}
	if museum.GetString("name") != "Musée du Louvre" || museum.GetString("city") != "Paris" || museum.GetString("country") != "FR" {
		t.Fatalf("museum location fields = %+v", museum)
	}
	if !museum.GetBool("museum") || !museum.GetBool("is_public") {
		t.Fatalf("museum location flags should be true")
	}

	private, err := app.FindRecordById(constants.CollectionLocations, "rlocation000002")
	if err != nil {
		t.Fatalf("find imported private location: %v", err)
	}
	if private.GetString("city") != "" || private.GetString("country") != "" {
		t.Fatalf("private location should leave city/country unset")
	}
	if private.GetBool("museum") || private.GetBool("is_public") {
		t.Fatalf("private location flags should be false")
	}
}

func TestImportSyntheticArtworksCarriesCollectionData(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	data := sourceData{
		artworkFiles: map[string]sourceFile{
			"rwork0000000001": {name: "artwork.jpg", preseededAssets: true},
		},
		artworks: []sourceArtwork{{
			ID:                "rwork0000000001",
			AuthorID:          "rartist00000001",
			Title:             "Artwork",
			DateText:          "before 1574",
			FormID:            "rform0000000001",
			ImagePath:         "artwork.jpg",
			SourceRow:         42,
			DateStart:         1574,
			IsCirca:           false,
			DateQualifier:     "before",
			TimeframeText:     "1601-1650",
			CurrentLocationID: "rlocation000001",
			ArtPeriodID:       "rperiod00000001",
		}},
	}

	if err := importSyntheticArtworks(app, data, true); err != nil {
		t.Fatalf("import artworks: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionArtworks, "rwork0000000001")
	if err != nil {
		t.Fatalf("find imported artwork: %v", err)
	}
	if got := record.GetInt("source_row"); got != 42 {
		t.Errorf("source_row = %d, want 42", got)
	}
	if got := record.GetInt("date_start"); got != 1574 {
		t.Errorf("date_start = %d, want 1574", got)
	}
	if got := record.GetInt("date_end"); got != 0 {
		t.Errorf("date_end = %d, want 0", got)
	}
	if record.GetBool("is_circa") {
		t.Error("is_circa = true, want false")
	}
	if got := record.GetString("date_qualifier"); got != "before" {
		t.Errorf("date_qualifier = %q, want before", got)
	}
	if got := record.GetString("timeframe_text"); got != "1601-1650" {
		t.Errorf("timeframe_text = %q, want 1601-1650", got)
	}
	if got := record.GetString("current_location_id"); got != "rlocation000001" {
		t.Errorf("current_location_id = %q, want rlocation000001", got)
	}
	if got := record.GetString("art_period_id"); got != "rperiod00000001" {
		t.Errorf("art_period_id = %q, want rperiod00000001", got)
	}
}

func TestImportSyntheticArtworksCarriesSourceAndColourFields(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	data := sourceData{
		artworkFiles: map[string]sourceFile{
			"rwork0000000001": {name: "artwork.jpg", preseededAssets: true},
		},
		artworks: []sourceArtwork{{
			ID:            "rwork0000000001",
			AuthorID:      "rartist00000001",
			Title:         "Artwork",
			DateText:      "before 1574",
			FormID:        "rform0000000001",
			ImagePath:     "artwork.jpg",
			SourceURL:     "html/a/artist/work.html",
			SourcePath:    "in/art/a/artist/work.jpg",
			SourceComment: "Raw source comment.",
			Comment:       "Enriched narrative commentary.",
			ColourPalette: []sourceColourPaletteEntry{
				{Hex: "#1a2b3c", Weight: 5000},
				{Hex: "#4d5e6f", Weight: 3000},
			},
			ColourSignature:      &sourceColourSignature{Space: "oklab-hcl-12x3x4", Bins: []int{100, 200, 300}},
			ColourProfileVersion: "oklab-hcl-v1",
			ColourImageHash:      "image-hash-abc",
		}},
	}

	if err := importSyntheticArtworks(app, data, true); err != nil {
		t.Fatalf("import artworks: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionArtworks, "rwork0000000001")
	if err != nil {
		t.Fatalf("find imported artwork: %v", err)
	}
	if got := record.GetString("source_url"); got != "html/a/artist/work.html" {
		t.Errorf("source_url = %q, want html/a/artist/work.html", got)
	}
	if got := record.GetString("source_path"); got != "in/art/a/artist/work.jpg" {
		t.Errorf("source_path = %q, want in/art/a/artist/work.jpg", got)
	}
	if got := record.GetString("source_comment"); got != "Raw source comment." {
		t.Errorf("source_comment = %q, want raw source comment", got)
	}
	if got := record.GetString("comment"); got != "Enriched narrative commentary." {
		t.Errorf("comment = %q, want enriched commentary (not fabricated)", got)
	}
	if got := record.GetString("colour_profile_version"); got != "oklab-hcl-v1" {
		t.Errorf("colour_profile_version = %q, want oklab-hcl-v1", got)
	}
	if got := record.GetString("colour_image_hash"); got != "image-hash-abc" {
		t.Errorf("colour_image_hash = %q, want image-hash-abc", got)
	}

	paletteJSON, err := json.Marshal(record.Get("colour_palette"))
	if err != nil {
		t.Fatalf("marshal colour_palette: %v", err)
	}
	paletteString := string(paletteJSON)
	if !strings.Contains(paletteString, "#1a2b3c") || !strings.Contains(paletteString, "5000") || !strings.Contains(paletteString, "#4d5e6f") {
		t.Errorf("colour_palette = %s, want hex/weight entries", paletteString)
	}

	signatureJSON, err := json.Marshal(record.Get("colour_signature"))
	if err != nil {
		t.Fatalf("marshal colour_signature: %v", err)
	}
	signatureString := string(signatureJSON)
	if !strings.Contains(signatureString, "oklab-hcl-12x3x4") || !strings.Contains(signatureString, "100") {
		t.Errorf("colour_signature = %s, want space/bins", signatureString)
	}
}

func TestImportSyntheticArtworksOmitsAbsentSourceAndColourFields(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	data := sourceData{
		artworkFiles: map[string]sourceFile{
			"rwork0000000001": {name: "artwork.jpg", preseededAssets: true},
		},
		artworks: []sourceArtwork{{
			ID:        "rwork0000000001",
			AuthorID:  "rartist00000001",
			Title:     "Artwork",
			DateText:  "1900",
			FormID:    "rform0000000001",
			ImagePath: "artwork.jpg",
		}},
	}

	if err := importSyntheticArtworks(app, data, true); err != nil {
		t.Fatalf("import artworks: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionArtworks, "rwork0000000001")
	if err != nil {
		t.Fatalf("find imported artwork: %v", err)
	}
	for _, field := range []string{
		"source_url", "source_path", "source_comment", "colour_profile_version", "colour_image_hash",
	} {
		if got := record.GetString(field); got != "" {
			t.Errorf("%s = %q, want empty", field, got)
		}
	}
	for _, field := range []string{"colour_palette", "colour_signature"} {
		marshalled, err := json.Marshal(record.GetRaw(field))
		if err != nil {
			t.Errorf("marshal %s: %v", field, err)
			continue
		}
		if got := string(marshalled); got != "null" && got != "" {
			t.Errorf("%s = %s, want empty (null)", field, got)
		}
	}
	// Absent enriched commentary falls back to the truthful metadata summary,
	// never invented prose.
	if got := record.GetString("comment"); !strings.Contains(got, "1900") {
		t.Errorf("comment = %q, want truthful metadata summary", got)
	}
	if strings.Contains(record.GetString("comment"), "Enriched") {
		t.Error("comment must not invent enriched prose when absent")
	}
}

func TestImportSyntheticArtworksImageLessPersistsWithoutFile(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		artworks: []sourceArtwork{{
			ID:        "rwork0000000001",
			AuthorID:  "rartist00000001",
			Title:     "Image-less artwork",
			FormID:    "rform0000000001",
			ImagePath: "",
		}},
	}

	if err := importSyntheticArtworks(app, data, true); err != nil {
		t.Fatalf("import image-less artwork: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionArtworks, "rwork0000000001")
	if err != nil {
		t.Fatalf("find imported image-less artwork: %v", err)
	}
	if got := record.GetString("image"); got != "" {
		t.Fatalf("image = %q, want empty for image-less artwork", got)
	}
}

func TestImportSyntheticArtworksRejectsDeclaredMissingMedia(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		artworks: []sourceArtwork{{
			ID:        "rwork0000000001",
			AuthorID:  "rartist00000001",
			Title:     "Artwork with declared media",
			FormID:    "rform0000000001",
			ImagePath: "artworks/rwork0000000001/image.jpg",
		}},
	}

	if err := importSyntheticArtworks(app, data, true); err == nil {
		t.Fatal("expected declared missing media error")
	}
}

func TestImportSyntheticArtworksRetainsLongSourceComment(t *testing.T) {
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

	createCollectionDataAppCollections(t, app)

	artistCollection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	artist := core.NewRecord(artistCollection)
	artist.Set("id", "rartist00000001")
	artist.Set("name", "Artist")
	artist.Set("slug", "artist")
	artist.Set("published", true)
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist: %v", err)
	}
	formCollection, err := app.FindCollectionByNameOrId("art_forms")
	if err != nil {
		t.Fatalf("find art_forms collection: %v", err)
	}
	form := core.NewRecord(formCollection)
	form.Set("id", "rform0000000001")
	form.Set("name", "Form")
	form.Set("slug", "form")
	if err := app.Save(form); err != nil {
		t.Fatalf("save form: %v", err)
	}

	// 6,000 characters exceeds PocketBase's unset 5,000-character text ceiling
	// but stays within the migration's raised source_comment ceiling.
	comment := strings.Repeat("x", 6000)
	content := []byte("image bytes")

	data := sourceData{
		artworkFiles: map[string]sourceFile{
			"rwork0000000001": {name: "image.jpg", content: content, size: int64(len(content))},
		},
		artworks: []sourceArtwork{{
			ID:            "rwork0000000001",
			AuthorID:      "rartist00000001",
			Title:         "Artwork",
			DateText:      "1900",
			FormID:        "rform0000000001",
			ImagePath:     "image.jpg",
			SourceComment: comment,
		}},
	}

	if err := importSyntheticArtworks(app, data, false); err != nil {
		t.Fatalf("import artworks: %v", err)
	}

	record, err := app.FindRecordById(constants.CollectionArtworks, "rwork0000000001")
	if err != nil {
		t.Fatalf("find imported artwork: %v", err)
	}
	if got := record.GetString("source_comment"); got != comment {
		t.Fatalf("source_comment length = %d, want %d (exact retention)", len(got), len(comment))
	}
	if got := record.GetInt("image_size_bytes"); got != int(len(content)) {
		t.Fatalf("image_size_bytes = %d, want %d", got, len(content))
	}
}
