package artists

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/agentcontent"
	"github.com/blackfyre/wga/internal/generatedpublication"
	"github.com/pocketbase/pocketbase/core"
)

func publishGeneratedMarkdownFixture(t *testing.T, app core.App, relative string, canonicalURL string, acceptedPaths []string) {
	t.Helper()
	staging, err := generatedpublication.NewStaging(app)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(staging, agentcontent.PublicationDirectoryName)
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Generated Markdown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(map[string]any{
		"resources":      map[string]string{relative: canonicalURL},
		"accepted_paths": map[string][]string{relative: acceptedPaths},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := generatedpublication.Publish(app, staging, "markdown-fixture"); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedMarkdownPath(t *testing.T) {
	if got := generatedMarkdownPath("artists", "synthetic-artist-artistone000001"); got != "/agents/artists/artistone000001.md" {
		t.Fatalf("generatedMarkdownPath = %q", got)
	}
}

func TestGeneratedMarkdownAvailableRequiresCurrentCanonicalPath(t *testing.T) {
	app := newArtistRecordApp(t)
	markdownPath := "/agents/artists/artistone000001.md"
	oldPath := "/artists/old-name-artistone000001"
	publishGeneratedMarkdownFixture(t, app, strings.TrimPrefix(markdownPath, "/"), "https://gallery.example"+oldPath, []string{oldPath})

	if !generatedMarkdownAvailable(app, markdownPath, oldPath) {
		t.Fatal("current generated Markdown was unavailable")
	}
	if generatedMarkdownAvailable(app, markdownPath, "/artists/new-name-artistone000001") {
		t.Fatal("stale generated Markdown was advertised for a changed canonical path")
	}
	if generatedMarkdownAvailable(app, "/agents/artists/missing.md", oldPath) {
		t.Fatal("missing generated Markdown was available")
	}
}
