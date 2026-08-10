package seed

import (
	"database/sql"
	"os"
	"path/filepath"
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
