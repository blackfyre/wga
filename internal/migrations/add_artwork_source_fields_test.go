package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestArtworkSourceFieldsMigrationAddsTypedFields(t *testing.T) {
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
	for _, check := range []struct {
		name string
		typ  string
	}{
		{"source_url", core.FieldTypeText},
		{"source_path", core.FieldTypeText},
		{"source_comment", core.FieldTypeText},
		{"colour_palette", core.FieldTypeJSON},
		{"colour_signature", core.FieldTypeJSON},
		{"colour_profile_version", core.FieldTypeText},
		{"colour_image_hash", core.FieldTypeText},
	} {
		field := artworks.Fields.GetByName(check.name)
		if field == nil {
			t.Fatalf("artworks missing field %q", check.name)
		}
		if field.Type() != check.typ {
			t.Fatalf("artworks field %q type = %q, want %q", check.name, field.Type(), check.typ)
		}
	}

	joinedIndexes := strings.Join(artworks.Indexes, "\n")
	if !strings.Contains(joinedIndexes, artworkColourImageHashIndex) {
		t.Fatalf("missing index %q in %q", artworkColourImageHashIndex, joinedIndexes)
	}
}

func TestArtworkSourceFieldsMigrationIsIdempotent(t *testing.T) {
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
	if err := addArtworkSourceFields(app); err != nil {
		t.Fatalf("add artwork source fields again: %v", err)
	}
}

func TestRemoveArtworkSourceFieldsKeepsFieldsAndDropsIndex(t *testing.T) {
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
	if err := removeArtworkSourceFields(app); err != nil {
		t.Fatalf("remove artwork source fields: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find rolled-back artworks: %v", err)
	}
	for _, field := range []string{
		"source_url", "source_path", "source_comment",
		"colour_palette", "colour_signature", "colour_profile_version", "colour_image_hash",
	} {
		if artworks.Fields.GetByName(field) == nil {
			t.Fatalf("authoritative field %q was removed by rollback", field)
		}
	}

	joinedIndexes := strings.Join(artworks.Indexes, "\n")
	if strings.Contains(joinedIndexes, artworkColourImageHashIndex) {
		t.Fatalf("rollback retained feature index %q", artworkColourImageHashIndex)
	}
}
