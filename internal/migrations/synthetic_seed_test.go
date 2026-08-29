package migrations

import (
	"database/sql"
	"errors"
	iofs "io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/blackfyre/wga/resources/synthetic"
	"github.com/pocketbase/pocketbase/core"
)

func TestSyntheticSeedMigrationImportsBaselineSchema(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	assertSyntheticCollectionCounts(t, app, map[string]int{
		"schools":        26,
		"art_forms":      14,
		"art_types":      3,
		"art_periods":    32,
		"artists":        10,
		"artworks":       27,
		"art_selections": 10,
		"glossary":       5,
		"guestbook":      2,
		"music_composer": 2,
		"music_song":     3,
		"strings":        7,
		"static_pages":   2,
	})

	for _, collectionName := range []string{"professions", "biographies", "biography_links", "source_attributions"} {
		_, err := app.FindCollectionByNameOrId(collectionName)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no %s collection, got %v", collectionName, err)
		}
	}

	for _, collection := range []struct {
		id   string
		name string
	}{
		{id: "artists", name: "Artists"},
		{id: "artworks", name: "Artworks"},
		{id: "postcards", name: "Postcards"},
		{id: "tracking_postcard_deliveries", name: "postcard_deliveries"},
		{id: "tracking_postcard_delivery_attempts", name: "postcard_delivery_attempts"},
		{id: "tracking_contributor_snapshots", name: "contributor_snapshots"},
		{id: "tracking_contributor_refresh_executions", name: "contributor_refresh_executions"},
	} {
		item, err := app.FindCollectionByNameOrId(collection.id)
		if err != nil {
			t.Fatalf("find %s collection: %v", collection.id, err)
		}
		if got := item.Name; got != collection.name {
			t.Fatalf("expected %s collection name %q, got %q", collection.id, collection.name, got)
		}
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	if got, want := users.Id, "_pb_users_auth_"; got != want {
		t.Fatalf("expected users collection ID %q, got %q", want, got)
	}
	for _, rule := range []*string{users.ListRule, users.ViewRule, users.UpdateRule, users.DeleteRule} {
		if rule == nil || *rule != "id = @request.auth.id" {
			t.Fatal("expected users self-only access rules")
		}
	}

	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	for _, fieldName := range []string{"source_path", "source_hash", "debug_hash", "professions"} {
		if artists.Fields.GetByName(fieldName) != nil {
			t.Fatalf("expected no %s artist field", fieldName)
		}
	}
	assertIndex(t, artists, "pbx_artist_published_name")
	portrait, ok := artists.Fields.GetByName("portrait").(*core.FileField)
	if !ok {
		t.Fatal("expected artist portrait file field")
	}
	if portrait.MaxSelect != 1 || portrait.MaxSize != 5*1024*1024 || !slices.Equal(portrait.Thumbs, []string{"500x0", "600x0"}) {
		t.Fatal("expected single-file artist portrait with visual thumbnail variants")
	}

	schools, err := app.FindCollectionByNameOrId("schools")
	if err != nil {
		t.Fatalf("find schools collection: %v", err)
	}
	assertIndex(t, schools, "idx_unique_school")

	for _, item := range []struct {
		collection string
		field      string
	}{
		{collection: "static_pages", field: "content"},
		{collection: "feedbacks", field: "message"},
	} {
		collection, err := app.FindCollectionByNameOrId(item.collection)
		if err != nil {
			t.Fatalf("find %s collection: %v", item.collection, err)
		}
		field, ok := collection.Fields.GetByName(item.field).(*core.EditorField)
		if !ok || !field.ConvertURLs {
			t.Fatalf("expected %s.%s to convert URLs", item.collection, item.field)
		}
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}
	for _, fieldName := range []string{"date_text", "source_image_path"} {
		if artworks.Fields.GetByName(fieldName) != nil {
			t.Fatalf("expected no %s artwork field", fieldName)
		}
	}
	if artworks.Fields.GetByName("year") == nil {
		t.Fatal("expected artwork year field")
	}
	image, ok := artworks.Fields.GetByName("image").(*core.FileField)
	if !ok {
		t.Fatal("expected artwork image file field")
	}
	if !slices.Equal(image.Thumbs, []string{"120x0", "200x0", "400x0", "500x0", "600x0", "700x0", "800x0", "900x0", "1000x0", "1100x0", "1400x0", "1600x0", "2000x0"}) {
		t.Fatal("expected artwork image field with visual thumbnail variants")
	}
	assertIndex(t, artworks, "pbx_artwork_published_title")

	attempts, err := app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if err != nil {
		t.Fatalf("find postcard attempts collection: %v", err)
	}
	assertIndex(t, attempts, "pbx_postcard_attempt_due")

	period, err := app.FindRecordById("art_periods", "3s9ubtzmmukygar")
	if err != nil {
		t.Fatalf("find art period: %v", err)
	}
	if period.GetString("slug") == "" || period.GetString("description") == "" {
		t.Fatal("expected art period slug and description")
	}

	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	if got, want := artist.GetString("name"), "SYNTHETIC ARTIST 02"; got != want {
		t.Fatalf("expected artist name %q, got %q", want, got)
	}
	if !strings.Contains(artist.GetString("bio"), "Synthetic Artist 02") {
		t.Fatal("expected artist biography HTML to be imported")
	}
	if len(artist.GetStringSlice("school")) != 1 {
		t.Fatal("expected artist school relation")
	}

	selection, err := app.FindRecordById("art_selections", "03f78c6a8fafae1")
	if err != nil {
		t.Fatalf("find synthetic selection: %v", err)
	}
	if !selection.GetBool("published") {
		t.Fatal("expected published synthetic selection")
	}
	if got, want := selection.GetString("commentary"), ""; got != want {
		t.Fatalf("synthetic selection commentary = %q, want %q", got, want)
	}
	if got, want := selection.GetStringSlice("artworks"), []string{"2225c982be1af02", "38311d50a756d76", "4447a2dfa34f956", "9d6478c242f98b2"}; !slices.Equal(got, want) {
		t.Fatalf("synthetic selection artwork order = %v, want %v", got, want)
	}
	for _, fieldName := range []string{"biography_image_width", "biography_image_height"} {
		if _, ok := artists.Fields.GetByName(fieldName).(*core.NumberField); !ok {
			t.Fatalf("expected artists.%s numeric field", fieldName)
		}
	}

	artwork, err := app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	if artwork.GetString("image") == "" {
		t.Fatal("expected artwork image")
	}
	if got, want := artwork.GetInt("year"), 1911; got != want {
		t.Fatalf("expected artwork year %d, got %d", want, got)
	}
	if got, want := artwork.GetInt("image_width"), 512; got != want {
		t.Fatalf("expected artwork image width %d, got %d", want, got)
	}
	if got, want := artwork.GetInt("image_height"), 1024; got != want {
		t.Fatalf("expected artwork image height %d, got %d", want, got)
	}
	for _, fieldName := range []string{"image_width", "image_height", "image_size_bytes"} {
		if _, ok := artworks.Fields.GetByName(fieldName).(*core.NumberField); !ok {
			t.Fatalf("expected artworks.%s numeric field", fieldName)
		}
	}
	embeddedImage, err := synthetic.Files.ReadFile("storage/artworks/07561d2efd0a6db/3a29b540e6908ad8.jpg")
	if err != nil {
		t.Fatalf("read embedded artwork image: %v", err)
	}
	if got, want := artwork.GetInt("image_size_bytes"), len(embeddedImage); got != want {
		t.Fatalf("embedded artwork image_size_bytes = %d, want %d", got, want)
	}

	about, err := app.FindFirstRecordByData("static_pages", "slug", "about")
	if err != nil {
		t.Fatalf("find seeded about page: %v", err)
	}
	if about.GetString("title") != "About" {
		t.Fatalf("seeded about title = %q, want About", about.GetString("title"))
	}

	song, err := app.FindRecordById("music_song", "72d6bb922f76aea")
	if err != nil {
		t.Fatalf("find music track: %v", err)
	}
	if song.GetString("source") == "" {
		t.Fatal("expected music source")
	}
	music, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find music collection: %v", err)
	}
	source, ok := music.Fields.GetByName("source").(*core.FileField)
	if !ok || source.MaxSize != 64*1024*1024 {
		t.Fatal("expected 64 MiB music source file limit")
	}
	glossary, err := app.FindCollectionByNameOrId("glossary")
	if err != nil {
		t.Fatalf("find glossary collection: %v", err)
	}
	definition, ok := glossary.Fields.GetByName("definition").(*core.TextField)
	if !ok || definition.Max != 10000 {
		t.Fatal("expected 10,000-character glossary definition limit")
	}
}

