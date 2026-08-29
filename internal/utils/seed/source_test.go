package seed

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadColourNamesUsesCanonicalHexKeys(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer closeDatabase(db)

	if _, err := db.Exec(`CREATE TABLE colour_names (hex_code TEXT, name TEXT); INSERT INTO colour_names VALUES ('#1a2b3c', 'Prussian Blue');`); err != nil {
		t.Fatalf("create colour names: %v", err)
	}

	names, err := loadColourNames(db)
	if err != nil {
		t.Fatalf("load colour names: %v", err)
	}
	if got := names["#1A2B3C"]; got != "Prussian Blue" {
		t.Errorf("colour name = %q, want Prussian Blue", got)
	}
}

func TestLoadBiographiesUsesArtistFieldsWhenLegacyTableIsAbsent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer closeDatabase(db)

	if _, err := db.Exec(`
		CREATE TABLE artists (
			id TEXT PRIMARY KEY,
			raw_biography_html TEXT NOT NULL,
			updated_biography_html TEXT NOT NULL,
			enriched_biography_html TEXT
		);
		INSERT INTO artists VALUES ('artist', '<p>raw</p>', '<p>updated</p>', '<p>enriched</p>');
	`); err != nil {
		t.Fatalf("create artist: %v", err)
	}

	biographies, err := loadBiographies(db)
	if err != nil {
		t.Fatalf("load biographies: %v", err)
	}
	if len(biographies) != 1 {
		t.Fatalf("biography count = %d, want 1", len(biographies))
	}
	if got, want := biographies[0].BiographyHTML, "<p>enriched</p>"; got != want {
		t.Fatalf("biography HTML = %q, want %q", got, want)
	}
}

func TestLoadArtistsPortraitPath(t *testing.T) {
	tests := []struct {
		name          string
		hasOutputPath bool
		portrait      string
		want          string
		wantErr       bool
		width         int
		height        int
	}{
		{name: "without required portrait output column", wantErr: true, width: 500, height: 750},
		{name: "with portrait path", hasOutputPath: true, portrait: "artists/artist/portrait.jpg", want: "portrait.jpg", width: 500, height: 750},
		{name: "with unsafe portrait path", hasOutputPath: true, portrait: "../portrait.jpg", wantErr: true, width: 500, height: 750},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("open database: %v", err)
			}
			defer closeDatabase(db)

			schema := `CREATE TABLE artists (id TEXT PRIMARY KEY, source_display_name TEXT, display_name TEXT, birth_year INTEGER, death_year INTEGER, birth_place TEXT, death_place TEXT, biography_image_width INTEGER, biography_image_height INTEGER`
			if test.hasOutputPath {
				schema += `, biography_image_output_path TEXT`
			}
			schema += `)`
			if _, err := db.Exec(schema); err != nil {
				t.Fatalf("create artists: %v", err)
			}

			if test.hasOutputPath {
				_, err = db.Exec(`INSERT INTO artists VALUES ('artist', 'ARTIST, Example', 'Example Artist', NULL, NULL, NULL, NULL, ?, ?, ?)`, test.width, test.height, test.portrait)
			} else {
				_, err = db.Exec(`INSERT INTO artists VALUES ('artist', 'ARTIST, Example', 'Example Artist', NULL, NULL, NULL, NULL, ?, ?)`, test.width, test.height)
			}
			if err != nil {
				t.Fatalf("insert artist: %v", err)
			}

			artists, err := loadArtists(db)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected portrait path error")
				}
				if !test.hasOutputPath && !strings.Contains(err.Error(), "biography_image_output_path") {
					t.Fatalf("expected missing portrait output column error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("load artists: %v", err)
			}
			if got := artists[0].Portrait; got != test.want {
				t.Fatalf("portrait = %q, want %q", got, test.want)
			}
			if got := artists[0].BiographyImageWidth; got != test.width {
				t.Fatalf("biography image width = %d, want %d", got, test.width)
			}
			if got := artists[0].BiographyImageHeight; got != test.height {
				t.Fatalf("biography image height = %d, want %d", got, test.height)
			}
		})
	}
}

