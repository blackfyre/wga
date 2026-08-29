package migrations

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/blackfyre/wga/resources/synthetic"
)

func TestReleaseDataPreflightExternalSourcePreservesReleaseData(t *testing.T) {
	data, err := synthetic.Files.ReadFile("wga-test.sqlite")
	if err != nil {
		t.Fatalf("read synthetic source: %v", err)
	}

	sourcePath := filepath.Join(t.TempDir(), "external-source.sqlite")
	if err := os.WriteFile(sourcePath, data, 0o600); err != nil {
		t.Fatalf("write external source: %v", err)
	}
	materializeEmbeddedStorage(t, filepath.Join(filepath.Dir(sourcePath), "storage"))

	source, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		t.Fatalf("open external source: %v", err)
	}
	defer source.Close()

	comment := strings.Repeat("release-source-comment-", 300)
	if _, err := source.Exec("UPDATE artworks SET source_comment = ? WHERE id = ?", comment, "07561d2efd0a6db"); err != nil {
		t.Fatalf("stage long source comment: %v", err)
	}

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtistSelections(app); err != nil {
		t.Fatalf("create artist selections collection: %v", err)
	}
	if err := addArtistIdentityFields(app); err != nil {
		t.Fatalf("create artist identity fields: %v", err)
	}
	if err := addCollectionData(app); err != nil {
		t.Fatalf("create collection-data collections/fields: %v", err)
	}
	if err := addArtworkSourceFields(app); err != nil {
		t.Fatalf("create artwork source/colour fields: %v", err)
	}
	if err := addArtworkFileByteSize(app); err != nil {
		t.Fatalf("create artwork release-data fields: %v", err)
	}

	if err := seed.Import(app, sourcePath); err != nil {
		t.Fatalf("import external SQLite source: %v", err)
	}

	artwork, err := app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find imported artwork: %v", err)
	}
	if got := artwork.GetString("source_comment"); got != comment {
		t.Fatalf("source_comment = %q, want exact %q", got, comment)
	}

	stagedPath := filepath.Join(filepath.Dir(sourcePath), "storage", "artworks", "07561d2efd0a6db", "3a29b540e6908ad8.jpg")
	stagedInfo, err := os.Stat(stagedPath)
	if err != nil {
		t.Fatalf("stat externally staged artwork: %v", err)
	}
	if got, want := artwork.GetInt("image_size_bytes"), stagedInfo.Size(); int64(got) != want {
		t.Fatalf("image_size_bytes = %d, want os.Stat size %d", got, want)
	}
	if got, want := artwork.GetInt("image_width"), 512; got != want {
		t.Fatalf("image_width = %d, want source dimension %d", got, want)
	}
	if got, want := artwork.GetInt("image_height"), 1024; got != want {
		t.Fatalf("image_height = %d, want source dimension %d", got, want)
	}

	if err := seedCurrentSyntheticData(app); err != nil {
		t.Fatalf("rerun seed migration equivalent: %v", err)
	}
	if got := artwork.GetString("source_comment"); got != comment {
		t.Fatalf("source_comment after no-op rerun = %q, want exact %q", got, comment)
	}
}