func TestSyntheticSeedMigrationSkipsPopulatedTarget(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	stringsCollection, err := app.FindCollectionByNameOrId("strings")
	if err != nil {
		t.Fatalf("find strings collection: %v", err)
	}
	custom := core.NewRecord(stringsCollection)
	custom.Set("name", "existing")
	custom.Set("content", "Existing content")
	if err := app.Save(custom); err != nil {
		t.Fatalf("create existing content: %v", err)
	}

	if err := seedCurrentSyntheticData(app); err != nil {
		t.Fatalf("skip populated target: %v", err)
	}
	if _, err := app.FindRecordById("strings", custom.Id); err != nil {
		t.Fatalf("find existing content: %v", err)
	}
}

func TestSyntheticSeedImportExternalSQLite(t *testing.T) {
	data, err := synthetic.Files.ReadFile("wga-test.sqlite")
	if err != nil {
		t.Fatalf("read synthetic source: %v", err)
	}
	sourcePath := filepath.Join(t.TempDir(), "wga-src.sqlite")
	if err := os.WriteFile(sourcePath, data, 0o600); err != nil {
		t.Fatalf("write external source: %v", err)
	}
	materializeEmbeddedStorage(t, filepath.Join(filepath.Dir(sourcePath), "storage"))
	source, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		t.Fatalf("open external source: %v", err)
	}
	defer source.Close()
	if _, err := source.Exec(`
		UPDATE artists
		SET biography_image_output_path = 'Artists/2236bdd57f7492e/portrait.jpg',
			biography_image_width = 500,
			biography_image_height = 750
		WHERE id = '2236bdd57f7492e'
	`); err != nil {
		t.Fatalf("add external portrait source data: %v", err)
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
		t.Fatalf("create artwork file byte size field: %v", err)
	}
	if err := seed.Import(app, sourcePath); err != nil {
		t.Fatalf("import external SQLite source: %v", err)
	}

	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find imported artist: %v", err)
	}
	if got, want := artist.GetString("portrait"), "portrait.jpg"; got != want {
		t.Fatalf("external artist portrait = %q, want %q", got, want)
	}
	if got, want := artist.GetInt("biography_image_width"), 500; got != want {
		t.Fatalf("external artist biography image width = %d, want %d", got, want)
	}
	if got, want := artist.GetInt("biography_image_height"), 750; got != want {
		t.Fatalf("external artist biography image height = %d, want %d", got, want)
	}
	if got, want := artist.GetString("filing_name"), "SYNTHETIC ARTIST 02"; got != want {
		t.Fatalf("external artist filing_name = %q, want %q", got, want)
	}
	if got, want := artist.GetString("short_name"), "Synthetic Artist 02"; got != want {
		t.Fatalf("external artist short_name = %q, want %q", got, want)
	}

	artwork, err := app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find imported artwork: %v", err)
	}
	if got, want := artwork.GetInt("image_width"), 512; got != want {
		t.Fatalf("external artwork image width = %d, want %d", got, want)
	}
	if got, want := artwork.GetInt("image_height"), 1024; got != want {
		t.Fatalf("external artwork image height = %d, want %d", got, want)
	}

	embeddedImage, err := synthetic.Files.ReadFile("storage/artworks/07561d2efd0a6db/3a29b540e6908ad8.jpg")
	if err != nil {
		t.Fatalf("read embedded artwork image: %v", err)
	}
	if got, want := artwork.GetInt("image_size_bytes"), len(embeddedImage); got != want {
		t.Fatalf("external artwork image_size_bytes = %d, want %d", got, want)
	}
}