func TestLoadArtistsCarriesFilingAndShortNamesVerbatim(t *testing.T) {
	tests := []struct {
		name       string
		filingName string
		shortName  string
	}{
		{name: "comma filing form", filingName: "VERMEER, Johannes", shortName: "Johannes Vermeer"},
		{name: "mononym", filingName: "MODERNO", shortName: "Moderno"},
		{name: "filing without comma", filingName: "LEONARDO da Vinci", shortName: "Leonardo da Vinci"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("open database: %v", err)
			}
			defer closeDatabase(db)

			if _, err := db.Exec(`
				CREATE TABLE artists (
					id TEXT PRIMARY KEY,
					source_display_name TEXT,
					display_name TEXT,
					birth_year INTEGER,
					death_year INTEGER,
					birth_place TEXT,
					death_place TEXT,
					biography_image_output_path TEXT,
					biography_image_width INTEGER,
					biography_image_height INTEGER
				);
			`); err != nil {
				t.Fatalf("create artists: %v", err)
			}
			if _, err := db.Exec(
				`INSERT INTO artists VALUES ('artist', ?, ?, NULL, NULL, NULL, NULL, NULL, 500, 750)`,
				test.filingName, test.shortName,
			); err != nil {
				t.Fatalf("insert artist: %v", err)
			}

			artists, err := loadArtists(db)
			if err != nil {
				t.Fatalf("load artists: %v", err)
			}
			if len(artists) != 1 {
				t.Fatalf("artist count = %d, want 1", len(artists))
			}
			if got, want := artists[0].DisplayName, test.filingName; got != want {
				t.Fatalf("display (filing) name = %q, want %q verbatim", got, want)
			}
			if got, want := artists[0].ShortName, test.shortName; got != want {
				t.Fatalf("short name = %q, want %q verbatim", got, want)
			}
		})
	}
}

func TestLoadArtistsRejectsBlankIdentityFields(t *testing.T) {
	tests := []struct {
		name       string
		filingName string
		shortName  string
		wantField  string
	}{
		{name: "blank filing name", filingName: "   ", shortName: "Moderno", wantField: "source_display_name"},
		{name: "blank short name", filingName: "MODERNO", shortName: "", wantField: "display_name"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("open database: %v", err)
			}
			defer closeDatabase(db)

			if _, err := db.Exec(`
				CREATE TABLE artists (
					id TEXT PRIMARY KEY,
					source_display_name TEXT,
					display_name TEXT,
					birth_year INTEGER,
					death_year INTEGER,
					birth_place TEXT,
					death_place TEXT,
					biography_image_output_path TEXT,
					biography_image_width INTEGER,
					biography_image_height INTEGER
				);
			`); err != nil {
				t.Fatalf("create artists: %v", err)
			}
			if _, err := db.Exec(
				`INSERT INTO artists VALUES ('artist', ?, ?, NULL, NULL, NULL, NULL, NULL, 500, 750)`,
				test.filingName, test.shortName,
			); err != nil {
				t.Fatalf("insert artist: %v", err)
			}

			_, err = loadArtists(db)
			if err == nil {
				t.Fatal("expected blank identity field error")
			}
			if !strings.Contains(err.Error(), test.wantField) {
				t.Fatalf("expected error mentioning %q, got %v", test.wantField, err)
			}
		})
	}
}

func TestLoadArtistsRejectsMissingShortNameColumn(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer closeDatabase(db)

	// A pre-identity export lacks the display_name column entirely.
	if _, err := db.Exec(`
		CREATE TABLE artists (
			id TEXT PRIMARY KEY,
			source_display_name TEXT,
			birth_year INTEGER,
			death_year INTEGER,
			birth_place TEXT,
			death_place TEXT,
			biography_image_output_path TEXT,
			biography_image_width INTEGER,
			biography_image_height INTEGER
		);
	`); err != nil {
		t.Fatalf("create artists: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO artists VALUES ('artist', 'ARTIST, Example', NULL, NULL, NULL, NULL, NULL, 500, 750)`); err != nil {
		t.Fatalf("insert artist: %v", err)
	}

	if _, err := loadArtists(db); err == nil {
		t.Fatal("expected missing display_name column error")
	}
}

func TestLoadArtworksImageDimensions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer closeDatabase(db)

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
	if got, want := artworks[0].ImageWidth, 1200; got != want {
		t.Fatalf("image width = %d, want %d", got, want)
	}
	if got, want := artworks[0].ImageHeight, 800; got != want {
		t.Fatalf("image height = %d, want %d", got, want)
	}
}

