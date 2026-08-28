package tours

import (
	"errors"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// Synthetic fixture contract for task 8.6.
//
// The identifiers below are deliberately distinct from any producer or release
// value so the fixture can never collide with real editorial data, and every
// human-readable string names the material as synthetic. These two tours are
// the same records the Playwright harness in internal/handlers/tours exposes.
//
// Rebuilt tour:
//   slug synthetic-rebuilt-tour-task86, title "Synthetic Rebuilt Tour (Task 8.6 Fixture)",
//   kind survey, tour_number T86, presentation_status rebuilt, publication_status published,
//   revision source_hash syn-revision-source-hash, one section "Synthetic Opening",
//   pages text/picture/list with source_path+source_hash, one bibliography entry.
//   The picture artwork has a 2048px source width, wider than the 2000 viewer
//   profile, so the deliberate plates resolve to a 1400x0 display and a 2000x0
//   zoom downscale rather than an upscale or an original-file fallback.
//   Total addresses: 5 (title + text + picture + list + sources).
//
// Original tour:
//   slug synthetic-legacy-tour-task86, title "Synthetic Legacy Tour (Task 8.6 Fixture)",
//   kind site, tour_number T87, presentation_status original, publication_status published,
//   legacy_url https://example.org/synthetic-legacy-original.

const (
	syntheticRebuiltSlug  = "synthetic-rebuilt-tour-task86"
	syntheticLegacySlug   = "synthetic-legacy-tour-task86"
	syntheticRevisionHash = "syn-revision-source-hash"
)

func seedSyntheticTourBase(t *testing.T, app core.App, slug, title, kind, number, presentation, legacyURL string) (*core.Record, *core.Record) {
	t.Helper()
	editor := saveRecord(t, app, "guided_tour_editors", map[string]any{"editor_key": "syn-editor-" + slug, "name": "Synthetic Fixture Editor"})
	tour := saveRecord(t, app, "guided_tours", map[string]any{"slug": slug, "title": title,
		"blurb": "Deterministic synthetic tour. Not editorial content.", "kind": kind,
		"tour_number": number, "editor": editor.Id, "series_position": 1, "published_year": 2001, "revised_year": 2002,
		"legacy_url": legacyURL, "presentation_status": presentation, "publication_status": "published"})
	revision := saveRecord(t, app, "guided_tour_revisions", map[string]any{"tour": tour.Id, "revision_key": "syn-rev-1",
		"revision_number": 1, "label": "Synthetic revision 1", "source_hash": syntheticRevisionHash})
	tour.Set("published_revision", revision.Id)
	if err := app.Save(tour); err != nil {
		t.Fatalf("publish synthetic revision: %v", err)
	}
	return tour, revision
}

func seedSyntheticRebuiltTour(t *testing.T, app core.App) tourFixture {
	t.Helper()
	tour, revision := seedSyntheticTourBase(t, app, syntheticRebuiltSlug,
		"Synthetic Rebuilt Tour (Task 8.6 Fixture)", "survey", "T86", "rebuilt", "")
	section := saveRecord(t, app, "guided_tour_sections", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"section_order": 1, "title": "Synthetic Opening"})
	textPage := saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"section": section.Id, "page_position": 1, "page_type": "text", "title": "Synthetic Text Page",
		"source_page_id": "syn-text", "source_path": "/tours/source/synthetic-text.html", "source_hash": "syn-text-hash"})
	saveRecord(t, app, "guided_tour_blocks", map[string]any{"page": textPage.Id, "block_order": 1, "block_kind": "prose",
		"content_html": "<p>Synthetic tour prose. Not editorial content.</p>"})
	artwork := saveRecord(t, app, "artworks", map[string]any{"title": "Synthetic Work", "image": "synthetic-work.jpg",
		"image_width": 2048, "published": true})
	saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": section.Id,
		"page_position": 2, "page_type": "picture", "title": "Synthetic Picture Page",
		"source_page_id": "syn-picture", "source_path": "/tours/source/synthetic-picture.html", "source_hash": "syn-picture-hash",
		"artwork": artwork.Id, "credit": "Synthetic credit", "work_target_path": "/artworks/synthetic-work"})
	listPage := saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"section": section.Id, "page_position": 3, "page_type": "list", "title": "Synthetic Index Page",
		"source_page_id": "syn-list", "source_path": "/tours/source/synthetic-list.html", "source_hash": "syn-list-hash"})
	saveRecord(t, app, "guided_tour_index_rows", map[string]any{"page": listPage.Id, "row_order": 1, "name": "Synthetic safe row",
		"dates": "1500–1550", "note": "Synthetic note", "target_path": "/artists/synthetic"})
	saveRecord(t, app, "guided_tour_index_rows", map[string]any{"page": listPage.Id, "row_order": 2, "name": "Synthetic unsafe row",
		"target_path": "https://evil.example"})
	saveRecord(t, app, "guided_tour_bibliography", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"item_order": 1, "citation": "Synthetic source citation. Not a real source."})
	saveRecord(t, app, "guided_tour_legacy_routes", map[string]any{"legacy_path": "/tours/source/synthetic-text.html", "tour_page": textPage.Id})
	saveRecord(t, app, "guided_tour_legacy_routes", map[string]any{"legacy_path": "/tours/source/synthetic-list.html", "tour_page": listPage.Id})
	return tourFixture{slug: syntheticRebuiltSlug, tourID: tour.Id}
}

