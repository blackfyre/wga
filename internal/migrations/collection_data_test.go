package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestCollectionDataMigrationCreatesLocationsAndArtworkFields(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	// The relation fields reference the artworks and art_periods collections, so
	// those must exist before the collection-data fields are added.
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}

	if err := addCollectionData(app); err != nil {
		t.Fatalf("add collection data: %v", err)
	}

	locations, err := app.FindCollectionByNameOrId("locations")
	if err != nil {
		t.Fatalf("find locations: %v", err)
	}
	if locations.Name != "Locations" || locations.Type != core.CollectionTypeBase {
		t.Fatalf("locations collection = %q/%q, want base Locations", locations.Name, locations.Type)
	}
	for _, check := range []struct {
		name string
		typ  string
	}{
		{"name", core.FieldTypeText},
		{"city", core.FieldTypeText},
		{"country", core.FieldTypeText},
		{"museum", core.FieldTypeBool},
		{"is_public", core.FieldTypeBool},
	} {
		field := locations.Fields.GetByName(check.name)
		if field == nil {
			t.Fatalf("locations missing field %q", check.name)
		}
		if field.Type() != check.typ {
			t.Fatalf("locations field %q type = %q, want %q", check.name, field.Type(), check.typ)
		}
	}
	for name, rule := range map[string]*string{
		"listRule":   locations.ListRule,
		"viewRule":   locations.ViewRule,
		"createRule": locations.CreateRule,
		"updateRule": locations.UpdateRule,
		"deleteRule": locations.DeleteRule,
	} {
		if rule != nil {
			t.Fatalf("locations %s = %q, want nil (operationally private)", name, *rule)
		}
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	for _, check := range []struct {
		name string
		typ  string
	}{
		{"source_row", core.FieldTypeNumber},
		{"date_start", core.FieldTypeNumber},
		{"date_end", core.FieldTypeNumber},
		{"is_circa", core.FieldTypeBool},
		{"date_qualifier", core.FieldTypeText},
		{"timeframe_text", core.FieldTypeText},
		{"current_location_id", core.FieldTypeRelation},
		{"art_period_id", core.FieldTypeRelation},
	} {
		field := artworks.Fields.GetByName(check.name)
		if field == nil {
			t.Fatalf("artworks missing field %q", check.name)
		}
		if field.Type() != check.typ {
			t.Fatalf("artworks field %q type = %q, want %q", check.name, field.Type(), check.typ)
		}
	}

	locationRelation, ok := artworks.Fields.GetByName("current_location_id").(*core.RelationField)
	if !ok {
		t.Fatal("current_location_id is not a relation field")
	}
	if locationRelation.CollectionId != "locations" || locationRelation.MinSelect != 0 || locationRelation.MaxSelect != 1 || locationRelation.Required {
		t.Fatalf("current_location_id relation = %+v, want optional single locations relation", locationRelation)
	}

	periodRelation, ok := artworks.Fields.GetByName("art_period_id").(*core.RelationField)
	if !ok {
		t.Fatal("art_period_id is not a relation field")
	}
	if periodRelation.CollectionId != "art_periods" || periodRelation.MinSelect != 0 || periodRelation.MaxSelect != 1 || periodRelation.Required {
		t.Fatalf("art_period_id relation = %+v, want optional single art_periods relation", periodRelation)
	}

	joinedIndexes := strings.Join(artworks.Indexes, "\n")
	for _, index := range []string{collectionDataIndexSourceRow, collectionDataIndexCurrentLocation, collectionDataIndexArtPeriod} {
		if !strings.Contains(joinedIndexes, index) {
			t.Fatalf("missing index %q in %q", index, joinedIndexes)
		}
	}
}

func TestCollectionDataMigrationIsIdempotent(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addCollectionData(app); err != nil {
		t.Fatalf("add collection data: %v", err)
	}
	if err := addCollectionData(app); err != nil {
		t.Fatalf("add collection data again: %v", err)
	}
}

func TestRemoveCollectionDataKeepsFieldsAndDropsIndexes(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addCollectionData(app); err != nil {
		t.Fatalf("add collection data: %v", err)
	}
	if err := removeCollectionData(app); err != nil {
		t.Fatalf("remove collection data: %v", err)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find rolled-back artworks: %v", err)
	}
	for _, field := range []string{
		"source_row", "date_start", "date_end", "is_circa", "date_qualifier", "timeframe_text",
		"current_location_id", "art_period_id",
	} {
		if artworks.Fields.GetByName(field) == nil {
			t.Fatalf("authoritative field %q was removed by rollback", field)
		}
	}
	if _, err := app.FindCollectionByNameOrId("locations"); err != nil {
		t.Fatalf("locations taxonomy was removed by rollback: %v", err)
	}

	joinedIndexes := strings.Join(artworks.Indexes, "\n")
	for _, index := range []string{collectionDataIndexSourceRow, collectionDataIndexCurrentLocation, collectionDataIndexArtPeriod} {
		if strings.Contains(joinedIndexes, index) {
			t.Fatalf("rollback retained feature index %q", index)
		}
	}
}
