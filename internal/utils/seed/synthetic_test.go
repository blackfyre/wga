package seed

import (
	"bytes"
	"io"
	iofs "io/fs"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/migrations"
	"github.com/pocketbase/pocketbase/core"
)

var configureSeedMigrationsOnce sync.Once
var configureSeedMigrationsErr error

func TestSeedDatabaseAndStorage(t *testing.T) {
	configureSeedMigrations(t)

	app := newSeedTestApp(t)
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	options := SourceOptions{Environment: config.EnvironmentDevelopment}
	databaseOptions := options
	databaseOptions.StoragePath = filepath.Join(t.TempDir(), "unavailable-storage")
	databaseOptions.ReplaceMinimal = true
	if err := SeedDatabase(app, databaseOptions); err != nil {
		t.Fatalf("seed database: %v", err)
	}

	assertCollectionCounts(t, app, map[string]int{
		"schools":             1,
		"art_forms":           1,
		"art_types":           1,
		"professions":         18,
		"artists":             10,
		"biographies":         10,
		"biography_links":     10,
		"artworks":            27,
		"glossary":            5,
		"guestbook":           2,
		"music_composer":      2,
		"music_song":          3,
		"source_attributions": 2,
		"strings":             8,
		"static_pages":        1,
	})

	marker, err := app.FindRecordById("strings", syntheticSeedMarkerID)
	if err != nil {
		t.Fatalf("find seed marker: %v", err)
	}
	if marker.GetString("name") != syntheticSeedMarkerName || marker.GetString("content") != syntheticSeedMarkerContent {
		t.Fatal("expected synthetic seed marker")
	}

	artwork, err := app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	if artwork.GetString("image") != "" {
		t.Fatalf("expected database-only seed to leave image empty, got %q", artwork.GetString("image"))
	}
	if artwork.GetString("source_url") != "https://example.test/artworks/08-03" {
		t.Fatalf("unexpected source URL %q", artwork.GetString("source_url"))
	}

	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find seeded artist: %v", err)
	}
	artist.Set("name", "Edited Synthetic Artist")
	if err := app.Save(artist); err != nil {
		t.Fatalf("edit seeded artist: %v", err)
	}
	if err := SeedDatabase(app, databaseOptions); err != nil {
		t.Fatalf("repeat database seed: %v", err)
	}
	artist, err = app.FindRecordById("artists", artist.Id)
	if err != nil {
		t.Fatalf("find edited seeded artist: %v", err)
	}
	if got, want := artist.GetString("name"), "Edited Synthetic Artist"; got != want {
		t.Fatalf("expected edited artist name %q, got %q", want, got)
	}

	if err := SeedStorage(app, options); err != nil {
		t.Fatalf("seed storage: %v", err)
	}

	artwork, err = app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find seeded artwork: %v", err)
	}
	if artwork.GetString("image") == "" {
		t.Fatal("expected storage seed to attach artwork image")
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer func() {
		_ = fsys.Close()
	}()

	artworkPath := artwork.BaseFilesPath() + "/" + artwork.GetString("image")
	if exists, err := fsys.Exists(artworkPath); err != nil || !exists {
		t.Fatalf("expected artwork storage file %q, exists=%t err=%v", artworkPath, exists, err)
	}

	song, err := app.FindRecordById("music_song", "73c3f0167cb7348")
	if err != nil {
		t.Fatalf("find music track: %v", err)
	}
	songPath := song.BaseFilesPath() + "/" + song.GetString("source")
	if exists, err := fsys.Exists(songPath); err != nil || !exists {
		t.Fatalf("expected music storage file %q, exists=%t err=%v", songPath, exists, err)
	}

	if err := fsys.Upload([]byte("incorrect"), artworkPath); err != nil {
		t.Fatalf("overwrite artwork storage file: %v", err)
	}
	if err := SeedStorage(app, options); err != nil {
		t.Fatalf("repair storage: %v", err)
	}

	reader, err := fsys.GetReader(artworkPath)
	if err != nil {
		t.Fatalf("read repaired artwork storage file: %v", err)
	}
	actual, err := io.ReadAll(reader)
	closeErr := reader.Close()
	if err != nil {
		t.Fatalf("read repaired artwork storage content: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close repaired artwork storage reader: %v", closeErr)
	}

	paths, err := resolveSourcePaths(options)
	if err != nil {
		t.Fatalf("resolve embedded storage source: %v", err)
	}
	defer func() {
		_ = paths.Close()
	}()
	expected, err := iofs.ReadFile(paths.storage, sourceFilePath("Artworks", artwork.Id, artwork.GetString("image")))
	if err != nil {
		t.Fatalf("read source artwork storage file: %v", err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("expected storage seed to replace an existing incorrect object")
	}
}

func TestResolveSourcePathsRequiresProductionOverride(t *testing.T) {
	_, err := resolveSourcePaths(SourceOptions{Environment: config.EnvironmentProduction})
	if err == nil {
		t.Fatal("expected production source path override error")
	}
}

func TestResolveSourcePathsUsesDevelopmentBundle(t *testing.T) {
	t.Chdir(t.TempDir())

	paths, err := resolveSourcePaths(SourceOptions{Environment: config.EnvironmentDevelopment})
	if err != nil {
		t.Fatalf("resolve development source: %v", err)
	}
	defer func() {
		_ = paths.Close()
	}()

	data, err := loadSourceData(paths)
	if err != nil {
		t.Fatalf("read embedded development source: %v", err)
	}
	if got, want := len(data.artworks), 27; got != want {
		t.Fatalf("expected %d embedded artworks, got %d", want, got)
	}
	if _, err := iofs.Stat(paths.storage, "Artworks/07561d2efd0a6db/3a29b540e6908ad8.jpg"); err != nil {
		t.Fatalf("find embedded artwork asset: %v", err)
	}
}

func TestResolveSourcePathsUsesConfiguredSQLite(t *testing.T) {
	options := syntheticSourceOptions(t)
	paths, err := resolveSourcePaths(options)
	if err != nil {
		t.Fatalf("resolve configured source: %v", err)
	}
	defer func() {
		_ = paths.Close()
	}()

	wantSQLitePath, err := filepath.Abs(options.SQLitePath)
	if err != nil {
		t.Fatalf("resolve expected SQLite path: %v", err)
	}
	if got := paths.sqlitePath; got != wantSQLitePath {
		t.Fatalf("expected SQLite path %q, got %q", wantSQLitePath, got)
	}
	if _, err := iofs.Stat(paths.storage, "Artworks/07561d2efd0a6db/3a29b540e6908ad8.jpg"); err != nil {
		t.Fatalf("find configured artwork asset: %v", err)
	}
}

func TestSeedDatabasePreservesNonminimalData(t *testing.T) {
	configureSeedMigrations(t)

	app := newSeedTestApp(t)
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	strings, err := app.FindCollectionByNameOrId("strings")
	if err != nil {
		t.Fatalf("find strings collection: %v", err)
	}
	custom := core.NewRecord(strings)
	custom.Set("name", "custom")
	custom.Set("content", "Custom content")
	if err := app.Save(custom); err != nil {
		t.Fatalf("create custom string: %v", err)
	}

	if err := SeedDatabase(app, syntheticSourceOptions(t)); err == nil {
		t.Fatal("expected data seed to reject a nonminimal database")
	}

	if _, err := app.FindRecordById("strings", custom.Id); err != nil {
		t.Fatalf("expected custom string to remain: %v", err)
	}
	if _, err := app.FindFirstRecordByData("artists", "slug", "mara-example"); err != nil {
		t.Fatalf("expected minimal artist to remain: %v", err)
	}
}

func TestSeedDatabasePreservesEditedStarterContentWithoutReplacementFlag(t *testing.T) {
	configureSeedMigrations(t)

	app := newSeedTestApp(t)
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	welcome, err := app.FindFirstRecordByData("strings", "name", "welcome")
	if err != nil {
		t.Fatalf("find welcome string: %v", err)
	}
	welcome.Set("content", "<p>Edited welcome content.</p>")
	if err := app.Save(welcome); err != nil {
		t.Fatalf("edit welcome string: %v", err)
	}

	if err := SeedDatabase(app, syntheticSourceOptions(t)); err == nil {
		t.Fatal("expected minimal replacement flag error")
	}

	welcome, err = app.FindRecordById("strings", welcome.Id)
	if err != nil {
		t.Fatalf("find edited welcome string: %v", err)
	}
	if got, want := welcome.GetString("content"), "<p>Edited welcome content.</p>"; got != want {
		t.Fatalf("expected welcome content %q, got %q", want, got)
	}
}

func TestSeedDatabaseRejectsEditedStarterContentWithReplacementFlag(t *testing.T) {
	configureSeedMigrations(t)

	app := newSeedTestApp(t)
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	welcome, err := app.FindFirstRecordByData("strings", "name", "welcome")
	if err != nil {
		t.Fatalf("find welcome string: %v", err)
	}
	welcome.Set("content", "<p>Edited welcome content.</p>")
	if err := app.Save(welcome); err != nil {
		t.Fatalf("edit welcome string: %v", err)
	}

	options := syntheticSourceOptions(t)
	options.ReplaceMinimal = true
	if err := SeedDatabase(app, options); err == nil {
		t.Fatal("expected edited starter content replacement error")
	}

	welcome, err = app.FindRecordById("strings", welcome.Id)
	if err != nil {
		t.Fatalf("find edited welcome string: %v", err)
	}
	if got, want := welcome.GetString("content"), "<p>Edited welcome content.</p>"; got != want {
		t.Fatalf("expected welcome content %q, got %q", want, got)
	}
}

func TestSeedDatabaseRejectsEditedExtensionFieldWithReplacementFlag(t *testing.T) {
	configureSeedMigrations(t)

	app := newSeedTestApp(t)
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	artist, err := app.FindFirstRecordByData("artists", "slug", "mara-example")
	if err != nil {
		t.Fatalf("find minimal artist: %v", err)
	}
	artist.Set("source_path", "edited-source-path")
	if err := app.Save(artist); err != nil {
		t.Fatalf("edit source path: %v", err)
	}

	options := syntheticSourceOptions(t)
	options.ReplaceMinimal = true
	if err := SeedDatabase(app, options); err == nil {
		t.Fatal("expected edited extension field replacement error")
	}

	artist, err = app.FindRecordById("artists", artist.Id)
	if err != nil {
		t.Fatalf("find edited artist: %v", err)
	}
	if got, want := artist.GetString("source_path"), "edited-source-path"; got != want {
		t.Fatalf("expected source path %q, got %q", want, got)
	}
}

func TestSeedDatabaseRejectsEditedLegacyRelationWithReplacementFlag(t *testing.T) {
	configureSeedMigrations(t)

	app := newSeedTestApp(t)
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	artist, err := app.FindFirstRecordByData("artists", "slug", "mara-example")
	if err != nil {
		t.Fatalf("find minimal artist: %v", err)
	}
	artist.Set("also_known_as", []string{artist.Id})
	if err := app.Save(artist); err != nil {
		t.Fatalf("edit artist relation: %v", err)
	}

	options := syntheticSourceOptions(t)
	options.ReplaceMinimal = true
	if err := SeedDatabase(app, options); err == nil {
		t.Fatal("expected edited legacy relation replacement error")
	}

	artist, err = app.FindRecordById("artists", artist.Id)
	if err != nil {
		t.Fatalf("find edited artist: %v", err)
	}
	if !hasOnlyID(artist.GetStringSlice("also_known_as"), artist.Id) {
		t.Fatal("expected edited legacy relation to remain")
	}
}

func configureSeedMigrations(t *testing.T) {
	t.Helper()

	configureSeedMigrationsOnce.Do(func() {
		configuration := config.LoadFrom(func(key string) string {
			return map[string]string{
				"WGA_ENV":            "development",
				"WGA_PROTOCOL":       "https",
				"WGA_HOSTNAME":       "gallery.example",
				"WGA_SMTP_HOST":      "smtp.example",
				"WGA_SMTP_PORT":      "2525",
				"WGA_SENDER_NAME":    "WGA Test",
				"WGA_SENDER_ADDRESS": "sender@example.com",
			}[key]
		})
		configureSeedMigrationsErr = migrations.Configure(configuration.Migrations())
	})
	if configureSeedMigrationsErr != nil {
		t.Fatalf("configure migrations: %v", configureSeedMigrationsErr)
	}
}

func newSeedTestApp(t *testing.T) *core.BaseApp {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "test-encryption-key",
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap application: %v", err)
	}

	return app
}

func syntheticSourceOptions(t *testing.T) SourceOptions {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate synthetic source")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../resources/synthetic"))
	return SourceOptions{
		Environment: config.EnvironmentDevelopment,
		SQLitePath:  filepath.Join(root, "wga-test.sqlite"),
		StoragePath: filepath.Join(root, "storage"),
	}
}

func assertCollectionCounts(t *testing.T, app core.App, expected map[string]int) {
	t.Helper()

	for collection, want := range expected {
		records, err := app.FindRecordsByFilter(collection, "", "", 0, 0)
		if err != nil {
			t.Fatalf("find %s records: %v", collection, err)
		}
		if got := len(records); got != want {
			t.Fatalf("expected %d %s records, got %d", want, collection, got)
		}
	}
}
