package agentcontent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/generatedpublication"
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
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "slug"},
		&core.BoolField{Name: "published"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.BoolField{Name: "exact_year_of_birth"},
		&core.BoolField{Name: "exact_year_of_death"},
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

func publishAgentContent(t *testing.T, app core.App) (PublicationResult, error) {
	t.Helper()
	staging, err := generatedpublication.NewStaging(app)
	if err != nil {
		return PublicationResult{}, err
	}
	defer os.RemoveAll(staging)
	result, err := Generate(app, publicationPublicURL(t), filepath.Join(staging, publicationDirectoryName))
	if err != nil {
		return PublicationResult{}, err
	}
	version := "test-" + strings.TrimPrefix(filepath.Base(staging), ".staging-")
	published, err := generatedpublication.Publish(app, staging, version)
	if err != nil {
		return PublicationResult{}, err
	}
	result.Directory = filepath.Join(published, publicationDirectoryName)
	result.CleanupErr = generatedpublication.Prune(app, version)
	return result, nil
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
		"name": "Jane Doe", "filing_name": "Doe, Jane", "short_name": "Jane", "slug": "authoritative-jane", "published": true,
		"year_of_birth": 1600, "year_of_death": 1670, "exact_year_of_birth": true, "exact_year_of_death": false,
		"profession": "painter", "school": []string{school.Id}, "bio": "<p>Public biography.</p>",
	})
	hiddenArtist := createPublicationRecord(t, app, constants.CollectionArtists, publicationArtist("Hidden Artist", "hidden-artist", false))
	incompleteArtist := createPublicationRecord(t, app, constants.CollectionArtists, map[string]any{
		"name": "Incomplete Artist", "filing_name": "Artist, Incomplete", "slug": "incomplete-artist", "published": true,
	})
	artwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Blue Study", "published": true, "author": []string{artist.Id}, "date_start": 1640,
		"technique": "Oil on canvas", "comment": "Blue Study · Gallery, London · 20 × 30 cm", "source_comment": "Public commentary.",
	})
	hiddenArtwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Hidden Work", "published": false, "author": []string{artist.Id},
	})
	createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{"title": "Incomplete Work", "published": true})
	coauthoredArtwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Shared Study", "published": true, "author": []string{hiddenArtist.Id, artist.Id},
	})

	result, err := publishAgentContent(t, app)
	if err != nil {
		t.Fatalf("publish agent content: %v", err)
	}
	if result.ArtistCount != 1 || result.ArtworkCount != 2 || result.ExcludedCount != 1 {
		t.Fatalf("result = %+v, want one artist, two artworks, and one exclusion", result)
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
	assertPublicationContains(t, filepath.Join(current, "agents", "artists", artist.Id+".md"), "# Jane Doe", "Dutch School", "Public biography.", "Lifespan: 1600–c. 1670")
	assertPublicationContains(t, filepath.Join(current, "agents", "artworks", artwork.Id+".md"), "# Blue Study", "Gallery, London", "20 × 30 cm")
	assertPublicationContains(t, filepath.Join(current, "agents", "artworks", coauthoredArtwork.Id+".md"), "# Shared Study", "/artists/authoritative-jane-"+artist.Id)
	resource, err := ReadCurrent(app, "agents/artists/"+artist.Id+".md")
	if err != nil {
		t.Fatalf("read current artist content: %v", err)
	}
	if resource.CanonicalURL != "https://gallery.example/artists/authoritative-jane-"+artist.Id || !strings.Contains(string(resource.Content), "# Jane Doe") {
		t.Fatalf("current artist resource = %+v", resource)
	}
	for _, unavailable := range []string{
		filepath.Join(current, "agents", "artists", hiddenArtist.Id+".md"),
		filepath.Join(current, "agents", "artists", incompleteArtist.Id+".md"),
		filepath.Join(current, "agents", "artworks", hiddenArtwork.Id+".md"),
		filepath.Join(current, "agents", "artists", "missing.md"),
	} {
		if _, err := os.Stat(unavailable); !os.IsNotExist(err) {
			t.Fatalf("unavailable generated path %q exists: %v", unavailable, err)
		}
	}
	if _, err := ReadCurrent(app, "agents/artists/missing.md"); !os.IsNotExist(err) {
		t.Fatalf("missing current resource error = %v, want not exist", err)
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
	artist := createPublicationRecord(t, app, constants.CollectionArtists, publicationArtist("Jane Doe", "jane-doe", true))
	if _, err := publishAgentContent(t, app); err != nil {
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
	if _, err := publishAgentContent(t, app); err == nil {
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

func TestPublishedArtworkAcceptsEveryPublicCoauthorPath(t *testing.T) {
	app := newPublicationTestApp(t)
	first := createPublicationRecord(t, app, constants.CollectionArtists, publicationArtist("First Author", "first-author", true))
	second := createPublicationRecord(t, app, constants.CollectionArtists, publicationArtist("Second Author", "second-author", true))
	artwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Shared Work", "published": true, "author": []string{first.Id, second.Id},
	})
	if _, err := publishAgentContent(t, app); err != nil {
		t.Fatal(err)
	}

	resource, err := ReadCurrent(app, "agents/artworks/"+artwork.Id+".md")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		canonicalArtworkPath(first, artwork),
		canonicalArtworkPath(second, artwork),
	}
	if strings.Join(resource.AcceptedPaths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("accepted paths = %v, want %v", resource.AcceptedPaths, want)
	}
}

func TestMissingResourceDoesNotParsePublicationManifest(t *testing.T) {
	app := newPublicationTestApp(t)
	createPublicationRecord(t, app, constants.CollectionArtists, publicationArtist("Jane Doe", "jane-doe", true))
	result, err := publishAgentContent(t, app)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(result.Directory, manifestFilename), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadCurrent(app, "agents/artists/missing.md"); !os.IsNotExist(err) {
		t.Fatalf("missing resource error = %v, want not exist without manifest parsing", err)
	}
}

func TestPublishPrunesStaleGeneratedRecords(t *testing.T) {
	app := newPublicationTestApp(t)
	artist := createPublicationRecord(t, app, constants.CollectionArtists, publicationArtist("Jane Doe", "jane-doe", true))
	artwork := createPublicationRecord(t, app, constants.CollectionArtworks, map[string]any{
		"title": "Blue Study", "published": true, "author": []string{artist.Id},
	})
	first, err := publishAgentContent(t, app)
	if err != nil {
		t.Fatal(err)
	}
	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatal(err)
	}
	second, err := publishAgentContent(t, app)
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

func publicationArtist(name string, slug string, published bool) map[string]any {
	return map[string]any{
		"name": name, "filing_name": name, "short_name": name, "slug": slug, "published": published,
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
