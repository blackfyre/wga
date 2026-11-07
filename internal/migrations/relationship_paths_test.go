package migrations

import (
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func newRelationshipPathsTestApp(t *testing.T) *core.BaseApp {
	t.Helper()

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ClearBootstrap(); err != nil {
			t.Error(err)
		}
	})
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if err := addCollectionData(app); err != nil {
		t.Fatalf("add collection data: %v", err)
	}
	if err := addRelationshipPaths(app); err != nil {
		t.Fatalf("add relationship paths: %v", err)
	}

	return app
}

func TestRelationshipPathsMigrationDefinesCollectionsAndArtworkRelations(t *testing.T) {
	app := newRelationshipPathsTestApp(t)

	// A second run must be a no-op rather than duplicating fields or indexes.
	if err := addRelationshipPaths(app); err != nil {
		t.Fatalf("repeat relationship paths: %v", err)
	}

	for _, check := range []struct {
		collection string
		fields     map[string]string
	}{
		{"subjects", map[string]string{"name": core.FieldTypeText}},
		{"series", map[string]string{"name": core.FieldTypeText, "kind": core.FieldTypeSelect}},
		{"techniques", map[string]string{"name": core.FieldTypeText, "comparison_key": core.FieldTypeText}},
		{"location_aliases", map[string]string{"comparison_key": core.FieldTypeText, "location": core.FieldTypeRelation}},
	} {
		collection, err := app.FindCollectionByNameOrId(check.collection)
		if err != nil {
			t.Fatalf("find %s: %v", check.collection, err)
		}
		for name, typ := range check.fields {
			field := collection.Fields.GetByName(name)
			if field == nil || field.Type() != typ {
				t.Fatalf("%s field %q = %v, want %s", check.collection, name, field, typ)
			}
		}
		for rule, value := range map[string]*string{"list": collection.ListRule, "view": collection.ViewRule, "create": collection.CreateRule, "update": collection.UpdateRule, "delete": collection.DeleteRule} {
			if value != nil {
				t.Fatalf("%s %s rule = %q, want nil (operationally private)", check.collection, rule, *value)
			}
		}
	}

	series, err := app.FindCollectionByNameOrId("series")
	if err != nil {
		t.Fatal(err)
	}
	kind := series.Fields.GetByName("kind").(*core.SelectField)
	if !slices.Equal(kind.Values, []string{"series", "altarpiece"}) || !kind.Required {
		t.Fatalf("series kind = %v required=%v, want required series|altarpiece", kind.Values, kind.Required)
	}

	aliases, err := app.FindCollectionByNameOrId("location_aliases")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(aliases.Indexes, "\n"), "CREATE UNIQUE INDEX `"+relationshipPathsIndexAliasKey+"`") {
		t.Fatalf("location alias comparison keys must be unique: %v", aliases.Indexes)
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		name      string
		target    string
		maxSelect int
	}{
		{"primary_author", "artists", 1},
		{"co_authors", "artists", 10},
		{"subjects", "subjects", 0},
		{"series", "series", 0},
		{"original_locations", "locations", 0},
		{"techniques", "techniques", 0},
		{"current_location_id", "locations", 1},
		{"art_period_id", "art_periods", 1},
		{"school", "schools", 10},
	} {
		relation, ok := artworks.Fields.GetByName(check.name).(*core.RelationField)
		if !ok {
			t.Fatalf("artworks relation %q missing", check.name)
		}
		if relation.CollectionId != check.target || relation.MaxSelect != check.maxSelect || relation.Required {
			t.Fatalf("artworks relation %q = %s/max %d/required %v, want optional %s/max %d", check.name, relation.CollectionId, relation.MaxSelect, relation.Required, check.target, check.maxSelect)
		}
	}
	if field := artworks.Fields.GetByName("source_location"); field == nil || field.Type() != core.FieldTypeText {
		t.Fatalf("artworks source_location = %v, want text", field)
	}
	if artworks.Fields.GetByName("author") == nil {
		t.Fatal("legacy author relation must be retained until review is complete")
	}
	if count := strings.Count(strings.Join(artworks.Indexes, "\n"), "`"+relationshipPathsIndexPrimaryAuthor+"`"); count != 1 {
		t.Fatalf("primary author index count = %d, want 1", count)
	}
}