func seedSyntheticOriginalTour(t *testing.T, app core.App) tourFixture {
	t.Helper()
	tour, _ := seedSyntheticTourBase(t, app, syntheticLegacySlug,
		"Synthetic Legacy Tour (Task 8.6 Fixture)", "site", "T87", "original", "https://example.org/synthetic-legacy-original")
	return tourFixture{slug: syntheticLegacySlug, tourID: tour.Id}
}

func TestSyntheticFixtureIndexGroupsRebuiltAndOriginalByKind(t *testing.T) {
	app := newTourTestApp(t)
	seedSyntheticRebuiltTour(t, app)
	seedSyntheticOriginalTour(t, app)

	service := NewService(app)
	all, err := service.Index("")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(all.Rebuilt) != 1 || all.Rebuilt[0].Slug != syntheticRebuiltSlug {
		t.Fatalf("rebuilt index = %+v", all.Rebuilt)
	}
	if len(all.Original) != 1 || all.Original[0].Slug != syntheticLegacySlug {
		t.Fatalf("original index = %+v", all.Original)
	}
	if all.Rebuilt[0].Number != "T86" || all.Original[0].Number != "T87" {
		t.Fatalf("textual numbers = %q / %q, want T86 / T87", all.Rebuilt[0].Number, all.Original[0].Number)
	}
	if all.Rebuilt[0].Pages != 5 {
		t.Fatalf("rebuilt page count = %d, want 5", all.Rebuilt[0].Pages)
	}

	survey, err := service.Index("survey")
	if err != nil {
		t.Fatalf("survey index: %v", err)
	}
	if len(survey.Rebuilt) != 1 || len(survey.Original) != 0 {
		t.Fatalf("survey filter = %+v", survey)
	}
	site, err := service.Index("site")
	if err != nil {
		t.Fatalf("site index: %v", err)
	}
	if len(site.Rebuilt) != 0 || len(site.Original) != 1 {
		t.Fatalf("site filter = %+v", site)
	}
	artist, err := service.Index("artist")
	if err != nil {
		t.Fatalf("artist index: %v", err)
	}
	if len(artist.Rebuilt) != 0 || len(artist.Original) != 0 {
		t.Fatalf("artist filter = %+v", artist)
	}
}

func TestSyntheticFixtureProvenanceFailsClosed(t *testing.T) {
	app := newTourTestApp(t)
	fixture := seedSyntheticRebuiltTour(t, app)
	service := NewService(app)

	tour, _ := app.FindRecordById("guided_tours", fixture.tourID)
	revision, _ := app.FindRecordById("guided_tour_revisions", tour.GetString("published_revision"))
	revision.Set("source_hash", "")
	if err := app.Save(revision); err != nil {
		t.Fatalf("strip revision provenance: %v", err)
	}
	if _, err := service.Page(syntheticRebuiltSlug, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revision without source_hash err=%v, want not found", err)
	}
	index, err := service.Index("survey")
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Rebuilt) != 0 {
		t.Fatalf("revision without source_hash entered index: %+v", index.Rebuilt)
	}

	revision.Set("source_hash", syntheticRevisionHash)
	if err := app.Save(revision); err != nil {
		t.Fatalf("restore revision provenance: %v", err)
	}
	pages, err := app.FindRecordsByFilter("guided_tour_pages", "tour = {:tour}", "+page_position,+id", 1, 0, map[string]any{"tour": fixture.tourID})
	if err != nil || len(pages) == 0 {
		t.Fatalf("find first page: %v", err)
	}
	pages[0].Set("source_path", "")
	if err := app.Save(pages[0]); err != nil {
		t.Fatalf("strip page provenance: %v", err)
	}
	if _, err := service.Page(syntheticRebuiltSlug, 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("page without source_path err=%v, want not found", err)
	}
	index, err = service.Index("survey")
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Rebuilt) != 0 {
		t.Fatalf("rebuilt tour with missing page provenance entered index: %+v", index.Rebuilt)
	}
}

