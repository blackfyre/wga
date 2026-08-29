package seed

import (
	"database/sql"
	"reflect"
	"testing"
)

// artworkSourceSchema models the producer artworks columns that the source
// reader carries: the base archive fields plus the later-added source URL/path,
// colour-profile, source-comment, and current-location/art-period columns.
const artworkSourceSchema = `
	CREATE TABLE artworks (
		id TEXT PRIMARY KEY,
		author_id TEXT,
		title TEXT,
		date_text TEXT,
		technique TEXT,
		dimensions TEXT,
		location TEXT,
		url TEXT,
		image_path TEXT,
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
		tone_keywords TEXT,
		colour_palette TEXT,
		colour_signature TEXT,
		colour_profile_version TEXT,
		colour_image_hash TEXT,
		source_comment TEXT,
		comment TEXT
	)
`

func openArtworkSourceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestLoadArtworksCarriesColourProfileAndSourceFields(t *testing.T) {
	db := openArtworkSourceTestDB(t)
	if _, err := db.Exec(artworkSourceSchema); err != nil {
		t.Fatalf("create artworks: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO artworks (
			id, author_id, title, date_text, technique, dimensions, location,
			url, image_path, output_image_path, image_width, image_height,
			school_id, form_id, type_id,
			source_row, date_start, date_end, is_circa, date_qualifier, timeframe_text,
			current_location_id, art_period_id, tone_keywords,
			colour_palette, colour_signature, colour_profile_version, colour_image_hash,
			source_comment, comment
		) VALUES (
			'artwork', 'artist', 'Artwork', 'before 1574', 'Oil', '101 x 201 cm', 'Paris',
			'html/a/artist/work.html', 'in/art/a/artist/work.jpg', 'artwork.jpg', 1200, 800,
			NULL, 'form', NULL,
			42, 1574, NULL, 0, 'before', '1601-1650',
			'loc1', 'period1', '["warm","cool"]',
			'[{"hex":"#1a2b3c","weight":5000},{"hex":"#4d5e6f","weight":3000}]',
			'{"space":"oklab-hcl-12x3x4","bins":[100,200,300]}',
			'oklab-hcl-v1', 'image-hash-abc',
			'Raw source comment.', 'Enriched narrative commentary.'
		);
	`); err != nil {
		t.Fatalf("insert artwork: %v", err)
	}

	artworks, err := loadArtworks(db, map[string]string{
		"#1A2B3C": "Prussian Blue",
		"#4D5E6F": "Slate Blue",
	})
	if err != nil {
		t.Fatalf("load artworks: %v", err)
	}
	if len(artworks) != 1 {
		t.Fatalf("artwork count = %d, want 1", len(artworks))
	}
	item := artworks[0]

	if item.SourceURL != "html/a/artist/work.html" {
		t.Errorf("source url = %q, want html/a/artist/work.html", item.SourceURL)
	}
	if item.SourcePath != "in/art/a/artist/work.jpg" {
		t.Errorf("source path = %q, want in/art/a/artist/work.jpg", item.SourcePath)
	}
	if item.Dimensions != "101 x 201 cm" {
		t.Errorf("dimensions = %q, want 101 x 201 cm", item.Dimensions)
	}

	wantPalette := []sourceColourPaletteEntry{{Name: "Prussian Blue", Hex: "#1a2b3c", Weight: 5000}, {Name: "Slate Blue", Hex: "#4d5e6f", Weight: 3000}}
	if !reflect.DeepEqual(item.ColourPalette, wantPalette) {
		t.Errorf("colour palette = %+v, want %+v", item.ColourPalette, wantPalette)
	}
	if item.ColourSignature == nil {
		t.Fatal("colour signature = nil, want parsed signature")
	}
	if item.ColourSignature.Space != "oklab-hcl-12x3x4" {
		t.Errorf("colour signature space = %q", item.ColourSignature.Space)
	}
	if !reflect.DeepEqual(item.ColourSignature.Bins, []int{100, 200, 300}) {
		t.Errorf("colour signature bins = %v, want [100 200 300]", item.ColourSignature.Bins)
	}
	if item.ColourProfileVersion != "oklab-hcl-v1" {
		t.Errorf("colour profile version = %q, want oklab-hcl-v1", item.ColourProfileVersion)
	}
	if item.ColourImageHash != "image-hash-abc" {
		t.Errorf("colour image hash = %q, want image-hash-abc", item.ColourImageHash)
	}

	if item.SourceComment != "Raw source comment." {
		t.Errorf("source comment = %q, want raw source comment", item.SourceComment)
	}
	if item.Comment != "Enriched narrative commentary." {
		t.Errorf("comment = %q, want enriched commentary", item.Comment)
	}

	// Existing current-location and art-period reads remain intact.
	if item.CurrentLocationID != "loc1" {
		t.Errorf("current location = %q, want loc1", item.CurrentLocationID)
	}
	if item.ArtPeriodID != "period1" {
		t.Errorf("art period = %q, want period1", item.ArtPeriodID)
	}
}

func TestLoadArtworksPreservesEmptyColourAndSourceFields(t *testing.T) {
	db := openArtworkSourceTestDB(t)
	if _, err := db.Exec(artworkSourceSchema); err != nil {
		t.Fatalf("create artworks: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO artworks (
			id, author_id, title, date_text, technique, dimensions, location,
			url, image_path, output_image_path, image_width, image_height,
			school_id, form_id, type_id,
			source_row, date_start, date_end, is_circa, date_qualifier, timeframe_text,
			current_location_id, art_period_id, tone_keywords,
			colour_palette, colour_signature, colour_profile_version, colour_image_hash,
			source_comment, comment
		) VALUES (
			'artwork', 'artist', 'Artwork', '1900', 'Oil', NULL, 'Gallery',
			'html/a/artist/work.html', NULL, 'artwork.jpg', 1200, 800,
			NULL, 'form', NULL,
			0, 1900, NULL, 0, NULL, NULL,
			NULL, NULL, NULL,
			NULL, NULL, NULL, NULL,
			NULL, NULL
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

	if item.ColourPalette != nil {
		t.Errorf("colour palette = %v, want nil", item.ColourPalette)
	}
	if item.ColourSignature != nil {
		t.Errorf("colour signature = %v, want nil", item.ColourSignature)
	}
	if item.ColourProfileVersion != "" || item.ColourImageHash != "" {
		t.Errorf("colour version/hash = %q/%q, want empty", item.ColourProfileVersion, item.ColourImageHash)
	}
	if item.SourceComment != "" || item.Comment != "" {
		t.Errorf("source comment/comment = %q/%q, want empty", item.SourceComment, item.Comment)
	}
	// url is NOT NULL in the producer contract and survives; image_path is
	// nullable and stays empty.
	if item.SourceURL != "html/a/artist/work.html" {
		t.Errorf("source url = %q, want html/a/artist/work.html", item.SourceURL)
	}
	if item.SourcePath != "" {
		t.Errorf("source path = %q, want empty", item.SourcePath)
	}
}

func TestLoadArtworksReadsSourceURLAndPathWithoutNewColumns(t *testing.T) {
	// An older source export that predates the colour-profile and source-comment
	// columns still loads, leaves those fields zero-valued, and preserves the
	// always-present source url/image_path columns.
	db := openArtworkSourceTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE artworks (
			id TEXT PRIMARY KEY,
			author_id TEXT,
			title TEXT,
			date_text TEXT,
			technique TEXT,
			dimensions TEXT,
			location TEXT,
			url TEXT,
			image_path TEXT,
			output_image_path TEXT,
			image_width INTEGER,
			image_height INTEGER,
			school_id TEXT,
			form_id TEXT,
			type_id TEXT
		);
		INSERT INTO artworks VALUES (
			'artwork', 'artist', 'Artwork', '1900', 'Oil', NULL, 'Gallery',
			'html/a/artist/work.html', 'in/art/a/artist/work.jpg', 'artwork.jpg', 1200, 800,
			NULL, 'form', NULL
		);
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

	if item.SourceURL != "html/a/artist/work.html" {
		t.Errorf("source url = %q, want preserved", item.SourceURL)
	}
	if item.SourcePath != "in/art/a/artist/work.jpg" {
		t.Errorf("source path = %q, want preserved", item.SourcePath)
	}
	if item.ColourPalette != nil || item.ColourSignature != nil ||
		item.ColourProfileVersion != "" || item.ColourImageHash != "" ||
		item.SourceComment != "" || item.Comment != "" {
		t.Fatalf("new fields should be zero-valued without their columns, got %+v", item)
	}
}

func TestLoadArtworksRejectsMalformedColourProfile(t *testing.T) {
	db := openArtworkSourceTestDB(t)
	if _, err := db.Exec(artworkSourceSchema); err != nil {
		t.Fatalf("create artworks: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO artworks (
			id, author_id, title, date_text, technique, dimensions, location,
			url, image_path, output_image_path, image_width, image_height,
			school_id, form_id, type_id, colour_palette
		) VALUES (
			'artwork', 'artist', 'Artwork', '1900', 'Oil', NULL, 'Gallery',
			'html/a/artist/work.html', NULL, 'artwork.jpg', 1200, 800,
			NULL, 'form', NULL, 'not-json'
		);
	`); err != nil {
		t.Fatalf("insert artwork: %v", err)
	}

	if _, err := loadArtworks(db); err == nil {
		t.Fatal("expected malformed colour palette to be rejected")
	}
}
