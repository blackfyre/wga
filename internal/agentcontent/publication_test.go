package agentcontent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newPublicationTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir(), EncryptionEnv: "test-encryption-key"})
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	schools := core.NewBaseCollection(constants.CollectionSchools)
	schools.Id = constants.CollectionSchools
	schools.MarkAsNew()
	schools.Fields.Add(&core.TextField{Name: "name"})
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools collection: %v", err)
	}

	artists := core.NewBaseCollection(constants.CollectionArtists)
	artists.Id = constants.CollectionArtists
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "published"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "exact_year_of_birth"},
		&core.TextField{Name: "exact_year_of_death"},
		&core.TextField{Name: "profession"},
		&core.TextField{Name: "bio"},
		&core.RelationField{Name: "school", CollectionId: schools.Id, MaxSelect: 10},
	)
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
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "date_end"},
		&core.BoolField{Name: "is_circa"},
		&core.TextField{Name: "date_qualifier"},
		&core.TextField{Name: "timeframe_text"},
		&core.TextField{Name: "technique"},
		&core.TextField{Name: "comment"},
		&core.TextField{Name: "source_comment"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}
	return app
}

func publicationPublicURL(t *testing.T) config.PublicURL {
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
	sitemap, err := runtimeConfig.Sitemap()
	if err != nil {
		t.Fatalf("load sitemap config: %v", err)
	}
	return sitemap.PublicURL
}

func createPublicationRecord(t *testing.T, app core.App, collectionName string, values map[string]any) *core.Record {
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

func TestPublishSelectsCompletePublicAgentContent(t *testing.T) {
	app := newPublicationTestApp(t)
	school := createPublicationRecord(t, app, constants.CollectionSchools, map[string]any{"name": "Dutch School"})
	artist := createPublicationRecord(t, app, constants.CollectionArtists, map[string]any{
		"name": "Jane Doe", "published": true, "year_of_birth": 1600, "year_of_death": 1670,
		"profession": "painter", "school": []string{school.Id}, "bio": "<p>Public biography.</p>",
	})
	hiddenArtist := createPublicationRecord(t, app, constants.CollectionArtists, map[string]any{"name": "Hidden Artist", "published": false})
	artwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Blue Study", "published": true, "author": []string{artist.Id}, "date_start": 1640,
		"technique": "Oil on canvas", "comment": "Blue Study · Gallery, London · 20 × 30 cm", "source_comment": "Public commentary.",
	})
	hiddenArtwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Hidden Work", "published": false, "author": []string{artist.Id},
	})
	createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{"title": "Incomplete Work", "published": true})

	result, err := Publish(app, publicationPublicURL(t))
	if err != nil {
		t.Fatalf("publish agent content: %v", err)
	}
	if result.ArtistCount != 1 || result.ArtworkCount != 1 || result.ExcludedCount != 1 {
		t.Fatalf("result = %+v, want one artist, one artwork, and one exclusion", result)
	}
	if result.CleanupErr != nil {
		t.Fatalf("clean publications: %v", result.CleanupErr)
	}
	current, err := CurrentDirectory(app)
	if err != nil {
		t.Fatalf("resolve current publication: %v", err)
	}
	if current != result.Directory {
		t.Fatalf("current directory = %q, want %q", current, result.Directory)
	}

	assertPublicationContains(t, filepath.Join(current, llmsFilename), "https://gallery.example/sitemap.xml", "/agents/artists/{id}.md")
	assertPublicationContains(t, filepath.Join(current, "agents", "artists", artist.Id+".md"), "# Jane Doe", "Dutch School", "Public biography.")
	assertPublicationContains(t, filepath.Join(current, "agents", "artworks", artwork.Id+".md"), "# Blue Study", "Gallery, London", "20 × 30 cm")
	for _, unavailable := range []string{
		filepath.Join(current, "agents", "artists", hiddenArtist.Id+".md"),
		filepath.Join(current, "agents", "artworks", hiddenArtwork.Id+".md"),
		filepath.Join(current, "agents", "artists", "missing.md"),
	} {
		if _, err := os.Stat(unavailable); !os.IsNotExist(err) {
			t.Fatalf("unavailable generated path %q exists: %v", unavailable, err)
		}
	}
	llms, err := os.ReadFile(filepath.Join(current, llmsFilename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(llms), "Jane Doe") || strings.Contains(string(llms), "Blue Study") {
		t.Fatal("llms.txt embeds catalogue records")
	}
}

func TestPublishFailureLeavesPreviousPublicationSelected(t *testing.T) {
	app := newPublicationTestApp(t)
	artist := createPublicationRecord(t, app, constants.CollectionArtists, map[string]any{"name": "Jane Doe", "published": true})
	if _, err := Publish(app, publicationPublicURL(t)); err != nil {
		t.Fatalf("publish initial agent content: %v", err)
	}
	previous, err := CurrentDirectory(app)
	if err != nil {
		t.Fatal(err)
	}
	previousContent, err := os.ReadFile(filepath.Join(previous, "agents", "artists", artist.Id+".md"))
	if err != nil {
		t.Fatal(err)
	}

	artworks, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Delete(artworks); err != nil {
		t.Fatalf("remove artworks collection: %v", err)
	}
	if _, err := Publish(app, publicationPublicURL(t)); err == nil {
		t.Fatal("publication succeeded without the artworks collection")
	}
	current, err := CurrentDirectory(app)
	if err != nil {
		t.Fatal(err)
	}
	if current != previous {
		t.Fatalf("failed generation selected %q, want previous %q", current, previous)
	}
	content, err := os.ReadFile(filepath.Join(current, "agents", "artists", artist.Id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(previousContent) {
		t.Fatal("failed generation changed the previous publication")
	}
}

func TestPublishPrunesStaleGeneratedRecords(t *testing.T) {
	app := newPublicationTestApp(t)
	artist := createPublicationRecord(t, app, constants.CollectionArtists, map[string]any{"name": "Jane Doe", "published": true})
	artwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Blue Study", "published": true, "author": []string{artist.Id},
	})
	first, err := Publish(app, publicationPublicURL(t))
	if err != nil {
		t.Fatal(err)
	}
	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatal(err)
	}
	second, err := Publish(app, publicationPublicURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if first.Directory == second.Directory {
		t.Fatal("publication directory was not versioned")
	}
	if _, err := os.Stat(filepath.Join(second.Directory, "agents", "artworks", artwork.Id+".md")); !os.IsNotExist(err) {
		t.Fatalf("stale artwork remains in current publication: %v", err)
	}
	if _, err := os.Stat(first.Directory); !os.IsNotExist(err) {
		t.Fatalf("previous publication was not pruned: %v", err)
	}
}

func assertPublicationContains(t *testing.T, filename string, expected ...string) {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	for _, value := range expected {
		if !strings.Contains(string(data), value) {
			t.Errorf("%s does not contain %q:\n%s", filename, value, data)
		}
	}
}