func TestSyntheticFixtureLegacyTourIsHonestAndSafe(t *testing.T) {
	app := newTourTestApp(t)
	seedSyntheticOriginalTour(t, app)
	service := NewService(app)

	page, err := service.Page(syntheticLegacySlug, 1)
	if err != nil {
		t.Fatalf("legacy title: %v", err)
	}
	if page.PresentationStatus != "original" || page.LegacyURL != "https://example.org/synthetic-legacy-original" {
		t.Fatalf("legacy projection = %+v", page)
	}
	if page.PageType != "title" || page.TotalPages != 1 {
		t.Fatalf("legacy page shape = %+v", page)
	}
	if _, err := service.Page(syntheticLegacySlug, 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("legacy numbered page err=%v, want not found", err)
	}
}

func TestSyntheticFixtureStableNumberRevisionAndSections(t *testing.T) {
	app := newTourTestApp(t)
	seedSyntheticRebuiltTour(t, app)
	service := NewService(app)

	title, err := service.Page(syntheticRebuiltSlug, 1)
	if err != nil {
		t.Fatalf("title: %v", err)
	}
	if title.Number != "T86" || title.Revision != "Synthetic revision 1" || title.RevisionSourceHash != syntheticRevisionHash {
		t.Fatalf("title number/revision = %+v", title)
	}
	if title.TotalPages != 5 || len(title.Contents) != 5 {
		t.Fatalf("title contents = %+v", title)
	}

	text, err := service.Page(syntheticRebuiltSlug, 2)
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	if text.PageType != "text" || text.Section != "Synthetic Opening" {
		t.Fatalf("text section = %+v", text)
	}
	if text.SourcePath != "/tours/source/synthetic-text.html" || text.SourceHash != "syn-text-hash" {
		t.Fatalf("text provenance = %+v", text)
	}
	if !strings.Contains(text.Blocks[0].HTML, "Synthetic tour prose") {
		t.Fatalf("text block = %q", text.Blocks[0].HTML)
	}
}

func TestSyntheticFixtureDeliberatePlatesWithoutUpscale(t *testing.T) {
	app := newTourTestApp(t)
	seedSyntheticRebuiltTour(t, app)
	service := NewService(app)

	index, err := service.Index("survey")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if !strings.Contains(index.Rebuilt[0].ImageURL, "thumb=800x0") {
		t.Errorf("card URL = %q, want 800 profile", index.Rebuilt[0].ImageURL)
	}

	title, err := service.Page(syntheticRebuiltSlug, 1)
	if err != nil {
		t.Fatalf("title: %v", err)
	}
	if !strings.Contains(title.DisplayURL, "thumb=1000x0") {
		t.Errorf("title URL = %q, want 1000 profile", title.DisplayURL)
	}

	picture, err := service.Page(syntheticRebuiltSlug, 3)
	if err != nil {
		t.Fatalf("picture: %v", err)
	}
	if !strings.Contains(picture.DisplayURL, "thumb=1400x0") {
		t.Errorf("picture display URL = %q, want 1400 profile", picture.DisplayURL)
	}
	// The 2048px source is wider than the 2000 viewer profile, so the deliberate
	// zoom plate is a 2000x0 downscale — never an upscale, and never an
	// original-file fallback.
	if !strings.Contains(picture.ZoomURL, "thumb=2000x0") {
		t.Errorf("picture zoom URL = %q, want 2000 profile (downscale of 2048px source)", picture.ZoomURL)
	}
}
