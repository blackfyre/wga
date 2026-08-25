package tours

import (
	"errors"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestIndexIsEmptyByDefaultAndOffersEditorialFilters(t *testing.T) {
	app := newTourTestApp(t)
	view, err := NewService(app).Index("")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(view.Rebuilt) != 0 || len(view.Original) != 0 {
		t.Fatalf("empty contracts returned %d rebuilt / %d original tours", len(view.Rebuilt), len(view.Original))
	}
	if len(view.Filters) != 5 {
		t.Fatalf("filters = %d, want All plus four kinds", len(view.Filters))
	}
	for index, label := range []string{"All", "Survey", "Artist", "Site", "Theme"} {
		if view.Filters[index].Label != label {
			t.Errorf("filter %d = %q, want %q", index, view.Filters[index].Label, label)
		}
	}
}

func TestEveryEditorialFilterSelectsOnlyItsPublishedKind(t *testing.T) {
	app := newTourTestApp(t)
	service := NewService(app)
	for _, kind := range []string{"survey", "artist", "site", "theme"} {
		seedRebuiltTour(t, app, kind, "published")
	}
	for _, kind := range []string{"survey", "artist", "site", "theme"} {
		view, err := service.Index(kind)
		if err != nil {
			t.Fatalf("%s filter: %v", kind, err)
		}
		if len(view.Rebuilt) != 1 || view.Rebuilt[0].Kind != kind {
			t.Errorf("%s filter returned %+v", kind, view.Rebuilt)
		}
	}
	all, err := service.Index("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Rebuilt) != 4 {
		t.Fatalf("all filter returned %d tours, want 4", len(all.Rebuilt))
	}
}

func TestPublishedTourBuildsEveryAddressedPageType(t *testing.T) {
	app := newTourTestApp(t)
	fixture := seedRebuiltTour(t, app, "survey", "published")
	seedRebuiltTour(t, app, "artist", "draft")

	service := NewService(app)
	index, err := service.Index("survey")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(index.Rebuilt) != 1 || index.Rebuilt[0].Slug != fixture.slug {
		t.Fatalf("published survey index = %+v", index.Rebuilt)
	}
	if index.Rebuilt[0].Number != "6a" {
		t.Fatalf("index card number = %q, want 6a", index.Rebuilt[0].Number)
	}
	if !strings.Contains(index.Rebuilt[0].ImageURL, "thumb=800x0") {
		t.Errorf("card URL = %q, want 800 profile", index.Rebuilt[0].ImageURL)
	}
	artist, err := service.Index("artist")
	if err != nil {
		t.Fatalf("artist index: %v", err)
	}
	if len(artist.Rebuilt) != 0 {
		t.Fatal("draft tour entered published index")
	}

	title, err := service.Page(fixture.slug, 1)
	if err != nil {
		t.Fatalf("title: %v", err)
	}
	if title.PageType != "title" || title.TotalPages != 5 || len(title.Contents) != 5 || title.NextURL == "" {
		t.Fatalf("title projection = %+v", title)
	}
	if title.Number != "6a" || title.RevisionSourceHash != "rev-source-hash" {
		t.Fatalf("title number/provenance = %+v", title)
	}
	if !strings.Contains(title.DisplayURL, "thumb=1000x0") {
		t.Errorf("title URL = %q, want 1000 profile", title.DisplayURL)
	}

	text, err := service.Page(fixture.slug, 2)
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	if text.PageType != "text" || text.Section != "Opening" || len(text.Blocks) != 1 {
		t.Fatalf("text projection = %+v", text)
	}
	if text.SourcePath != "/tours/source/text.html" || text.SourceHash != "text-hash" {
		t.Fatalf("text provenance = %+v", text)
	}
	if strings.Contains(text.Blocks[0].HTML, "script") || !strings.Contains(text.Blocks[0].HTML, "Approved prose") {
		t.Errorf("block was not safely sanitised: %q", text.Blocks[0].HTML)
	}

	picture, err := service.Page(fixture.slug, 3)
	if err != nil {
		t.Fatalf("picture: %v", err)
	}
	if picture.PageType != "picture" || !strings.Contains(picture.DisplayURL, "thumb=1400x0") {
		t.Fatalf("picture display = %q", picture.DisplayURL)
	}
	if strings.Contains(picture.ZoomURL, "thumb=") {
		t.Errorf("2000 profile should not upscale 1600px source: %q", picture.ZoomURL)
	}
	if picture.ArtworkCredit != "Approved credit" || picture.ArtworkURL != "/artworks/approved-work" {
		t.Fatalf("picture context = %+v", picture)
	}
	pictureRecords, err := app.FindRecordsByFilter("guided_tour_pages", "tour = {:tour} && page_type = 'picture'", "", 1, 0, map[string]any{"tour": fixture.tourID})
	if err != nil || len(pictureRecords) != 1 {
		t.Fatalf("find picture fixture: %v", err)
	}
	artwork, err := app.FindRecordById("artworks", pictureRecords[0].GetString("artwork"))
	if err != nil {
		t.Fatal(err)
	}
	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Page(fixture.slug, 3); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unpublished picture artwork error = %v", err)
	}

	indexPage, err := service.Page(fixture.slug, 4)
	if err != nil {
		t.Fatalf("index page: %v", err)
	}
	if indexPage.PageType != "list" || len(indexPage.IndexRows) != 2 {
		t.Fatalf("index projection = %+v", indexPage)
	}
	if indexPage.IndexRows[0].TargetPath != "/artists/approved" || indexPage.IndexRows[1].TargetPath != "" {
		t.Fatalf("safe target handling = %+v", indexPage.IndexRows)
	}

	sources, err := service.Page(fixture.slug, 5)
	if err != nil {
		t.Fatalf("sources: %v", err)
	}
	if sources.PageType != "sources" || len(sources.Sources) != 1 || sources.NextURL != "" || sources.PreviousURL == "" {
		t.Fatalf("sources projection = %+v", sources)
	}
	if _, err := service.Page(fixture.slug, 6); !errors.Is(err, ErrNotFound) {
		t.Fatalf("page beyond sources error = %v, want not found", err)
	}
}

