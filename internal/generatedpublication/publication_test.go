package generatedpublication

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestFailedPublicationLeavesAllReadersOnPreviousVersion(t *testing.T) {
	app, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir(), EncryptionEnv: "test-encryption-key"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	initial, err := NewStaging(app)
	if err != nil {
		t.Fatal(err)
	}
	writeGeneratedTestFile(t, initial, "sitemap/sitemap.xml", "old sitemap")
	writeGeneratedTestFile(t, initial, "agent-content/agents/artists/a.md", "old artist")
	previous, err := Publish(app, initial, "previous")
	if err != nil {
		t.Fatal(err)
	}

	next, err := NewStaging(app)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(next)
	writeGeneratedTestFile(t, next, "sitemap/sitemap.xml", "new sitemap")
	writeGeneratedTestFile(t, next, "agent-content/agents/artists/a.md", "new artist")
	conflict := filepath.Join(Directory(app), versionsName, "next")
	if err := os.WriteFile(conflict, []byte("blocks rename"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Publish(app, next, "next"); err == nil {
		t.Fatal("publication unexpectedly succeeded")
	}
	current, err := CurrentDirectory(app)
	if err != nil {
		t.Fatal(err)
	}
	if current != previous {
		t.Fatalf("current publication = %q, want %q", current, previous)
	}
	for relative, want := range map[string]string{
		"sitemap/sitemap.xml":               "old sitemap",
		"agent-content/agents/artists/a.md": "old artist",
	} {
		data, err := os.ReadFile(filepath.Join(current, filepath.FromSlash(relative)))
		if err != nil || string(data) != want {
			t.Fatalf("current %s = %q, %v; want %q", relative, data, err, want)
		}
	}
}

func writeGeneratedTestFile(t *testing.T, root string, relative string, content string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
