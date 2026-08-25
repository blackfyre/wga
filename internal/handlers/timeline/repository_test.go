package timeline

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func newTimelineApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	createTimelineCollections(t, app)

	return app
}

func createTimelineCollections(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	periods := core.NewBaseCollection("Art_periods")
	periods.Id = "art_periods"
	periods.MarkAsNew()
	periods.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.NumberField{Name: "start"},
		&core.NumberField{Name: "end"},
		&core.TextField{Name: "description"},
	)
	if err := app.Save(periods); err != nil {
		t.Fatalf("save art periods: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Name: "published"},
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "date_end"},
		&core.BoolField{Name: "is_circa"},
		&core.TextField{Name: "date_qualifier"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}
}

func saveTimelineRecord(t *testing.T, app *pocketbase.PocketBase, collection string, id string, fields map[string]any) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	record := core.NewRecord(coll)
	record.Id = id
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

func TestRepositoryArtworkBoundsExcludesUnpublishedAndUnknown(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "Pub", "published": true})
	saveTimelineRecord(t, app, "artists", "artisthid000001", map[string]any{"name": "Hid", "published": false})

	saveTimelineRecord(t, app, "artworks", "artworkdate0001", map[string]any{
		"title": "Dated", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1500, "date_end": 1510,
	})
	saveTimelineRecord(t, app, "artworks", "artworkrange001", map[string]any{
		"title": "Range", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1600, "date_end": 1620,
	})
	saveTimelineRecord(t, app, "artworks", "artworkhidden01", map[string]any{
		"title": "Hidden", "author": []string{"artistpub000001"}, "published": false,
		"date_start": 1000, "date_end": 1010,
	})
	saveTimelineRecord(t, app, "artworks", "artworkunpub001", map[string]any{
		"title": "UnpubAuthor", "author": []string{"artisthid000001"}, "published": true,
		"date_start": 1700, "date_end": 1710,
	})
	saveTimelineRecord(t, app, "artworks", "artworkunkno001", map[string]any{
		"title": "Unknown", "author": []string{"artistpub000001"}, "published": true,
	})

	repo := newRepository(app)
	min, max, err := repo.artworkBounds()
	if err != nil {
		t.Fatalf("artworkBounds: %v", err)
	}
	if min != 1500 || max != 1620 {
		t.Errorf("artworkBounds = (%d, %d), want (1500, 1620)", min, max)
	}
}

func TestRepositoryCountWorksOverlapSemantics(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "Pub", "published": true})

	saveTimelineRecord(t, app, "artworks", "artworkspan0001", map[string]any{
		"title": "Span", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1400, "date_end": 1450,
	})
	saveTimelineRecord(t, app, "artworks", "artworkpoint001", map[string]any{
		"title": "Point", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1500,
	})
	saveTimelineRecord(t, app, "artworks", "artworklater001", map[string]any{
		"title": "Later", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1700, "date_end": 1750,
	})

	repo := newRepository(app)

	count, err := repo.countWorks(1450, 1550)
	if err != nil {
		t.Fatalf("countWorks: %v", err)
	}
	// "Span" (ends 1450) and "Point" (1500) overlap; "Later" (1700) does not.
	if count != 2 {
		t.Errorf("countWorks(1450, 1550) = %d, want 2", count)
	}
}

func TestRepositoryListWorksBoundedAndOrdered(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "Pub", "published": true})

	years := []int{1500, 1600, 1550, 1500}
	ids := []string{"artworklst00001", "artworklst00002", "artworklst00003", "artworklst00004"}
	for i, year := range years {
		saveTimelineRecord(t, app, "artworks", ids[i], map[string]any{
			"title": "T", "author": []string{"artistpub000001"}, "published": true,
			"date_start": year,
		})
	}

	repo := newRepository(app)
	rows, err := repo.listWorks(1450, 1700, 2, 0)
	if err != nil {
		t.Fatalf("listWorks: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("listWorks length = %d, want bounded 2", len(rows))
	}
	if rows[0].DateStart != 1500 || rows[1].DateStart != 1500 {
		t.Errorf("listWorks order starts = (%d, %d), want (1500, 1500)", rows[0].DateStart, rows[1].DateStart)
	}
}

func TestRepositoryListWorksHonorsOffset(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "Pub", "published": true})

	years := []int{1500, 1510, 1520}
	ids := []string{"artworkofs00001", "artworkofs00002", "artworkofs00003"}
	for i, year := range years {
		saveTimelineRecord(t, app, "artworks", ids[i], map[string]any{
			"title": "T", "author": []string{"artistpub000001"}, "published": true,
			"date_start": year,
		})
	}

	repo := newRepository(app)
	rows, err := repo.listWorks(1450, 1700, 2, 1)
	if err != nil {
		t.Fatalf("listWorks: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("listWorks length = %d, want 2", len(rows))
	}
	if rows[0].DateStart != 1510 || rows[1].DateStart != 1520 {
		t.Errorf("listWorks offset starts = (%d, %d), want (1510, 1520)", rows[0].DateStart, rows[1].DateStart)
	}
}