func TestPublicationRevisionAndLegacySafetyDenyInvalidReads(t *testing.T) {
	app := newTourTestApp(t)
	published := seedRebuiltTour(t, app, "theme", "published")
	draft := seedRebuiltTour(t, app, "site", "draft")
	withdrawn := seedRebuiltTour(t, app, "artist", "withdrawn")
	service := NewService(app)
	if _, err := service.Page(draft.slug, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("draft page error = %v", err)
	}
	if _, err := service.Page(withdrawn.slug, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("withdrawn page error = %v", err)
	}

	tour, _ := app.FindRecordById("guided_tours", published.tourID)
	otherTour := seedOriginalTour(t, app, "wrong-revision", "https://example.org/original")
	other, _ := app.FindRecordById("guided_tours", otherTour.tourID)
	tour.Set("published_revision", other.GetString("published_revision"))
	if err := app.Save(tour); err != nil {
		t.Fatalf("save mismatch: %v", err)
	}
	if _, err := service.Page(published.slug, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revision mismatch error = %v", err)
	}

	safe := seedOriginalTour(t, app, "safe-original", "https://example.org/original")
	page, err := service.Page(safe.slug, 1)
	if err != nil || page.LegacyURL != "https://example.org/original" || page.PresentationStatus != "original" {
		t.Fatalf("safe legacy page = %+v, err=%v", page, err)
	}
	if _, err := service.Page(safe.slug, 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("legacy numbered page error = %v", err)
	}
	unsafe := seedOriginalTour(t, app, "unsafe-original", "javascript:alert(1)")
	page, err = service.Page(unsafe.slug, 1)
	if err != nil || page.LegacyURL != "" {
		t.Fatalf("unsafe legacy destination exposed: %+v, err=%v", page, err)
	}
}

func TestSafeInternalPathRejectsBackslashesAndCrossOrigin(t *testing.T) {
	for _, raw := range []string{
		"/artists/approved",
		"/tours/example/title.html",
		"/artworks/a/b/c",
	} {
		if got := safeInternalPath(raw); got != raw {
			t.Errorf("safeInternalPath(%q) = %q, want %q", raw, got, raw)
		}
	}
	for _, raw := range []string{
		"",
		"artists/approved",
		"https://evil.example",
		"//evil.example",
		`\evil.example`,
		`/\evil.example`,
		`/artists/\evil.example`,
		`/artists/approved\evil.example`,
		"//",
	} {
		if got := safeInternalPath(raw); got != "" {
			t.Errorf("safeInternalPath(%q) = %q, want empty", raw, got)
		}
	}
}