// materializeEmbeddedStorage copies the embedded synthetic storage tree to a
// real filesystem directory so the preseeded external import can stat the
// paired staged originals without reading from the embed FS.
func materializeEmbeddedStorage(t *testing.T, root string) {
	t.Helper()

	storage, err := iofs.Sub(synthetic.Files, "storage")
	if err != nil {
		t.Fatalf("open embedded storage: %v", err)
	}
	if err := iofs.WalkDir(storage, ".", func(p string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(root, p)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := iofs.ReadFile(storage, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o600)
	}); err != nil {
		t.Fatalf("materialize embedded storage: %v", err)
	}
}

func assertSyntheticCollectionCounts(t *testing.T, app core.App, expected map[string]int) {
	t.Helper()

	for collectionName, want := range expected {
		records, err := app.FindRecordsByFilter(collectionName, "", "", 0, 0)
		if err != nil {
			t.Fatalf("find %s records: %v", collectionName, err)
		}
		if got := len(records); got != want {
			t.Fatalf("expected %d %s records, got %d", want, collectionName, got)
		}
	}
}

func assertIndex(t *testing.T, collection *core.Collection, name string) {
	t.Helper()

	for _, index := range collection.Indexes {
		if strings.Contains(index, name) {
			return
		}
	}
	t.Fatalf("expected %s index", name)
}