func TestLegacyAuthorMigrationCopiesSingleAndReportsMultipleAuthors(t *testing.T) {
	app := newRelationshipPathsTestApp(t)

	artistsCollection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatal(err)
	}
	newArtist := func(name string) *core.Record {
		record := core.NewRecord(artistsCollection)
		record.Set("name", name)
		record.Set("slug", strings.ToLower(name))
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatalf("save artist %s: %v", name, err)
		}
		return record
	}
	first := newArtist("First")
	second := newArtist("Second")

	artworksCollection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatal(err)
	}
	newArtwork := func(title string, authors []string, primary string) *core.Record {
		record := core.NewRecord(artworksCollection)
		record.Set("title", title)
		record.Set("author", authors)
		if primary != "" {
			record.Set("primary_author", primary)
		}
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatalf("save artwork %s: %v", title, err)
		}
		return record
	}
	single := newArtwork("Single", []string{first.Id}, "")
	multiple := newArtwork("Multiple", []string{first.Id, second.Id}, "")
	curated := newArtwork("Curated", []string{first.Id}, second.Id)
	authorless := newArtwork("Authorless", nil, "")

	report, err := migrateLegacyArtworkAuthors(app)
	if err != nil {
		t.Fatalf("migrate legacy authors: %v", err)
	}
	if report.Copied != 1 {
		t.Fatalf("copied = %d, want 1", report.Copied)
	}
	if !slices.Equal(report.ReviewIDs, []string{multiple.Id}) {
		t.Fatalf("review ids = %v, want [%s]", report.ReviewIDs, multiple.Id)
	}

	reload := func(record *core.Record) *core.Record {
		found, err := app.FindRecordById("artworks", record.Id)
		if err != nil {
			t.Fatal(err)
		}
		return found
	}
	if got := reload(single).GetString("primary_author"); got != first.Id {
		t.Fatalf("single primary_author = %q, want %q", got, first.Id)
	}
	gotMultiple := reload(multiple)
	if got := gotMultiple.GetString("primary_author"); got != "" {
		t.Fatalf("multiple primary_author = %q, want unset for review", got)
	}
	if got := gotMultiple.GetStringSlice("author"); !slices.Equal(got, []string{first.Id, second.Id}) {
		t.Fatalf("multiple legacy authors = %v, want both retained", got)
	}
	if got := reload(curated).GetString("primary_author"); got != second.Id {
		t.Fatalf("curated primary_author = %q, want existing %q preserved", got, second.Id)
	}
	if got := reload(authorless).GetString("primary_author"); got != "" {
		t.Fatalf("authorless primary_author = %q, want empty", got)
	}
	if got := reload(single).GetStringSlice("author"); !slices.Equal(got, []string{first.Id}) {
		t.Fatalf("single legacy author = %v, want retained", got)
	}

	// The dashboard review filter selects exactly the reported artworks.
	reviewed, err := app.FindRecordsByFilter("artworks", legacyAuthorReviewFilter, "id", 0, 0)
	if err != nil {
		t.Fatalf("review filter: %v", err)
	}
	if len(reviewed) != 1 || reviewed[0].Id != multiple.Id {
		t.Fatalf("review filter matched %d records, want only %s", len(reviewed), multiple.Id)
	}

	// Repeating the migration copies nothing new and reports the same review set.
	again, err := migrateLegacyArtworkAuthors(app)
	if err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if again.Copied != 0 || !slices.Equal(again.ReviewIDs, []string{multiple.Id}) {
		t.Fatalf("repeat report = %+v, want no copies and the same review set", again)
	}
	if err := copyLegacyArtworkAuthors(app); err != nil {
		t.Fatalf("registered migration: %v", err)
	}
}