func TestLegacyRouteResolvesCanonicalAddressWithoutOpenRedirect(t *testing.T) {
	app := newTourTestApp(t)
	fixture := seedRebuiltTour(t, app, "survey", "published")
	service := NewService(app)

	text, err := service.LegacyRoute("/tours/source/text.html")
	if err != nil || text != "/tours/"+fixture.slug+"/2" {
		t.Fatalf("text legacy route = %q, err=%v", text, err)
	}
	list, err := service.LegacyRoute("/tours/source/list.html")
	if err != nil || list != "/tours/"+fixture.slug+"/4" {
		t.Fatalf("list legacy route = %q, err=%v", list, err)
	}

	for _, raw := range []string{"/tours/source/missing.html", "//evil.example", `/\evil.example`, "https://evil.example"} {
		if _, err := service.LegacyRoute(raw); !errors.Is(err, ErrNotFound) {
			t.Errorf("LegacyRoute(%q) err=%v, want not found", raw, err)
		}
	}

	tour, _ := app.FindRecordById("guided_tours", fixture.tourID)
	tour.Set("publication_status", "withdrawn")
	if err := app.Save(tour); err != nil {
		t.Fatalf("withdraw tour: %v", err)
	}
	if _, err := service.LegacyRoute("/tours/source/text.html"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("withdrawn legacy route err=%v, want not found", err)
	}
}

