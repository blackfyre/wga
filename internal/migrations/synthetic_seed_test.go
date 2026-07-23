package migrations

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSyntheticSeedMigrationImportsExistingSchema(t *testing.T) {
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
		"schools":        1,
		"art_forms":      1,
		"art_types":      1,
		"artists":        10,
		"artworks":       27,
		"glossary":       5,
		"guestbook":      2,
		"music_composer": 2,
		"music_song":     3,
		"strings":        7,
		"static_pages":   1,
	})

	for _, collectionName := range []string{"professions", "biographies", "biography_links", "source_attributions"} {
		_, err := app.FindCollectionByNameOrId(collectionName)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no %s collection, got %v", collectionName, err)
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

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}
	for _, fieldName := range []string{"date_text", "source_url", "source_image_path"} {
		if artworks.Fields.GetByName(fieldName) != nil {
			t.Fatalf("expected no %s artwork field", fieldName)
		}
	}

	songs, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find music songs collection: %v", err)
	}
	sourceField, ok := songs.Fields.GetByName("source").(*core.FileField)
	if !ok {
		t.Fatal("expected music source file field")
	}
	if got, want := sourceField.MaxSize, int64(syntheticMusicSourceMaxSize); got != want {
		t.Fatalf("expected music source limit %d, got %d", want, got)
	}

	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	if got, want := artist.GetString("name"), "Synthetic Artist 02"; got != want {
		t.Fatalf("expected artist name %q, got %q", want, got)
	}
	if !strings.Contains(artist.GetString("bio"), "Synthetic Artist 02") {
		t.Fatal("expected artist biography HTML to be imported")
	}
	if len(artist.GetStringSlice("school")) != 1 {
		t.Fatal("expected artist school relation")
	}
	if artist.GetString("profession") == "" {
		t.Fatal("expected artist profession text")
	}

	glossary, err := app.FindRecordById("glossary", "0b0a0a50235e4f9")
	if err != nil {
		t.Fatalf("find glossary entry: %v", err)
	}
	if got, want := glossary.GetString("expression"), "pala"; got != want {
		t.Fatalf("expected glossary expression %q, got %q", want, got)
	}
	if got, want := glossary.GetString("definition"), "Synthetic glossary definition for pala."; got != want {
		t.Fatalf("expected glossary definition %q, got %q", want, got)
	}

	guestbook, err := app.FindRecordById("guestbook", "005f6da0aa860f1")
	if err != nil {
		t.Fatalf("find guestbook entry: %v", err)
	}
	if got, want := guestbook.GetString("email"), "synthetic-guest-one@example.test"; got != want {
		t.Fatalf("expected guestbook email %q, got %q", want, got)
	}
	if !strings.HasPrefix(guestbook.GetString("created"), "2020-01-01") {
		t.Fatalf("expected preserved guestbook timestamp, got %q", guestbook.GetString("created"))
	}

	artwork, err := app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	if got, want := artwork.GetString("comment"), "<p>1911 · Synthetic Gallery, Test City · 125 x 225 cm</p>"; got != want {
		t.Fatalf("expected artwork comment %q, got %q", want, got)
	}
	if artwork.GetString("image") == "" {
		t.Fatal("expected artwork image")
	}

	song, err := app.FindRecordById("music_song", "72d6bb922f76aea")
	if err != nil {
		t.Fatalf("find music track: %v", err)
	}
	if song.GetString("source") == "" {
		t.Fatal("expected music source")
	}
	composerIDs := song.GetStringSlice("composer")
	if len(composerIDs) != 1 {
		t.Fatal("expected music composer relation")
	}
	composer, err := app.FindRecordById("music_composer", composerIDs[0])
	if err != nil {
		t.Fatalf("find music composer: %v", err)
	}
	if got, want := composer.GetString("name"), "Frédéric Chopin"; got != want {
		t.Fatalf("expected composer %q, got %q", want, got)
	}
	if got, want := composer.GetString("century"), "19"; got != want {
		t.Fatalf("expected composer century %q, got %q", want, got)
	}
	if composer.GetString("language") != "" {
		t.Fatal("expected no language inferred from music origin")
	}

	welcome, err := app.FindRecordById("strings", "0caeb5cf345e5e2")
	if err != nil {
		t.Fatalf("find welcome string: %v", err)
	}
	if !strings.Contains(welcome.GetString("content"), "Web Gallery of Art") {
		t.Fatal("expected welcome content")
	}

	privacyPage, err := app.FindRecordById("static_pages", "008bf0df3b30afe")
	if err != nil {
		t.Fatalf("find privacy page: %v", err)
	}
	if got, want := privacyPage.GetString("slug"), "privacy-policy"; got != want {
		t.Fatalf("expected privacy page slug %q, got %q", want, got)
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("open filesystem: %v", err)
	}
	defer func() {
		_ = fsys.Close()
	}()

	for _, record := range []*core.Record{artwork, song} {
		fieldName := "image"
		if record.Collection().Id == "music_song" {
			fieldName = "source"
		}
		filePath := record.BaseFilesPath() + "/" + record.GetString(fieldName)
		if exists, err := fsys.Exists(filePath); err != nil || !exists {
			t.Fatalf("expected source file %q, exists=%t err=%v", filePath, exists, err)
		}
	}

	if _, err := app.FindRecordById("strings", "syntheticseedv1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected no synthetic seed marker, got %v", err)
	}

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}
	assertSyntheticCollectionCounts(t, app, map[string]int{"artists": 10, "artworks": 27, "music_song": 3})
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
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	artists.Fields.Add(&core.TextField{Name: "source_path"})
	if err := app.Save(artists); err != nil {
		t.Fatalf("add legacy artist field: %v", err)
	}
	createLegacyCollection(t, app, "Professions", "professions", nil)
	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("source_path", "artists/synthetic-artist-02.json")
	if err := app.Save(artist); err != nil {
		t.Fatalf("save legacy artist field value: %v", err)
	}
	professions, err := app.FindCollectionByNameOrId("professions")
	if err != nil {
		t.Fatalf("find professions collection: %v", err)
	}
	profession := core.NewRecord(professions)
	if err := app.Save(profession); err != nil {
		t.Fatalf("save legacy profession: %v", err)
	}

	if err := seedSyntheticData(app); err != nil {
		t.Fatalf("skip populated target: %v", err)
	}
	assertSyntheticCollectionCounts(t, app, map[string]int{"artists": 10, "artworks": 27, "music_song": 3})
	artists, err = app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection after skip: %v", err)
	}
	if artists.Fields.GetByName("source_path") == nil {
		t.Fatal("expected legacy artist field to be retained")
	}
	artist, err = app.FindRecordById("artists", artist.Id)
	if err != nil {
		t.Fatalf("find artist after skip: %v", err)
	}
	if got, want := artist.GetString("source_path"), "artists/synthetic-artist-02.json"; got != want {
		t.Fatalf("expected retained legacy artist field value %q, got %q", want, got)
	}
	if _, err := app.FindRecordById("professions", profession.Id); err != nil {
		t.Fatalf("expected retained legacy profession: %v", err)
	}
}

