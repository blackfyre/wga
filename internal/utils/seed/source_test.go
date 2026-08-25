package seed

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

			schema := `CREATE TABLE artists (id TEXT PRIMARY KEY, source_display_name TEXT, birth_year INTEGER, death_year INTEGER, birth_place TEXT, death_place TEXT, biography_image_width INTEGER, biography_image_height INTEGER`
			if test.hasOutputPath {
				schema += `, biography_image_output_path TEXT`
			}
			schema += `)`
			if _, err := db.Exec(schema); err != nil {
				t.Fatalf("create artists: %v", err)
			}

			if test.hasOutputPath {
				_, err = db.Exec(`INSERT INTO artists VALUES ('artist', 'Artist', NULL, NULL, NULL, NULL, ?, ?, ?)`, test.width, test.height, test.portrait)
			} else {
				_, err = db.Exec(`INSERT INTO artists VALUES ('artist', 'Artist', NULL, NULL, NULL, NULL, ?, ?)`, test.width, test.height)
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
	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		musicFiles:   map[string]sourceFile{},
		artworks: []sourceArtwork{
			{ID: "rwork0000000001", ImagePath: ""},
			{ID: "rwork0000000002", ImagePath: "artworks/rwork0000000002/image.jpg"},
		},
	}

	if err := loadPreseededSourceFiles(&data); err != nil {
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
}

func TestLoadPreseededSourceFilesRejectsUnsafeArtworkPath(t *testing.T) {
	data := sourceData{
		artworkFiles: map[string]sourceFile{},
		musicFiles:   map[string]sourceFile{},
		artworks: []sourceArtwork{
			{ID: "rwork0000000001", ImagePath: "../image.jpg"},
		},
	}

	if err := loadPreseededSourceFiles(&data); err == nil {
		t.Fatal("expected unsafe artwork path error")
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