func TestRebuiltTourWithoutProvenanceIsRejected(t *testing.T) {
	app := newTourTestApp(t)
	fixture := seedRebuiltTour(t, app, "survey", "published")
	service := NewService(app)

	tour, _ := app.FindRecordById("guided_tours", fixture.tourID)
	revision, _ := app.FindRecordById("guided_tour_revisions", tour.GetString("published_revision"))
	revision.Set("source_hash", "")
	if err := app.Save(revision); err != nil {
		t.Fatalf("strip revision provenance: %v", err)
	}
	if _, err := service.Page(fixture.slug, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revision without source_hash err=%v, want not found", err)
	}
	index, err := service.Index("survey")
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Rebuilt) != 0 {
		t.Fatalf("revision without source_hash entered index: %+v", index.Rebuilt)
	}

	revision.Set("source_hash", "rev-source-hash")
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
	if _, err := service.Page(fixture.slug, 2); !errors.Is(err, ErrNotFound) {
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

func TestIndexDerivesRebuiltPageCount(t *testing.T) {
	app := newTourTestApp(t)
	seedRebuiltTour(t, app, "survey", "published")
	service := NewService(app)

	index, err := service.Index("survey")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(index.Rebuilt) != 1 {
		t.Fatalf("rebuilt index = %+v", index.Rebuilt)
	}
	// title (1) + text/picture/list (3) + sources (1) = 5.
	if index.Rebuilt[0].Pages != 5 {
		t.Fatalf("derived page count = %d, want 5", index.Rebuilt[0].Pages)
	}
	if index.Rebuilt[0].Scope != "" {
		t.Fatalf("scope = %q, want empty until approved", index.Rebuilt[0].Scope)
	}
}

func TestContentsCarryStableSectionIdentity(t *testing.T) {
	app := newTourTestApp(t)
	tour, revision := seedTourBase(t, app, "dup-section", "survey", "published", "rebuilt", "")
	sectionA := saveRecord(t, app, "guided_tour_sections", map[string]any{"tour": tour.Id, "revision": revision.Id, "section_order": 1, "title": "Opening"})
	sectionB := saveRecord(t, app, "guided_tour_sections", map[string]any{"tour": tour.Id, "revision": revision.Id, "section_order": 2, "title": "Opening"})
	saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": sectionA.Id, "page_position": 1, "page_type": "text", "title": "First", "source_page_id": "s1", "source_path": "/tours/s1.html", "source_hash": "h1"})
	saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": sectionB.Id, "page_position": 2, "page_type": "text", "title": "Second", "source_page_id": "s2", "source_path": "/tours/s2.html", "source_hash": "h2"})

	page, err := NewService(app).Page("dup-section", 2)
	if err != nil {
		t.Fatalf("page: %v", err)
	}
	if len(page.Contents) != 3 {
		t.Fatalf("contents = %+v", page.Contents)
	}
	if page.Contents[1].SectionID == "" || page.Contents[2].SectionID == "" {
		t.Fatalf("missing section identity: %+v", page.Contents)
	}
	if page.Contents[1].SectionID == page.Contents[2].SectionID {
		t.Fatalf("distinct sections share identity: %+v", page.Contents)
	}
	if page.Contents[1].Section != "Opening" || page.Contents[2].Section != "Opening" {
		t.Fatalf("identical titles not preserved: %+v", page.Contents)
	}
}

type tourFixture struct{ slug, tourID string }

func newTourTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	artworks := testCollection("Artworks", "artworks", &core.TextField{Name: "title"}, &core.TextField{Name: "image"}, &core.NumberField{Name: "image_width"}, &core.BoolField{Name: "published"})
	saveCollection(t, app, artworks)
	editors := testCollection("Guided_tour_editors", "guided_tour_editors", &core.TextField{Name: "editor_key"}, &core.TextField{Name: "name"})
	saveCollection(t, app, editors)
	tours := testCollection("Guided_tours", "guided_tours",
		&core.TextField{Name: "slug"}, &core.TextField{Name: "title"}, &core.TextField{Name: "blurb"}, &core.TextField{Name: "kind"},
		&core.TextField{Name: "tour_number"}, &core.RelationField{Name: "editor", CollectionId: editors.Id, MaxSelect: 1}, &core.NumberField{Name: "series_position"},
		&core.NumberField{Name: "published_year"}, &core.NumberField{Name: "revised_year"}, &core.TextField{Name: "legacy_url"},
		&core.TextField{Name: "presentation_status"}, &core.TextField{Name: "publication_status"})
	saveCollection(t, app, tours)
	revisions := testCollection("Guided_tour_revisions", "guided_tour_revisions", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.TextField{Name: "revision_key"}, &core.NumberField{Name: "revision_number"}, &core.TextField{Name: "label"}, &core.TextField{Name: "source_hash"})
	saveCollection(t, app, revisions)
	tours.Fields.Add(&core.RelationField{Name: "published_revision", CollectionId: revisions.Id, MaxSelect: 1})
	if err := app.Save(tours); err != nil {
		t.Fatalf("add revision relation: %v", err)
	}
	sections := testCollection("Guided_tour_sections", "guided_tour_sections", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.RelationField{Name: "revision", CollectionId: revisions.Id, MaxSelect: 1}, &core.NumberField{Name: "section_order"}, &core.TextField{Name: "title"})
	saveCollection(t, app, sections)
	pages := testCollection("Guided_tour_pages", "guided_tour_pages", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.RelationField{Name: "revision", CollectionId: revisions.Id, MaxSelect: 1}, &core.RelationField{Name: "section", CollectionId: sections.Id, MaxSelect: 1},
		&core.NumberField{Name: "page_position"}, &core.TextField{Name: "page_type"}, &core.TextField{Name: "title"}, &core.TextField{Name: "dateline"},
		&core.TextField{Name: "source_page_id"}, &core.TextField{Name: "source_path"}, &core.TextField{Name: "source_hash"},
		&core.RelationField{Name: "artwork", CollectionId: artworks.Id, MaxSelect: 1}, &core.TextField{Name: "credit"}, &core.TextField{Name: "work_target_path"})
	saveCollection(t, app, pages)
	saveCollection(t, app, testCollection("Guided_tour_blocks", "guided_tour_blocks", &core.RelationField{Name: "page", CollectionId: pages.Id, MaxSelect: 1},
		&core.NumberField{Name: "block_order"}, &core.TextField{Name: "block_kind"}, &core.EditorField{Name: "content_html"}))
	saveCollection(t, app, testCollection("Guided_tour_index_rows", "guided_tour_index_rows", &core.RelationField{Name: "page", CollectionId: pages.Id, MaxSelect: 1},
		&core.NumberField{Name: "row_order"}, &core.TextField{Name: "name"}, &core.TextField{Name: "dates"}, &core.TextField{Name: "note"}, &core.TextField{Name: "target_path"}))
	saveCollection(t, app, testCollection("Guided_tour_bibliography", "guided_tour_bibliography", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.RelationField{Name: "revision", CollectionId: revisions.Id, MaxSelect: 1}, &core.NumberField{Name: "item_order"}, &core.TextField{Name: "citation"}))
	saveCollection(t, app, testCollection("Guided_tour_legacy_routes", "guided_tour_legacy_routes",
		&core.TextField{Name: "legacy_path"}, &core.RelationField{Name: "tour_page", CollectionId: pages.Id, MaxSelect: 1}))
	return app
}

