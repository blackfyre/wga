package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

var guidedTourIndexNames = []string{
	"pbx_guided_tour_editor_key", "pbx_guided_tour_slug", "pbx_guided_tour_source",
	"pbx_guided_tour_listing", "pbx_guided_tour_revision_key", "pbx_guided_tour_revision_number",
	"pbx_guided_tour_section_order", "pbx_guided_tour_page_order", "pbx_guided_tour_page_source",
	"pbx_guided_tour_block_order", "pbx_guided_tour_index_order", "pbx_guided_tour_bibliography_order",
	"pbx_guided_tour_legacy_path",
}

func init() {
	m.Register(addGuidedTourContracts, removeGuidedTourContracts)
}

// addGuidedTourContracts creates operationally private editorial collections.
// It deliberately inserts no records: tour content remains producer-owned and
// is unavailable until an approved curated projection exists.
func addGuidedTourContracts(app core.App) error {
	if err := ensureGuidedTourCollection(app, "Guided_tour_editors", "guided_tour_editors",
		textField("editor_key", true, false), textField("name", true, true)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tours", "guided_tours",
		textField("source_entry_id", false, false), numberField("series_position", false),
		textField("tour_number", false, false), textField("slug", true, false), textField("title", true, true),
		selectField("kind", []string{"survey", "artist", "site", "theme"}, true),
		textFieldWithMax("blurb", false, false, 1000), relationField("editor", "guided_tour_editors", 1, 1, true, false),
		numberField("published_year", false), numberField("revised_year", false), urlField("legacy_url", false),
		selectField("presentation_status", []string{"rebuilt", "original"}, true),
		selectField("publication_status", []string{"draft", "published", "withdrawn"}, true)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_revisions", "guided_tour_revisions",
		relationField("tour", "guided_tours", 1, 1, true, true), textField("revision_key", true, false),
		numberField("revision_number", true), textField("label", false, false), dateField("published_at", false),
		textField("source_hash", true, false)); err != nil {
		return err
	}
	tours, err := app.FindCollectionByNameOrId("guided_tours")
	if err != nil {
		return err
	}
	if tours.Fields.GetByName("published_revision") == nil {
		tours.Fields.Add(relationField("published_revision", "guided_tour_revisions", 0, 1, false, false))
		if err := app.Save(tours); err != nil {
			return err
		}
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_sections", "guided_tour_sections",
		relationField("tour", "guided_tours", 1, 1, true, true), relationField("revision", "guided_tour_revisions", 1, 1, true, true),
		numberField("section_order", true), textField("title", true, true)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_pages", "guided_tour_pages",
		relationField("tour", "guided_tours", 1, 1, true, true), relationField("revision", "guided_tour_revisions", 1, 1, true, true),
		relationField("section", "guided_tour_sections", 0, 1, false, false), numberField("page_position", true),
		selectField("page_type", []string{"text", "picture", "list"}, true), textField("title", true, true),
		textField("dateline", false, false), textField("source_page_id", true, false),
		textField("source_path", true, false), textField("source_hash", true, false),
		textField("image_url", false, false), textField("credit", false, false), textField("work_target_path", false, false),
		relationField("artwork", "artworks", 0, 1, false, false)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_blocks", "guided_tour_blocks",
		relationField("page", "guided_tour_pages", 1, 1, true, true), numberField("block_order", true),
		selectField("block_kind", []string{"prose", "heading", "quote", "note"}, true), editorField("content_html", true)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_index_rows", "guided_tour_index_rows",
		relationField("page", "guided_tour_pages", 1, 1, true, true), numberField("row_order", true),
		textField("name", true, true), textField("dates", false, false), textField("note", false, false),
		textField("target_path", false, false)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_bibliography", "guided_tour_bibliography",
		relationField("tour", "guided_tours", 1, 1, true, true), relationField("revision", "guided_tour_revisions", 1, 1, true, true),
		numberField("item_order", true), textFieldWithMax("citation", true, true, 5000)); err != nil {
		return err
	}
	if err := ensureGuidedTourCollection(app, "Guided_tour_legacy_routes", "guided_tour_legacy_routes",
		textField("legacy_path", true, true), relationField("tour_page", "guided_tour_pages", 1, 1, true, false)); err != nil {
		return err
	}

	indexes := map[string][]string{
		"guided_tour_editors": {"CREATE UNIQUE INDEX `pbx_guided_tour_editor_key` ON `Guided_tour_editors` (`editor_key`)"},
		"guided_tours": {
			"CREATE UNIQUE INDEX `pbx_guided_tour_slug` ON `Guided_tours` (`slug`)",
			"CREATE UNIQUE INDEX `pbx_guided_tour_source` ON `Guided_tours` (`source_entry_id`) WHERE `source_entry_id` != ''",
			"CREATE INDEX `pbx_guided_tour_listing` ON `Guided_tours` (`publication_status`, `kind`, `series_position`, `id`)",
		},
		"guided_tour_revisions": {
			"CREATE UNIQUE INDEX `pbx_guided_tour_revision_key` ON `Guided_tour_revisions` (`tour`, `revision_key`)",
			"CREATE UNIQUE INDEX `pbx_guided_tour_revision_number` ON `Guided_tour_revisions` (`tour`, `revision_number`)",
		},
		"guided_tour_sections": {"CREATE UNIQUE INDEX `pbx_guided_tour_section_order` ON `Guided_tour_sections` (`revision`, `section_order`)"},
		"guided_tour_pages": {
			"CREATE UNIQUE INDEX `pbx_guided_tour_page_order` ON `Guided_tour_pages` (`revision`, `page_position`)",
			"CREATE UNIQUE INDEX `pbx_guided_tour_page_source` ON `Guided_tour_pages` (`tour`, `source_page_id`) WHERE `source_page_id` != ''",
		},
		"guided_tour_blocks":        {"CREATE UNIQUE INDEX `pbx_guided_tour_block_order` ON `Guided_tour_blocks` (`page`, `block_order`)"},
		"guided_tour_index_rows":    {"CREATE UNIQUE INDEX `pbx_guided_tour_index_order` ON `Guided_tour_index_rows` (`page`, `row_order`)"},
		"guided_tour_bibliography":  {"CREATE UNIQUE INDEX `pbx_guided_tour_bibliography_order` ON `Guided_tour_bibliography` (`revision`, `item_order`)"},
		"guided_tour_legacy_routes": {"CREATE UNIQUE INDEX `pbx_guided_tour_legacy_path` ON `Guided_tour_legacy_routes` (`legacy_path`)"},
	}
	for collectionName, additions := range indexes {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			return err
		}
		for _, addition := range additions {
			collection.Indexes = appendGuidedTourIndex(collection.Indexes, addition)
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}

func ensureGuidedTourCollection(app core.App, name string, id string, fields ...core.Field) error {
	collection, err := app.FindCollectionByNameOrId(id)
	if err != nil {
		return saveCurrentCollection(app, currentCollection(name, id, fields...))
	}
	for _, field := range fields {
		if collection.Fields.GetByName(field.GetName()) == nil {
			collection.Fields.Add(field)
		}
	}
	return app.Save(collection)
}

func appendGuidedTourIndex(indexes []string, addition string) []string {
	for _, name := range guidedTourIndexNames {
		if strings.Contains(addition, name) {
			for _, index := range indexes {
				if strings.Contains(index, name) {
					return indexes
				}
			}
			break
		}
	}
	return append(indexes, addition)
}

// removeGuidedTourContracts removes only feature query indexes. Collections,
// fields and editorial records remain intact for forward-compatible rollback.
func removeGuidedTourContracts(app core.App) error {
	for _, collectionName := range []string{
		"guided_tour_editors", "guided_tours", "guided_tour_revisions", "guided_tour_sections",
		"guided_tour_pages", "guided_tour_blocks", "guided_tour_index_rows",
		"guided_tour_bibliography", "guided_tour_legacy_routes",
	} {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			return err
		}
		kept := make([]string, 0, len(collection.Indexes))
		for _, index := range collection.Indexes {
			remove := false
			for _, name := range guidedTourIndexNames {
				if strings.Contains(index, name) {
					remove = true
					break
				}
			}
			if !remove {
				kept = append(kept, index)
			}
		}
		collection.Indexes = kept
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