func TestExternalSourcePaths(t *testing.T) {
	root := t.TempDir()
	sqlitePath := filepath.Join(root, "wga-src.sqlite")
	if err := os.WriteFile(sqlitePath, []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatalf("write seed database: %v", err)
	}
	paths, err := externalSourcePaths(sqlitePath)
	if err != nil {
		t.Fatalf("external source paths: %v", err)
	}
	if paths.sqlitePath != sqlitePath {
		t.Fatalf("SQLite path = %q, want %q", paths.sqlitePath, sqlitePath)
	}
	if !paths.preseededAssets {
		t.Fatal("expected external source to use preseeded assets")
	}
	info, err := os.Stat(paths.storageRoot)
	if err != nil {
		t.Fatalf("stat created storage root: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("storage root %q is not a directory", paths.storageRoot)
	}
}

func TestPreseededSourceFile(t *testing.T) {
	file, err := preseededSourceFile("Artworks/artwork/image.jpg")
	if err != nil {
		t.Fatalf("preseeded source file: %v", err)
	}
	if got, want := file.name, "image.jpg"; got != want {
		t.Fatalf("file name = %q, want %q", got, want)
	}
	if !file.preseededAssets {
		t.Fatal("expected preseeded asset")
	}
}

func TestLoadPreseededSourceFilesSkipsImageLessArtwork(t *testing.T) {
	root := t.TempDir()
	content := []byte("staged artwork bytes")
	filePath := filepath.Join(root, "artworks", "rwork0000000002", "image.jpg")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("write staged original: %v", err)
	}

	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		musicFiles:   map[string]sourceFile{},
		artworks: []sourceArtwork{
			{ID: "rwork0000000001", ImagePath: ""},
			{ID: "rwork0000000002", ImagePath: "artworks/rwork0000000002/image.jpg"},
		},
	}

	if err := loadPreseededSourceFiles(root, false, &data); err != nil {
		t.Fatalf("load preseeded source files: %v", err)
	}

	if _, ok := data.artworkFiles["rwork0000000001"]; ok {
		t.Fatal("image-less artwork should have no source file entry")
	}
	file, ok := data.artworkFiles["rwork0000000002"]
	if !ok {
		t.Fatal("artwork with an image path should have a source file entry")
	}
	if got, want := file.name, "image.jpg"; got != want {
		t.Fatalf("file name = %q, want %q", got, want)
	}
	if got, want := file.size, int64(len(content)); got != want {
		t.Fatalf("file size = %d, want %d", got, want)
	}
}

func TestLoadPreseededSourceFilesRejectsUnsafeArtworkPath(t *testing.T) {
	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		musicFiles:   map[string]sourceFile{},
		artworks: []sourceArtwork{
			{ID: "rwork0000000001", ImagePath: "../image.jpg"},
		},
	}

	if err := loadPreseededSourceFiles(t.TempDir(), false, &data); err == nil {
		t.Fatal("expected unsafe artwork path error")
	}
}

func TestPreseededArtworkFileRecordsStagedByteSize(t *testing.T) {
	root := t.TempDir()
	content := []byte("staged original image bytes")
	filePath := filepath.Join(root, "artworks", "rwork0000000001", "image.jpg")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("write staged original: %v", err)
	}

	file, err := preseededArtworkFile(root, false, "artworks/rwork0000000001/image.jpg")
	if err != nil {
		t.Fatalf("preseeded artwork file: %v", err)
	}
	if got, want := file.name, "image.jpg"; got != want {
		t.Fatalf("file name = %q, want %q", got, want)
	}
	if got, want := file.size, int64(len(content)); got != want {
		t.Fatalf("file size = %d, want %d", got, want)
	}
	if !file.preseededAssets {
		t.Fatal("expected preseeded asset")
	}
}