func testCollection(name, id string, fields ...core.Field) *core.Collection {
	collection := core.NewBaseCollection(name)
	collection.Id = id
	collection.MarkAsNew()
	collection.Fields.Add(fields...)
	return collection
}
func saveCollection(t *testing.T, app core.App, collection *core.Collection) {
	t.Helper()
	if err := app.Save(collection); err != nil {
		t.Fatalf("save %s: %v", collection.Name, err)
	}
}
func saveRecord(t *testing.T, app core.App, collection string, values map[string]any) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(coll)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return record
}

func seedTourBase(t *testing.T, app core.App, slug, kind, status, presentation, legacyURL string) (*core.Record, *core.Record) {
	editor := saveRecord(t, app, "guided_tour_editors", map[string]any{"editor_key": slug, "name": "Named Editor"})
	tour := saveRecord(t, app, "guided_tours", map[string]any{"slug": slug, "title": "A Tour", "blurb": "A bounded editorial reading.", "kind": kind,
		"tour_number": "6a", "editor": editor.Id, "series_position": 1, "published_year": 2001, "revised_year": 2002, "legacy_url": legacyURL,
		"presentation_status": presentation, "publication_status": status})
	revision := saveRecord(t, app, "guided_tour_revisions", map[string]any{"tour": tour.Id, "revision_key": "r1", "revision_number": 1, "label": "Revised edition", "source_hash": "rev-source-hash"})
	tour.Set("published_revision", revision.Id)
	if err := app.Save(tour); err != nil {
		t.Fatalf("publish revision: %v", err)
	}
	return tour, revision
}

func seedRebuiltTour(t *testing.T, app core.App, kind, status string) tourFixture {
	slug := kind + "-tour-" + status
	tour, revision := seedTourBase(t, app, slug, kind, status, "rebuilt", "")
	section := saveRecord(t, app, "guided_tour_sections", map[string]any{"tour": tour.Id, "revision": revision.Id, "section_order": 1, "title": "Opening"})
	textPage := saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": section.Id, "page_position": 1,
		"page_type": "text", "title": "Text page", "source_page_id": "src-text", "source_path": "/tours/source/text.html", "source_hash": "text-hash"})
	saveRecord(t, app, "guided_tour_blocks", map[string]any{"page": textPage.Id, "block_order": 1, "block_kind": "prose", "content_html": "<p>Approved prose</p><script>alert(1)</script>"})
	artwork := saveRecord(t, app, "artworks", map[string]any{"title": "Published Work", "image": "work.jpg", "image_width": 1600, "published": true})
	saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": section.Id, "page_position": 2,
		"page_type": "picture", "title": "Picture page", "source_page_id": "src-picture", "source_path": "/tours/source/picture.html", "source_hash": "picture-hash",
		"artwork": artwork.Id, "credit": "Approved credit", "work_target_path": "/artworks/approved-work"})
	indexPage := saveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": section.Id, "page_position": 3,
		"page_type": "list", "title": "Index page", "source_page_id": "src-list", "source_path": "/tours/source/list.html", "source_hash": "list-hash"})
	saveRecord(t, app, "guided_tour_index_rows", map[string]any{"page": indexPage.Id, "row_order": 1, "name": "Safe", "dates": "1500–1550", "note": "Note", "target_path": "/artists/approved"})
	saveRecord(t, app, "guided_tour_index_rows", map[string]any{"page": indexPage.Id, "row_order": 2, "name": "Unsafe", "target_path": "https://evil.example"})
	saveRecord(t, app, "guided_tour_bibliography", map[string]any{"tour": tour.Id, "revision": revision.Id, "item_order": 1, "citation": "Approved source"})
	saveRecord(t, app, "guided_tour_legacy_routes", map[string]any{"legacy_path": "/tours/source/text.html", "tour_page": textPage.Id})
	saveRecord(t, app, "guided_tour_legacy_routes", map[string]any{"legacy_path": "/tours/source/list.html", "tour_page": indexPage.Id})
	return tourFixture{slug: slug, tourID: tour.Id}
}
func seedOriginalTour(t *testing.T, app core.App, slug, legacyURL string) tourFixture {
	tour, _ := seedTourBase(t, app, slug, "site", "published", "original", legacyURL)
	return tourFixture{slug: slug, tourID: tour.Id}
}