func TestRepositoryDateSpansOverlapSemantics(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "Pub", "published": true})

	saveTimelineRecord(t, app, "artworks", "artworkden00001", map[string]any{
		"title": "A", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1400, "date_end": 1450,
	})
	saveTimelineRecord(t, app, "artworks", "artworkden00002", map[string]any{
		"title": "B", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1500,
	})
	saveTimelineRecord(t, app, "artworks", "artworkden00003", map[string]any{
		"title": "C", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1700, "date_end": 1750,
	})

	repo := newRepository(app)
	spans, err := repo.dateSpans(1450, 1550)
	if err != nil {
		t.Fatalf("dateSpans: %v", err)
	}
	if len(spans) != 2 {
		t.Fatalf("dateSpans length = %d, want 2 (A ends 1450, B at 1500; C at 1700 does not)", len(spans))
	}
}

func TestRepositoryListPeriodsOrdered(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "art_periods", "periodlate00001", map[string]any{"name": "Late", "start": 1500, "end": 1700, "description": "later"})
	saveTimelineRecord(t, app, "art_periods", "periodyearly001", map[string]any{"name": "Early", "start": 1000, "end": 1200, "description": "earlier"})

	repo := newRepository(app)
	periods, err := repo.listPeriods()
	if err != nil {
		t.Fatalf("listPeriods: %v", err)
	}
	if len(periods) != 2 {
		t.Fatalf("listPeriods length = %d, want 2", len(periods))
	}
	if periods[0].name != "Early" || periods[1].name != "Late" {
		t.Errorf("listPeriods order = (%q, %q), want (Early, Late)", periods[0].name, periods[1].name)
	}
}

func TestRepositoryListPeriodsSortsTiesDeterministically(t *testing.T) {
	app := newTimelineApp(t)

	// Equal start AND equal name so only the +id key breaks the tie.
	saveTimelineRecord(t, app, "art_periods", "periodtie000001", map[string]any{"name": "Same", "start": 1400, "end": 1500, "description": "b"})
	saveTimelineRecord(t, app, "art_periods", "periodtie000002", map[string]any{"name": "Same", "start": 1400, "end": 1600, "description": "a"})

	repo := newRepository(app)
	first, err := repo.listPeriods()
	if err != nil {
		t.Fatalf("listPeriods: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("listPeriods length = %d, want 2", len(first))
	}
	if first[0].id != "periodtie000001" || first[1].id != "periodtie000002" {
		t.Errorf("listPeriods tie order = (%q, %q), want (periodtie000001, periodtie000002)", first[0].id, first[1].id)
	}

	second, err := repo.listPeriods()
	if err != nil {
		t.Fatalf("second listPeriods: %v", err)
	}
	for i := range first {
		if first[i].id != second[i].id {
			t.Fatalf("listPeriods is not deterministic across calls: %v vs %v", first, second)
		}
	}
}

func TestRepositoryPeriodOnlyWindowHasNoWorks(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "art_periods", "periodonly00001", map[string]any{"name": "Only", "start": 1000, "end": 1200, "description": ""})

	repo := newRepository(app)

	min, max, err := repo.artworkBounds()
	if err != nil {
		t.Fatalf("artworkBounds: %v", err)
	}
	if min != 0 || max != 0 {
		t.Errorf("artworkBounds(period-only) = (%d, %d), want (0, 0)", min, max)
	}

	count, err := repo.countWorks(1000, 1200)
	if err != nil {
		t.Fatalf("countWorks: %v", err)
	}
	if count != 0 {
		t.Errorf("countWorks(period-only) = %d, want 0", count)
	}

	rows, err := repo.listWorks(1000, 1200, worksPageSize, 999*worksPageSize)
	if err != nil {
		t.Fatalf("listWorks: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("listWorks(period-only, huge offset) length = %d, want 0", len(rows))
	}
}

func TestRepositoryEmptyData(t *testing.T) {
	app := newTimelineApp(t)

	repo := newRepository(app)
	min, max, err := repo.artworkBounds()
	if err != nil {
		t.Fatalf("artworkBounds: %v", err)
	}
	if min != 0 || max != 0 {
		t.Errorf("artworkBounds(empty) = (%d, %d), want (0, 0)", min, max)
	}
	count, err := repo.countWorks(1400, 1500)
	if err != nil {
		t.Fatalf("countWorks: %v", err)
	}
	if count != 0 {
		t.Errorf("countWorks(empty) = %d, want 0", count)
	}
	rows, err := repo.listWorks(1400, 1500, 10, 0)
	if err != nil {
		t.Fatalf("listWorks: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("listWorks(empty) length = %d, want 0", len(rows))
	}
	spans, err := repo.dateSpans(1400, 1500)
	if err != nil {
		t.Fatalf("dateSpans: %v", err)
	}
	if len(spans) != 0 {
		t.Errorf("dateSpans(empty) length = %d, want 0", len(spans))
	}
}
