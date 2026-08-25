package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestAddArtworkDateSpansAddsFieldsAndIndex(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	if err := addArtworkDateSpans(app); err != nil {
		t.Fatalf("add artwork date spans: %v", err)
	}
	updated, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find updated artworks collection: %v", err)
	}
	for _, want := range []struct {
		name  string
		check func(core.Field) bool
	}{
		{"date_start", func(f core.Field) bool { _, ok := f.(*core.NumberField); return ok }},
		{"date_end", func(f core.Field) bool { _, ok := f.(*core.NumberField); return ok }},
		{"is_circa", func(f core.Field) bool { _, ok := f.(*core.BoolField); return ok }},
		{"date_qualifier", func(f core.Field) bool { _, ok := f.(*core.TextField); return ok }},
		{"timeframe_text", func(f core.Field) bool { _, ok := f.(*core.TextField); return ok }},
	} {
		got := updated.Fields.GetByName(want.name)
		if got == nil {
			t.Fatalf("missing field %q", want.name)
		}
		if !want.check(got) {
			t.Fatalf("field %q has unexpected type %T", want.name, got)
		}
	}

	joinedIndexes := strings.Join(updated.Indexes, "\n")
	if !strings.Contains(joinedIndexes, artworkDateRangeIndex) {
		t.Fatalf("missing index %q in %q", artworkDateRangeIndex, joinedIndexes)
	}

	if err := addArtworkDateSpans(app); err != nil {
		t.Fatalf("repeat artwork date spans migration: %v", err)
	}
}

func TestRemoveArtworkDateSpansKeepsFieldsAndDropsIndex(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	if err := addArtworkDateSpans(app); err != nil {
		t.Fatalf("add artwork date spans: %v", err)
	}
	if err := removeArtworkDateSpans(app); err != nil {
		t.Fatalf("remove artwork date spans: %v", err)
	}
	rolledBack, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find rolled-back artworks collection: %v", err)
	}
	for _, field := range []string{"date_start", "date_end", "is_circa", "date_qualifier", "timeframe_text"} {
		if rolledBack.Fields.GetByName(field) == nil {
			t.Fatalf("authoritative field %q was removed by rollback", field)
		}
	}
	joinedIndexes := strings.Join(rolledBack.Indexes, "\n")
	if strings.Contains(joinedIndexes, artworkDateRangeIndex) {
		t.Fatalf("rollback retained feature index %q", artworkDateRangeIndex)
	}
}
