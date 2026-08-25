package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestGuidedTourContractsUpDownAndReUpRetainEditorialData(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	if err := addGuidedTourContracts(app); err != nil {
		t.Fatalf("up: %v", err)
	}

	expected := map[string][]string{
		"guided_tour_editors":       {"editor_key", "name"},
		"guided_tours":              {"source_entry_id", "series_position", "tour_number", "slug", "title", "kind", "blurb", "editor", "published_year", "revised_year", "legacy_url", "presentation_status", "publication_status", "published_revision"},
		"guided_tour_revisions":     {"tour", "revision_key", "revision_number", "label", "published_at", "source_hash"},
		"guided_tour_sections":      {"tour", "revision", "section_order", "title"},
		"guided_tour_pages":         {"tour", "revision", "section", "page_position", "page_type", "title", "dateline", "source_page_id", "source_path", "source_hash", "image_url", "credit", "work_target_path", "artwork"},
		"guided_tour_blocks":        {"page", "block_order", "block_kind", "content_html"},
		"guided_tour_index_rows":    {"page", "row_order", "name", "dates", "note", "target_path"},
		"guided_tour_bibliography":  {"tour", "revision", "item_order", "citation"},
		"guided_tour_legacy_routes": {"legacy_path", "tour_page"},
	}
	for name, fields := range expected {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		for _, field := range fields {
			if collection.Fields.GetByName(field) == nil {
				t.Errorf("%s missing %s", name, field)
			}
		}
		for ruleName, rule := range map[string]*string{"list": collection.ListRule, "view": collection.ViewRule, "create": collection.CreateRule, "update": collection.UpdateRule, "delete": collection.DeleteRule} {
			if rule != nil {
				t.Errorf("%s %s rule is open", name, ruleName)
			}
		}
	}
	assertGuidedTourIndexes(t, app, true)

	editors, _ := app.FindCollectionByNameOrId("guided_tour_editors")
	editor := core.NewRecord(editors)
	editor.Set("editor_key", "fixture")
	editor.Set("name", "Fixture Editor")
	if err := app.Save(editor); err != nil {
		t.Fatalf("save editor: %v", err)
	}
	tourCollection, _ := app.FindCollectionByNameOrId("guided_tours")
	tour := core.NewRecord(tourCollection)
	tour.Set("slug", "retained-tour")
	tour.Set("title", "Retained Tour")
	tour.Set("kind", "survey")
	tour.Set("editor", editor.Id)
	tour.Set("presentation_status", "rebuilt")
	tour.Set("publication_status", "published")
	if err := app.Save(tour); err != nil {
		t.Fatalf("save tour: %v", err)
	}
	revisionCollection, _ := app.FindCollectionByNameOrId("guided_tour_revisions")
	revision := core.NewRecord(revisionCollection)
	revision.Set("tour", tour.Id)
	revision.Set("revision_key", "r1")
	revision.Set("revision_number", 1)
	revision.Set("source_hash", "retained-source-hash")
	if err := app.Save(revision); err != nil {
		t.Fatalf("save revision: %v", err)
	}
	tour.Set("published_revision", revision.Id)
	if err := app.Save(tour); err != nil {
		t.Fatalf("publish revision: %v", err)
	}

	if err := removeGuidedTourContracts(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := app.FindRecordById("guided_tour_editors", editor.Id); err != nil {
		t.Fatalf("rollback deleted editor data: %v", err)
	}
	if _, err := app.FindRecordById("guided_tours", tour.Id); err != nil {
		t.Fatalf("rollback deleted tour data: %v", err)
	}
	if _, err := app.FindRecordById("guided_tour_revisions", revision.Id); err != nil {
		t.Fatalf("rollback deleted revision data: %v", err)
	}
	assertGuidedTourIndexes(t, app, false)
	if err := addGuidedTourContracts(app); err != nil {
		t.Fatalf("re-up: %v", err)
	}
	assertGuidedTourIndexes(t, app, true)
	if _, err := app.FindRecordById("guided_tour_editors", editor.Id); err != nil {
		t.Fatalf("re-up lost editorial data: %v", err)
	}
	if err := addGuidedTourContracts(app); err != nil {
		t.Fatalf("idempotent up: %v", err)
	}
}

func TestGuidedTourContractsUseProducerTextAndListContracts(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	if err := createCurrentSchema(app); err != nil {
		t.Fatal(err)
	}
	if err := addGuidedTourContracts(app); err != nil {
		t.Fatal(err)
	}

	tours, err := app.FindCollectionByNameOrId("guided_tours")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tours.Fields.GetByName("tour_number").(*core.TextField); !ok {
		t.Errorf("tour_number is %T, want text", tours.Fields.GetByName("tour_number"))
	}

	pages, err := app.FindCollectionByNameOrId("guided_tour_pages")
	if err != nil {
		t.Fatal(err)
	}
	pageType, ok := pages.Fields.GetByName("page_type").(*core.SelectField)
	if !ok {
		t.Fatalf("page_type is %T, want select", pages.Fields.GetByName("page_type"))
	}
	hasValue := func(want string) bool {
		for _, value := range pageType.Values {
			if value == want {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"text", "picture", "list"} {
		if !hasValue(want) {
			t.Errorf("page_type values %v missing %q", pageType.Values, want)
		}
	}
	if hasValue("index") {
		t.Errorf("page_type still contains WGA-only index value: %v", pageType.Values)
	}

	sourcePageID, ok := pages.Fields.GetByName("source_page_id").(*core.TextField)
	if !ok || !sourcePageID.Required {
		t.Errorf("source_page_id = %#v, want required text field", pages.Fields.GetByName("source_page_id"))
	}

	revisions, err := app.FindCollectionByNameOrId("guided_tour_revisions")
	if err != nil {
		t.Fatal(err)
	}
	revisionSourceHash, ok := revisions.Fields.GetByName("source_hash").(*core.TextField)
	if !ok || !revisionSourceHash.Required {
		t.Errorf("revision source_hash = %#v, want required text field", revisions.Fields.GetByName("source_hash"))
	}

	for _, name := range []string{"source_path", "source_hash"} {
		field, ok := pages.Fields.GetByName(name).(*core.TextField)
		if !ok || !field.Required {
			t.Errorf("page %s = %#v, want required text field", name, pages.Fields.GetByName(name))
		}
	}
}

func TestGuidedTourContractsRejectMissingProvenance(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	if err := createCurrentSchema(app); err != nil {
		t.Fatal(err)
	}
	if err := addGuidedTourContracts(app); err != nil {
		t.Fatal(err)
	}

	editors, _ := app.FindCollectionByNameOrId("guided_tour_editors")
	editor := core.NewRecord(editors)
	editor.Set("editor_key", "e")
	editor.Set("name", "Editor")
	if err := app.Save(editor); err != nil {
		t.Fatal(err)
	}
	toursColl, _ := app.FindCollectionByNameOrId("guided_tours")
	tour := core.NewRecord(toursColl)
	tour.Set("slug", "provenance-tour")
	tour.Set("title", "Provenance Tour")
	tour.Set("kind", "survey")
	tour.Set("editor", editor.Id)
	tour.Set("presentation_status", "rebuilt")
	tour.Set("publication_status", "published")
	if err := app.Save(tour); err != nil {
		t.Fatal(err)
	}

	revisionsColl, _ := app.FindCollectionByNameOrId("guided_tour_revisions")
	revision := core.NewRecord(revisionsColl)
	revision.Set("tour", tour.Id)
	revision.Set("revision_key", "r1")
	revision.Set("revision_number", 1)
	if err := app.Save(revision); err == nil {
		t.Fatal("revision without source_hash was accepted")
	}

	revision.Set("source_hash", "hash")
	if err := app.Save(revision); err != nil {
		t.Fatal(err)
	}

	pagesColl, _ := app.FindCollectionByNameOrId("guided_tour_pages")
	page := core.NewRecord(pagesColl)
	page.Set("tour", tour.Id)
	page.Set("revision", revision.Id)
	page.Set("page_position", 1)
	page.Set("page_type", "text")
	page.Set("title", "Text page")
	page.Set("source_page_id", "src")
	if err := app.Save(page); err == nil {
		t.Fatal("page without source_path/source_hash was accepted")
	}
	page.Set("source_path", "/tours/source.html")
	page.Set("source_hash", "page-hash")
	if err := app.Save(page); err != nil {
		t.Fatal(err)
	}
}

func TestGuidedTourContractsStartEmpty(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	if err := createCurrentSchema(app); err != nil {
		t.Fatal(err)
	}
	if err := addGuidedTourContracts(app); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"guided_tour_editors", "guided_tours", "guided_tour_revisions", "guided_tour_sections", "guided_tour_pages", "guided_tour_blocks", "guided_tour_index_rows", "guided_tour_bibliography", "guided_tour_legacy_routes"} {
		records, err := app.FindRecordsByFilter(name, "", "", 0, 0)
		if err != nil || len(records) != 0 {
			t.Errorf("%s records=%d err=%v, want empty", name, len(records), err)
		}
	}
}

func assertGuidedTourIndexes(t *testing.T, app core.App, present bool) {
	t.Helper()
	joined := ""
	for _, name := range []string{"guided_tour_editors", "guided_tours", "guided_tour_revisions", "guided_tour_sections", "guided_tour_pages", "guided_tour_blocks", "guided_tour_index_rows", "guided_tour_bibliography", "guided_tour_legacy_routes"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		joined += strings.Join(collection.Indexes, "\n")
	}
	for _, name := range guidedTourIndexNames {
		got := strings.Contains(joined, name)
		if got != present {
			t.Errorf("index %s present=%v, want %v", name, got, present)
		}
	}
}
