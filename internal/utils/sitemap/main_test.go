package sitemap

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newSitemapTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir(), EncryptionEnv: "test-encryption-key"})
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection(constants.CollectionArtists)
	artists.Id = constants.CollectionArtists
	artists.MarkAsNew()
	artists.Fields.Add(&core.TextField{Name: "name"}, &core.BoolField{Name: "published"})
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection(constants.CollectionArtworks)
	artworks.Id = constants.CollectionArtworks
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.BoolField{Name: "published"},
		&core.RelationField{Name: "author", CollectionId: artists.Id, MaxSelect: 10},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	return app
}

func sitemapConfig(t *testing.T) config.Sitemap {
	t.Helper()
	runtimeConfig := config.LoadFrom(func(key string) string {
		switch key {
		case "WGA_ENV":
			return "test"
		case "WGA_PROTOCOL":
			return "https"
		case "WGA_HOSTNAME":
			return "gallery.example"
		default:
			return ""
		}
	})
	value, err := runtimeConfig.Sitemap()
	if err != nil {
		t.Fatalf("load sitemap config: %v", err)
	}
	return value
}

func createSitemapRecord(t *testing.T, app core.App, collectionName string, values map[string]any) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("find %s collection: %v", collectionName, err)
	}
	record := core.NewRecord(collection)
	for name, value := range values {
		record.Set(name, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s record: %v", collectionName, err)
	}
	return record
}

func TestGenerateSiteMapPublishesCanonicalPublicURLs(t *testing.T) {
	app := newSitemapTestApp(t)
	artist := createSitemapRecord(t, app, constants.CollectionArtists, map[string]any{"name": "Jane Doe", "published": true})
	createSitemapRecord(t, app, constants.CollectionArtists, map[string]any{"name": "Hidden Artist", "published": false})
	artwork := createSitemapRecord(t, app, constants.CollectionArtworks, map[string]any{"title": "Blue Study", "published": true, "author": []string{artist.Id}})
	createSitemapRecord(t, app, constants.CollectionArtworks, map[string]any{"title": "Hidden Work", "published": false, "author": []string{artist.Id}})
	createSitemapRecord(t, app, constants.CollectionArtworks, map[string]any{"title": "Incomplete Work", "published": true})

	result, err := GenerateSiteMap(app, sitemapConfig(t))
	if err != nil {
		t.Fatalf("generate sitemap: %v", err)
	}
	if result.URLCount != 2 || result.ExcludedCount != 1 {
		t.Fatalf("result = %+v, want 2 URLs and 1 exclusion", result)
	}
	if result.CleanupErr != nil {
		t.Fatalf("cleanup sitemap files: %v", result.CleanupErr)
	}

	indexData, err := os.ReadFile(filepath.Join(Directory(app), indexFilename))
	if err != nil {
		t.Fatalf("read sitemap index: %v", err)
	}
	var index sitemapIndex
	if err := xml.Unmarshal(indexData, &index); err != nil {
		t.Fatalf("parse sitemap index: %v", err)
	}
	if len(index.Maps) != 2 {
		t.Fatalf("child map count = %d, want 2", len(index.Maps))
	}
	if !strings.Contains(string(indexData), `<?xml-stylesheet type="text/xsl" href="/sitemap.xsl"?>`) {
		t.Fatal("sitemap index does not reference the sitemap stylesheet")
	}

	allURLs := ""
	for _, entry := range index.Maps {
		filename := filepath.Base(entry.Loc)
		child, err := os.ReadFile(filepath.Join(Directory(app), filename))
		if err != nil {
			t.Fatalf("read child sitemap %s: %v", filename, err)
		}
		allURLs += string(child)
	}
	for _, expected := range []string{
		"https://gallery.example/artists/jane-doe-" + artist.Id,
		"https://gallery.example/artists/jane-doe-" + artist.Id + "/blue-study-" + artwork.Id,
	} {
		if !strings.Contains(allURLs, expected) {
			t.Errorf("sitemap URLs do not contain %q", expected)
		}
	}
	for _, excluded := range []string{"hidden-artist", "hidden-work", "incomplete-work"} {
		if strings.Contains(allURLs, excluded) {
			t.Errorf("sitemap URLs unexpectedly contain %q", excluded)
		}
	}
	if !strings.Contains(allURLs, `<?xml-stylesheet type="text/xsl" href="/sitemap.xsl"?>`) {
		t.Fatal("child sitemap does not reference the sitemap stylesheet")
	}
}

func TestPublishFailureLeavesPreviousIndex(t *testing.T) {
	output := t.TempDir()
	oldIndex := filepath.Join(output, indexFilename)
	if err := os.WriteFile(oldIndex, []byte("old index"), 0o644); err != nil {
		t.Fatalf("write old index: %v", err)
	}
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, indexFilename), []byte("new index"), 0o644); err != nil {
		t.Fatalf("write staged index: %v", err)
	}

	if err := publish(staging, output, map[string]struct{}{"missing.xml": {}}); err == nil {
		t.Fatal("expected child publication error")
	}
	data, err := os.ReadFile(oldIndex)
	if err != nil {
		t.Fatalf("read old index: %v", err)
	}
	if string(data) != "old index" {
		t.Fatalf("published index = %q, want old index", data)
	}
}

func TestPruneRemovesStaleChildMaps(t *testing.T) {
	output := t.TempDir()
	for _, filename := range []string{"current.xml", "stale.xml", indexFilename} {
		if err := os.WriteFile(filepath.Join(output, filename), []byte("xml"), 0o644); err != nil {
			t.Fatalf("write %s: %v", filename, err)
		}
	}
	if err := prune(output, map[string]struct{}{"current.xml": {}}); err != nil {
		t.Fatalf("prune sitemap files: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "stale.xml")); !os.IsNotExist(err) {
		t.Fatalf("stale sitemap still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, indexFilename)); err != nil {
		t.Fatalf("index was pruned: %v", err)
	}
}
