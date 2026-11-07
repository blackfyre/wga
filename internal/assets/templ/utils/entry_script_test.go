package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

func TestConfigureEntryScriptVersionsByContent(t *testing.T) {
	t.Cleanup(func() { entryScriptURL.Store(EntryScriptPath) })

	ConfigureEntryScript(fstest.MapFS{"js/app.js": {Data: []byte("release one")}})
	first := EntryScriptURL()
	if !regexp.MustCompile(`^/assets/js/app\.js\?v=[0-9a-f]{12}$`).MatchString(first) {
		t.Fatalf("EntryScriptURL() = %q", first)
	}

	ConfigureEntryScript(fstest.MapFS{"js/app.js": {Data: []byte("release one")}})
	if got := EntryScriptURL(); got != first {
		t.Fatalf("same content changed URL: %q then %q", first, got)
	}

	ConfigureEntryScript(fstest.MapFS{"js/app.js": {Data: []byte("release two")}})
	if got := EntryScriptURL(); got == first {
		t.Fatalf("changed content kept URL %q", got)
	}

	ConfigureEntryScript(fstest.MapFS{})
	if got := EntryScriptURL(); got != EntryScriptPath {
		t.Fatalf("missing entry URL = %q, want %q", got, EntryScriptPath)
	}
}

// Every page that loads the entry module must use EntryScriptURL: a
// hard-coded src keeps the stale-entry failure on pages outside the layout.
func TestNoSourceReferencesTheUnversionedEntryModule(t *testing.T) {
	unversioned := regexp.MustCompile(`src=["']?/assets/js/app\.js`)
	root := filepath.Join("..", "..", "..")
	scanned := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "public" || entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_templ.go") {
			return nil
		}
		if !strings.HasSuffix(name, ".templ") && !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".html") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		if unversioned.Match(content) {
			t.Errorf("%s references the unversioned entry module; use utils.EntryScriptURL()", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 50 {
		t.Fatalf("scanned only %d source files under %s", scanned, root)
	}
}