func TestSyntheticSeedMigrationUpgradesLegacyDatabase(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := runMigrationsBeforeSyntheticSeed(app); err != nil {
		t.Fatalf("run legacy migrations: %v", err)
	}

	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find legacy artists collection: %v", err)
	}
	artists.Fields.Add(&core.TextField{Name: "source_path"})
	if err := app.Save(artists); err != nil {
		t.Fatalf("add legacy artist field: %v", err)
	}
	createLegacyCollection(t, app, "Professions", "professions", nil)

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

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("upgrade legacy database: %v", err)
	}

	if _, err := app.FindRecordById("strings", custom.Id); err != nil {
		t.Fatalf("find existing content: %v", err)
	}
	if _, err := app.FindRecordById("artists", "2236bdd57f7492e"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected no synthetic artist in populated database, got %v", err)
	}
	artists, err = app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection after upgrade: %v", err)
	}
	if artists.Fields.GetByName("source_path") == nil {
		t.Fatal("expected legacy artist field to be retained")
	}
	if _, err := app.FindCollectionByNameOrId("professions"); err != nil {
		t.Fatalf("expected professions collection to be retained: %v", err)
	}

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("rerun upgraded migrations: %v", err)
	}
}

func TestSyntheticSeedMigrationRemovesLegacySourceSchema(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	createLegacyCollection(t, app, "Artists", "artists", []string{"source_path", "professions"})
	createLegacyCollection(t, app, "Artworks", "artworks", []string{"date_text", "source_url"})
	createLegacyCollection(t, app, "Glossary", "glossary", []string{"anchor", "source_page"})
	createLegacyCollection(t, app, "Music_song", "music_song", []string{"track_order", "playback_url"})
	for _, collection := range []struct {
		name string
		id   string
	}{
		{name: "Professions", id: "professions"},
		{name: "Biographies", id: "biographies"},
		{name: "Biography_links", id: "biography_links"},
		{name: "Source_attributions", id: "source_attributions"},
	} {
		createLegacyCollection(t, app, collection.name, collection.id, nil)
	}

	if err := removeLegacySyntheticSourceSchema(app); err != nil {
		t.Fatalf("remove legacy synthetic source schema: %v", err)
	}

	for _, collectionName := range []string{"professions", "biographies", "biography_links", "source_attributions"} {
		_, err := app.FindCollectionByNameOrId(collectionName)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no %s collection, got %v", collectionName, err)
		}
	}

	for _, item := range []struct {
		collection string
		field      string
	}{
		{collection: "artists", field: "source_path"},
		{collection: "artists", field: "professions"},
		{collection: "artworks", field: "date_text"},
		{collection: "artworks", field: "source_url"},
		{collection: "glossary", field: "anchor"},
		{collection: "music_song", field: "track_order"},
	} {
		collection, err := app.FindCollectionByNameOrId(item.collection)
		if err != nil {
			t.Fatalf("find %s collection: %v", item.collection, err)
		}
		if collection.Fields.GetByName(item.field) != nil {
			t.Fatalf("expected no %s field on %s", item.field, item.collection)
		}
	}
}

func createLegacyCollection(t *testing.T, app core.App, name string, id string, fields []string) {
	t.Helper()

	collection := core.NewBaseCollection(name)
	collection.Id = id
	collection.MarkAsNew()
	for _, fieldName := range fields {
		collection.Fields.Add(&core.TextField{Name: fieldName})
	}
	if err := app.Save(collection); err != nil {
		t.Fatalf("create %s collection: %v", id, err)
	}
}

func runMigrationsBeforeSyntheticSeed(app core.App) error {
	list := core.MigrationsList{}
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1784808383_seed_synthetic_data.go" {
			continue
		}
		list.Add(&core.Migration{
			File: migration.File,
			Up:   migration.Up,
			Down: migration.Down,
		})
	}

	_, err := core.NewMigrationsRunner(app, list).Up()

	return err
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
