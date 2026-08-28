package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestArtworkFileByteSizeMigrationAddsFieldAndRaisesMax(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtworkSourceFields(app); err != nil {
		t.Fatalf("add artwork source fields: %v", err)
	}
	if err := addArtworkFileByteSize(app); err != nil {
		t.Fatalf("add artwork file byte size: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}

	size, ok := artworks.Fields.GetByName("image_size_bytes").(*core.NumberField)
	if !ok {
		t.Fatalf("artworks missing numeric image_size_bytes field")
	}
	if size.Required {
		t.Fatal("image_size_bytes should be optional")
	}

	comment, ok := artworks.Fields.GetByName("source_comment").(*core.TextField)
	if !ok {
		t.Fatalf("artworks missing source_comment text field")
	}
	if comment.Max != artworkSourceCommentMaxChars {
		t.Fatalf("source_comment max = %d, want %d", comment.Max, artworkSourceCommentMaxChars)
	}
}

func TestArtworkFileByteSizeMigrationAddsSourceCommentWhenAbsent(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtworkFileByteSize(app); err != nil {
		t.Fatalf("add artwork file byte size: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	comment, ok := artworks.Fields.GetByName("source_comment").(*core.TextField)
	if !ok {
		t.Fatalf("artworks missing source_comment text field")
	}
	if comment.Max != artworkSourceCommentMaxChars {
		t.Fatalf("source_comment max = %d, want %d", comment.Max, artworkSourceCommentMaxChars)
	}
	if artworks.Fields.GetByName("image_size_bytes") == nil {
		t.Fatal("artworks missing image_size_bytes field")
	}
}

func TestArtworkFileByteSizeMigrationIsIdempotent(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtworkSourceFields(app); err != nil {
		t.Fatalf("add artwork source fields: %v", err)
	}
	if err := addArtworkFileByteSize(app); err != nil {
		t.Fatalf("add artwork file byte size: %v", err)
	}
	if err := addArtworkFileByteSize(app); err != nil {
		t.Fatalf("add artwork file byte size again: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	comment, ok := artworks.Fields.GetByName("source_comment").(*core.TextField)
	if !ok {
		t.Fatalf("artworks missing source_comment text field")
	}
	if comment.Max != artworkSourceCommentMaxChars {
		t.Fatalf("source_comment max = %d after repeat, want %d", comment.Max, artworkSourceCommentMaxChars)
	}
}

func TestArtworkFileByteSizeMigrationPreservesExistingComment(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addArtworkSourceFields(app); err != nil {
		t.Fatalf("add artwork source fields: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(artworks)
	record.Set("title", "Existing artwork")
	record.Set("source_comment", "Pre-existing source commentary")
	if err := app.Save(record); err != nil {
		t.Fatalf("save existing artwork: %v", err)
	}

	if err := addArtworkFileByteSize(app); err != nil {
		t.Fatalf("add artwork file byte size: %v", err)
	}

	saved, err := app.FindRecordById("artworks", record.Id)
	if err != nil {
		t.Fatalf("find preserved artwork: %v", err)
	}
	if got := saved.GetString("source_comment"); got != "Pre-existing source commentary" {
		t.Fatalf("source_comment = %q, want preserved value", got)
	}
	if got := saved.GetInt("image_size_bytes"); got != 0 {
		t.Fatalf("image_size_bytes = %d, want unset (0) for pre-migration record", got)
	}
}

func TestArtworkFileByteSizeMigrationRegistrationAndOrder(t *testing.T) {
	const (
		name            = "1784808002_add_artwork_file_byte_size.go"
		bootstrapSchema = "1784808001_current_schema.go"
		bootstrapSeed   = "1784808002_seed_synthetic_data.go"
	)

	byteIndex := -1
	schemaIndex := -1
	seedIndex := -1
	count := 0

	for i, migration := range core.AppMigrations.Items() {
		switch migration.File {
		case name:
			byteIndex = i
			count++
		case bootstrapSchema:
			schemaIndex = i
		case bootstrapSeed:
			seedIndex = i
		}
	}

	if count != 1 {
		t.Fatalf("migration %q registered %d times, want exactly 1", name, count)
	}
	if schemaIndex == -1 {
		t.Fatalf("bootstrap schema migration %q is not registered", bootstrapSchema)
	}
	if seedIndex == -1 {
		t.Fatalf("bootstrap seed migration %q is not registered", bootstrapSeed)
	}
	if byteIndex <= schemaIndex {
		t.Fatalf("migration %q must sort after bootstrap schema %q", name, bootstrapSchema)
	}
	if byteIndex >= seedIndex {
		t.Fatalf("migration %q must sort before bootstrap seed %q", name, bootstrapSeed)
	}
}

func TestArtworkFileByteSizeMigrationAppliesOnCleanSchemaSetup(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations on clean schema: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	if _, ok := artworks.Fields.GetByName("image_size_bytes").(*core.NumberField); !ok {
		t.Fatal("clean schema setup missing numeric image_size_bytes field")
	}
}

func TestArtworkFileByteSizeMigrationUpgradesExistingBootstrapHistory(t *testing.T) {
	const (
		byteSizeFile = "1784808002_add_artwork_file_byte_size.go"
		seedFile     = "1784808002_seed_synthetic_data.go"
	)

	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	// Reconstruct the migration history that predates the byte-size migration:
	// every registered app migration except the new one. seed_synthetic_data is
	// applied separately below because its importer now requires the
	// image_size_bytes field, which cannot exist yet at this point in history.
	prior := core.MigrationsList{}
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == byteSizeFile || migration.File == seedFile {
			continue
		}
		prior.Register(migration.Up, migration.Down, migration.File)
	}
	if _, err := core.NewMigrationsRunner(app, prior).Up(); err != nil {
		t.Fatalf("apply prior bootstrap migrations: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	if got := artworks.Fields.GetByName("image_size_bytes"); got != nil {
		t.Fatal("prior bootstrap already has image_size_bytes field")
	}
	sourceComment, ok := artworks.Fields.GetByName("source_comment").(*core.TextField)
	if !ok {
		t.Fatal("prior bootstrap missing source_comment field")
	}
	if sourceComment.Max != 0 {
		t.Fatalf("prior bootstrap source_comment max = %d, want unset (0)", sourceComment.Max)
	}

	record := core.NewRecord(artworks)
	record.Set("title", "Pre-existing artwork")
	record.Set("source_comment", "Pre-existing source commentary")
	if err := app.Save(record); err != nil {
		t.Fatalf("save pre-existing artwork: %v", err)
	}

	// Record the historical seed step as applied. The application database is
	// already populated, so the seed migration correctly no-ops and only its
	// history entry is written.
	seedOnly := core.MigrationsList{}
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == seedFile {
			seedOnly.Register(migration.Up, migration.Down, migration.File)
		}
	}
	seeded, err := core.NewMigrationsRunner(app, seedOnly).Up()
	if err != nil {
		t.Fatalf("record prior seed history: %v", err)
	}
	if len(seeded) != 1 || seeded[0] != seedFile {
		t.Fatalf("seed history applied = %v, want exactly [%s]", seeded, seedFile)
	}

	// Invoke PocketBase's actual migration runner (the exact steps behind
	// RunAllMigrations) and observe which migrations it applies.
	full := core.MigrationsList{}
	full.Copy(core.SystemMigrations)
	full.Copy(core.AppMigrations)
	applied, err := core.NewMigrationsRunner(app, full).Up()
	if err != nil {
		t.Fatalf("apply byte-size migration via runner: %v", err)
	}
	if len(applied) != 1 || applied[0] != byteSizeFile {
		t.Fatalf("applied migrations = %v, want exactly [%s]", applied, byteSizeFile)
	}

	artworks, err = app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks after migration: %v", err)
	}
	if _, ok := artworks.Fields.GetByName("image_size_bytes").(*core.NumberField); !ok {
		t.Fatal("artworks missing numeric image_size_bytes field after migration")
	}
	sourceComment, ok = artworks.Fields.GetByName("source_comment").(*core.TextField)
	if !ok {
		t.Fatal("artworks missing source_comment field after migration")
	}
	if sourceComment.Max != artworkSourceCommentMaxChars {
		t.Fatalf("source_comment max = %d, want %d", sourceComment.Max, artworkSourceCommentMaxChars)
	}

	saved, err := app.FindRecordById("artworks", record.Id)
	if err != nil {
		t.Fatalf("find preserved artwork: %v", err)
	}
	if got := saved.GetString("source_comment"); got != "Pre-existing source commentary" {
		t.Fatalf("source_comment = %q, want preserved value", got)
	}
	if got := saved.GetInt("image_size_bytes"); got != 0 {
		t.Fatalf("image_size_bytes = %d, want unset (0) for pre-migration record", got)
	}

	rerun, err := core.NewMigrationsRunner(app, full).Up()
	if err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}
	if len(rerun) != 0 {
		t.Fatalf("rerun applied migrations = %v, want none", rerun)
	}
}