func TestPreseededArtworkFileWithConfiguredStorageSkipsLocalStaging(t *testing.T) {
	storagePath := "artworks/rwork0000000001/image.jpg"
	file, err := preseededArtworkFile(t.TempDir(), true, storagePath)
	if err != nil {
		t.Fatalf("preseeded artwork file: %v", err)
	}
	if got, want := file.name, "image.jpg"; got != want {
		t.Fatalf("file name = %q, want %q", got, want)
	}
	if file.size != 0 {
		t.Fatalf("file size = %d, want zero without a local staged original", file.size)
	}
}

func TestPreseededArtworkFileRejectsMissingOriginal(t *testing.T) {
	if _, err := preseededArtworkFile(t.TempDir(), false, "artworks/rwork0000000001/image.jpg"); err == nil {
		t.Fatal("expected missing staged original error")
	}
}

func TestPreseededArtworkFileRejectsNonRegularOriginal(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "artworks", "rwork0000000001", "image.jpg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := preseededArtworkFile(root, false, "artworks/rwork0000000001/image.jpg"); err == nil {
		t.Fatal("expected non-regular staged original error")
	}
}

func TestPreseededArtworkFileRejectsEmptyOriginal(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "artworks", "rwork0000000001", "image.jpg")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte{}, 0o600); err != nil {
		t.Fatalf("write empty staged original: %v", err)
	}

	if _, err := preseededArtworkFile(root, false, "artworks/rwork0000000001/image.jpg"); err == nil {
		t.Fatal("expected empty staged original error")
	}
}

func TestPreseededArtworkFileRejectsFileSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "escaped.jpg")
	if err := os.WriteFile(outside, []byte("outside root bytes"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	linkDir := filepath.Join(root, "artworks", "rwork0000000001")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(linkDir, "image.jpg")); err != nil {
		t.Fatalf("symlink file: %v", err)
	}

	if _, err := preseededArtworkFile(root, false, "artworks/rwork0000000001/image.jpg"); err == nil {
		t.Fatal("expected file symlink escape error")
	}
}

func TestPreseededArtworkFileRejectsParentSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "image.jpg"), []byte("outside parent bytes"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	parent := filepath.Join(root, "artworks")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir artworks: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(parent, "rwork0000000001")); err != nil {
		t.Fatalf("symlink parent: %v", err)
	}

	if _, err := preseededArtworkFile(root, false, "artworks/rwork0000000001/image.jpg"); err == nil {
		t.Fatal("expected parent-directory symlink escape error")
	}
}

func TestLoadEmbeddedSourceFilesRecordsByteSize(t *testing.T) {
	content := []byte("embedded source image bytes")
	storage := fstest.MapFS{
		"Artworks/rwork0000000001/image.jpg": &fstest.MapFile{Data: content},
	}
	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		musicFiles:   map[string]sourceFile{},
		artworks: []sourceArtwork{
			{ID: "rwork0000000001"},
		},
	}

	if err := loadEmbeddedSourceFiles(storage, &data); err != nil {
		t.Fatalf("load embedded source files: %v", err)
	}
	file, ok := data.artworkFiles["rwork0000000001"]
	if !ok {
		t.Fatal("artwork should have a source file entry")
	}
	if got, want := file.size, int64(len(content)); got != want {
		t.Fatalf("file size = %d, want %d", got, want)
	}
	if !bytes.Equal(file.content, content) {
		t.Fatal("file content should match embedded bytes")
	}
}

func TestSafeRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "nested path", value: "music/track.mp3", want: "music/track.mp3"},
		{name: "normalised nested path", value: "music/../music/track.mp3", want: "music/track.mp3"},
		{name: "empty path", value: "", wantErr: true},
		{name: "absolute path", value: "/music/track.mp3", wantErr: true},
		{name: "parent directory", value: "..", wantErr: true},
		{name: "parent traversal", value: "../music/track.mp3", wantErr: true},
		{name: "normalised traversal", value: "music/../../track.mp3", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := safeRelativePath(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("safeRelativePath(%q): %v", test.value, err)
			}
			if got != test.want {
				t.Fatalf("safeRelativePath(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
