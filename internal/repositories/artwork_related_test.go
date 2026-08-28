package repositories

import (
	"fmt"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// relatedID returns a 15-character record id built from a short prefix and a
// zero-padded number, matching PocketBase's record-id length requirement.
func relatedID(prefix string, n int) string {
	return prefix + fmt.Sprintf("%0*d", 15-len(prefix), n)
}

func newArtworkRelatedTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection("Artists")
	artists.Id = constants.CollectionArtists
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "filing_name"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	locations := core.NewBaseCollection("Locations")
	locations.Id = constants.CollectionLocations
	locations.MarkAsNew()
	locations.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.BoolField{Name: "museum"},
		&core.BoolField{Name: "is_public"},
	)
	if err := app.Save(locations); err != nil {
		t.Fatalf("save locations: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = constants.CollectionArtworks
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: constants.CollectionArtists, MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Name: "current_location_id", CollectionId: constants.CollectionLocations, MinSelect: 0, MaxSelect: 1},
		&core.TextField{Name: "art_period_id"},
		&core.BoolField{Name: "published"},
		&core.NumberField{Name: "date_start"},
		&core.JSONField{Name: "colour_signature"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	return app
}

func saveRelatedRecord(t *testing.T, app *tests.TestApp, collection string, id string, fields map[string]any) {
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

func saveRelatedWork(t *testing.T, app *tests.TestApp, id string, title string, authors []string, published bool) {
	t.Helper()
	saveRelatedRecord(t, app, constants.CollectionArtworks, id, map[string]any{
		"title": title, "author": authors, "published": published,
	})
}

func TestParseRelatedWorkBasis(t *testing.T) {
	cases := []struct {
		raw  string
		want RelatedWorkBasis
	}{
		{"artist", RelatedByArtist},
		{"collection", RelatedByCollection},
		{"palette", RelatedByPalette},
		{"period", RelatedByPeriod},
		{"", DefaultRelatedWorkBasis},
		{"unknown", DefaultRelatedWorkBasis},
		{"ARTIST", DefaultRelatedWorkBasis},
		{"related", DefaultRelatedWorkBasis},
	}
	for _, c := range cases {
		if got := ParseRelatedWorkBasis(c.raw); got != c.want {
			t.Errorf("ParseRelatedWorkBasis(%q) = %q, want %q", c.raw, got, c.want)
		}
	}

	if !RelatedByArtist.IsDefault() {
		t.Error("RelatedByArtist should be the default basis")
	}
	if RelatedByCollection.IsDefault() {
		t.Error("RelatedByCollection should not be the default basis")
	}
}

func TestRelatedByArtistReturnsPublishedNonSelfDeterministic(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})
	saveRelatedWork(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 1), "Beta Work", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 2), "Alpha Work", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 3), "Delta Work", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 4), "Hidden Work", []string{"artistone000001"}, false)
	saveRelatedWork(t, app, relatedID("work", 5), "Other Work", []string{"artisttwo000001"}, true)

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")

	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Basis != RelatedByArtist {
		t.Errorf("basis = %q, want %q", got.Basis, RelatedByArtist)
	}
	assertRelatedTitles(t, got.Works, []string{"Alpha Work", "Beta Work", "Delta Work"})
}

func TestRelatedByArtistCapsAtFour(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedWork(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true)
	for i, title := range []string{"A", "B", "C", "D", "E", "F"} {
		saveRelatedWork(t, app, relatedID("work", i+1), title, []string{"artistone000001"}, true)
	}

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != relatedWorksLimit {
		t.Fatalf("works = %d, want capped at %d", len(got.Works), relatedWorksLimit)
	}
}

func TestRelatedByArtistEmptyWhenNoAuthor(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedWork(t, app, "current00000001", "Current Work", []string{}, true)

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != 0 {
		t.Fatalf("works = %d, want 0 (no author)", len(got.Works))
	}
}

func TestRelatedByCollectionReturnsSharedPublicMuseum(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 1), map[string]any{
		"name": "Louvre", "museum": true, "is_public": true,
	})
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 2), map[string]any{
		"name": "Elsewhere Museum", "museum": true, "is_public": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current Work", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Shared A", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 2), map[string]any{
		"title": "Shared B", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 3), map[string]any{
		"title": "Hidden Shared", "author": []string{"artisttwo000001"}, "published": false,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 4), map[string]any{
		"title": "Elsewhere", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 2)},
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Shared A", "Shared B"})
}

func TestRelatedByCollectionEmptyWhenLocationAbsent(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedWork(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true)

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != 0 {
		t.Fatalf("works = %d, want 0 (no current location)", len(got.Works))
	}
}

func TestRelatedByCollectionEmptyWhenPrivate(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 1), map[string]any{
		"name": "Private Collection", "museum": false, "is_public": false,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current Work", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Other Work", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != 0 {
		t.Fatalf("works = %d, want 0 (private collection)", len(got.Works))
	}
}

func TestRelatedByCollectionEmptyWhenPublicNonMuseum(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 1), map[string]any{
		"name": "Public Square", "museum": false, "is_public": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current Work", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Other Work", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != 0 {
		t.Fatalf("works = %d, want 0 (public non-museum location)", len(got.Works))
	}
}

func TestColourSignatureBinCountMatchesProducer(t *testing.T) {
	if colourSignatureBinCount != 148 {
		t.Fatalf("colourSignatureBinCount = %d, want the producer's 148 bins (12*3*4 chromatic + 4 neutral)", colourSignatureBinCount)
	}
}

func TestRelatedBasesExcludeCandidateWithOnlyUnpublishedAuthor(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Hidden", "slug": "hidden", "published": false,
	})
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 1), map[string]any{
		"name": "Louvre", "museum": true, "is_public": true,
	})

	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
		"date_start":          1600,
		"colour_signature":    colourSig(0, 0, 0),
	})
	// Published candidate whose only author is unpublished: shares the current
	// collection, falls within the period window, and carries a valid signature,
	// but must be excluded from every basis.
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Unpublished Author Work", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
		"date_start":          1610,
		"colour_signature":    colourSig(1, 0, 0),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	resolver := NewRelatedWorkResolver(app)

	for _, basis := range []RelatedWorkBasis{RelatedByCollection, RelatedByPalette, RelatedByPeriod} {
		got, err := resolver.Resolve(current, basis)
		if err != nil {
			t.Fatalf("resolve %s: %v", basis, err)
		}
		if len(got.Works) != 0 {
			t.Errorf("%s: works = %d, want 0 (candidate has only unpublished author)", basis, len(got.Works))
		}
	}
}

func TestRelatedBasesKeepCandidateWithMixedAuthors(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Coauthor", "slug": "coauthor", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistthr000001", map[string]any{
		"name": "Unpublished Coauthor", "slug": "unpublished-coauthor", "published": false,
	})
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 1), map[string]any{
		"name": "Louvre", "museum": true, "is_public": true,
	})

	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	// One published and one unpublished author: the published author makes the
	// candidate renderable, so it must not be filtered out.
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Mixed Authors Work", "author": []string{"artisttwo000001", "artistthr000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Mixed Authors Work"})
}

func TestResolveDefaultsInvalidBasis(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedWork(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 1), "Alpha Work", []string{"artistone000001"}, true)

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedWorkBasis("garbage"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Basis != RelatedByArtist {
		t.Errorf("basis = %q, want %q (default)", got.Basis, RelatedByArtist)
	}
	assertRelatedTitles(t, got.Works, []string{"Alpha Work"})
}

// colourSig returns a producer-shaped colour signature in the expected space,
// padded to the full producer bin count so the SQL distance ranking treats it as
// a complete signature.
func colourSig(bins ...int) map[string]any {
	full := make([]int, colourSignatureBinCount)
	copy(full, bins)
	return map[string]any{"space": "oklab-hcl-12x3x4", "bins": full}
}

func saveRelatedWorkFull(t *testing.T, app *tests.TestApp, id string, title string, authors []string, published bool, extra map[string]any) {
	t.Helper()
	fields := map[string]any{"title": title, "author": authors, "published": published}
	for key, value := range extra {
		fields[key] = value
	}
	saveRelatedRecord(t, app, constants.CollectionArtworks, id, fields)
}

func TestRelatedByPaletteDistanceDefinesCandidateSet(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(),
	})

	// Nine candidates whose id order does not match their distance order: the
	// first candidate by id is the farthest, while the remaining eight carry
	// increasing distances 1..8. Distance ranking must exclude work1 (distance
	// 9^2) and keep work9 (distance 8^2), proving the candidate set is
	// distance-ranked rather than id-ordered.
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Farthest First", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(9),
	})
	for i := 2; i <= 9; i++ {
		saveRelatedWorkFull(t, app, relatedID("work", i), fmt.Sprintf("Work %d", i), []string{"artisttwo000001"}, true, map[string]any{
			"colour_signature": colourSig(i - 1),
		})
	}

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	candidates, err := NewRelatedWorkResolver(app).relatedByPalette(current)
	if err != nil {
		t.Fatalf("relatedByPalette: %v", err)
	}
	if len(candidates) != relatedCandidatesLimit {
		t.Fatalf("candidates = %d, want %d", len(candidates), relatedCandidatesLimit)
	}

	ids := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		ids[candidate.Id] = true
	}
	if ids[relatedID("work", 1)] {
		t.Error("farthest candidate (distance 81) must be excluded from the eight-candidate distance-ranked set")
	}
	if !ids[relatedID("work", 9)] {
		t.Error("candidate with distance 64 must be retained in the eight-candidate distance-ranked set")
	}
}

func TestRelatedByPaletteSelectsClosestDateFromDistanceSet(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(),
		"date_start":       1600,
	})

	// Eight candidates within the palette cap, with dates that do not correlate
	// with their colour distance. The closest-date selector must return the four
	// nearest to 1600 regardless of distance.
	saveRelatedWorkFull(t, app, relatedID("work", 1), "At Current", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(8), "date_start": 1600,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 2), "One Year Early", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(1), "date_start": 1599,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 3), "One Year Late", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(7), "date_start": 1601,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 4), "Five Years Early", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(2), "date_start": 1595,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 5), "Ten Years Late", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(3), "date_start": 1610,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 6), "Twenty Years Late", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(4), "date_start": 1620,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 7), "Thirty Years Late", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(5), "date_start": 1630,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 8), "No Date", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(6),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// Closest to 1600: At Current (0), One Year Early (1), One Year Late (1),
	// Five Years Early (5). Ties at distance 1 break to the earlier date first.
	assertRelatedTitles(t, got.Works, []string{"At Current", "One Year Early", "One Year Late", "Five Years Early"})
}

func TestRelatedByPaletteDeterministicTieBreak(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(0, 0, 0),
	})
	// Equal distance; order falls back to title.
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Beta Equal", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Alpha Equal", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Alpha Equal", "Beta Equal"})
}

func TestRelatedByPaletteExcludesSelfAndSameAuthor(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(0, 0, 0),
	})
	// Same author as current: must be excluded even with a near signature.
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Same Author", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})
	// Different author: included.
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Other Author", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(2, 0, 0),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Other Author"})
}

func TestRelatedByPaletteExcludesUnpublished(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(0, 0, 0),
	})
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Hidden Work", []string{"artisttwo000001"}, false, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != 0 {
		t.Fatalf("works = %d, want 0 (unpublished candidate)", len(got.Works))
	}
}

func TestRelatedByPaletteNoResultForMissingOrInvalidSignature(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	// Current without a signature.
	saveRelatedWork(t, app, "current00000001", "No Signature", []string{"artistone000001"}, true)
	// Current with an invalid signature shape.
	saveRelatedWorkFull(t, app, "invalid00000001", "Bad Shape", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": map[string]any{"foo": "bar"},
	})
	// Current with a different space.
	saveRelatedWorkFull(t, app, "wrongsp00000001", "Wrong Space", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": map[string]any{"space": "other-space", "bins": []int{1, 2, 3}},
	})
	// A valid candidate that must never be returned for an invalid current.
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Candidate", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})

	for _, id := range []string{"current00000001", "invalid00000001", "wrongsp00000001"} {
		current := mustFindRecord(t, app, constants.CollectionArtworks, id)
		got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
		if err != nil {
			t.Fatalf("resolve %s: %v", id, err)
		}
		if len(got.Works) != 0 {
			t.Errorf("works for %s = %d, want 0 (invalid/missing signature)", id, len(got.Works))
		}
	}
}

func TestRelatedByPaletteSkipsInvalidCandidateSignature(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(0, 0, 0),
	})
	// Candidate with an invalid signature must be skipped.
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Invalid Candidate", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": map[string]any{"space": "oklab-hcl-12x3x4"},
	})
	// Candidate with a valid signature must be returned.
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Valid Candidate", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Valid Candidate"})
}

func TestRelatedByPeriodReturnsCrossArtistWithinWindow(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"date_start": 1600,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Ten Years", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1610,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Thirty Years", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1630,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 3), "Fifty Years", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1650,
	})
	// Same author, within window: excluded.
	saveRelatedWorkFull(t, app, relatedID("work", 4), "Same Author", []string{"artistone000001"}, true, map[string]any{
		"date_start": 1605,
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Ten Years", "Thirty Years"})
}

func TestRelatedByPeriodNearestDateFirst(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"date_start": 1600,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Twenty Years", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1620,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Ten Years", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1610,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 3), "Thirty Years", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1630,
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Ten Years", "Twenty Years", "Thirty Years"})
}

func TestRelatedByPeriodExactBoundary(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"date_start": 1600,
	})
	// Exactly 40 years: included (inclusive).
	saveRelatedWorkFull(t, app, relatedID("work", 1), "At Boundary", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1640,
	})
	// 41 years: excluded.
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Beyond Boundary", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1641,
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"At Boundary"})
}

func TestRelatedByPeriodNoResultWhenDateUnknown(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWork(t, app, "current00000001", "No Date", []string{"artistone000001"}, true)
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Dated Candidate", []string{"artisttwo000001"}, true, map[string]any{
		"date_start": 1600,
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != 0 {
		t.Fatalf("works = %d, want 0 (unknown current date)", len(got.Works))
	}
}

func TestRelatedByPeriodExcludesUnpublishedAndCapsAtFour(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true,
	})

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"date_start": 1600,
	})
	saveRelatedWorkFull(t, app, relatedID("work", 9), "Hidden Work", []string{"artisttwo000001"}, false, map[string]any{
		"date_start": 1610,
	})
	for i, title := range []string{"A", "B", "C", "D", "E", "F"} {
		saveRelatedWorkFull(t, app, relatedID("work", i+1), title, []string{"artisttwo000001"}, true, map[string]any{
			"date_start": 1610,
		})
	}

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != relatedWorksLimit {
		t.Fatalf("works = %d, want capped at %d", len(got.Works), relatedWorksLimit)
	}
	// Unpublished hidden work must not appear among results.
	for _, work := range got.Works {
		if work.GetString("title") == "Hidden Work" {
			t.Error("unpublished candidate must not appear in results")
		}
	}
}

func TestRelatedByArtistCapsCandidatesAtEight(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true,
	})
	saveRelatedWork(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true)
	for i := 1; i <= 10; i++ {
		saveRelatedWork(t, app, relatedID("work", i), fmt.Sprintf("Work %02d", i), []string{"artistone000001"}, true)
	}

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	resolver := NewRelatedWorkResolver(app)

	candidates, err := resolver.relatedByArtist(current)
	if err != nil {
		t.Fatalf("relatedByArtist: %v", err)
	}
	if len(candidates) != relatedCandidatesLimit {
		t.Fatalf("candidates = %d, want %d", len(candidates), relatedCandidatesLimit)
	}

	got, err := resolver.Resolve(current, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Works) != relatedWorksLimit {
		t.Fatalf("works = %d, want %d", len(got.Works), relatedWorksLimit)
	}
}

func TestSelectClosestDateWorksOrdering(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	artworks, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	makeRecord := func(id, title string, date int) *core.Record {
		record := core.NewRecord(artworks)
		record.Id = id
		record.Set("title", title)
		record.Set("date_start", date)
		return record
	}

	candidates := []*core.Record{
		makeRecord(relatedID("work", 1), "Unknown", 0),
		makeRecord(relatedID("work", 2), "Late", 1610),
		makeRecord(relatedID("work", 3), "Exact", 1600),
		makeRecord(relatedID("work", 4), "Early", 1590),
	}

	got := selectClosestDateWorks(candidates, 1600, 4)
	if len(got) != 4 {
		t.Fatalf("works = %d, want 4", len(got))
	}
	// Known dates first: Exact (distance 0), then the distance-10 tie between
	// Early (1590) and Late (1610) resolved to the earlier date, then unknown.
	assertRelatedTitles(t, got, []string{"Exact", "Early", "Late", "Unknown"})
}

func TestSelectClosestDateWorksUnknownCurrentDate(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	artworks, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	makeRecord := func(id, title string, date int) *core.Record {
		record := core.NewRecord(artworks)
		record.Id = id
		record.Set("title", title)
		record.Set("date_start", date)
		return record
	}

	candidates := []*core.Record{
		makeRecord(relatedID("work", 1), "Later", 1610),
		makeRecord(relatedID("work", 2), "Unknown", 0),
		makeRecord(relatedID("work", 3), "Earlier", 1590),
	}

	got := selectClosestDateWorks(candidates, 0, 4)
	// Unknown current date: known dates ordered by their own date ascending,
	// unknown dates last.
	assertRelatedTitles(t, got, []string{"Earlier", "Later", "Unknown"})
}

func TestRelatedHoldingArtistCountIncludesUnpublishedAnomaly(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true, "filing_name": "Dürer, Albrecht",
	})
	// Unpublished author sharing the primary artist's filing name: counted by the
	// search predicate but excluded from the candidate sample.
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Hidden Dürer", "slug": "hidden-durer", "published": false, "filing_name": "Dürer, Albrecht",
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistthr000001", map[string]any{
		"name": "Other", "slug": "other", "published": true, "filing_name": "Other, Artist",
	})

	saveRelatedWork(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 1), "By Primary", []string{"artistone000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 2), "Unpublished Anomaly", []string{"artisttwo000001"}, true)
	saveRelatedWork(t, app, relatedID("work", 3), "Other Artist", []string{"artistthr000001"}, true)

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got.Holding == nil {
		t.Fatal("holding = nil, want artist holding")
	}
	if got.Holding.QueryKey != relatedHoldingArtist {
		t.Errorf("QueryKey = %q, want %q", got.Holding.QueryKey, relatedHoldingArtist)
	}
	if got.Holding.QueryValue != "Dürer, Albrecht" {
		t.Errorf("QueryValue = %q, want filing name", got.Holding.QueryValue)
	}
	// current + By Primary + Unpublished Anomaly share the filing name.
	if got.Holding.Count != 3 {
		t.Errorf("Count = %d, want 3 (current, by primary, and the unpublished-author anomaly)", got.Holding.Count)
	}

	// The presented sample excludes the unpublished-author anomaly.
	assertRelatedTitles(t, got.Works, []string{"By Primary"})
}

func TestRelatedHoldingCollectionCountAndPrivateNil(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 1), map[string]any{
		"name": "Louvre", "museum": true, "is_public": true,
	})
	saveRelatedRecord(t, app, constants.CollectionLocations, relatedID("loc", 2), map[string]any{
		"name": "Private", "museum": false, "is_public": false,
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true, "filing_name": "Dürer, Albrecht",
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true, "filing_name": "Other, Artist",
	})

	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current", "author": []string{"artistone000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Shared A", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 2), map[string]any{
		"title": "Shared B", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 1)},
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 3), map[string]any{
		"title": "Elsewhere", "author": []string{"artisttwo000001"}, "published": true,
		"current_location_id": []string{relatedID("loc", 2)},
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got.Holding == nil {
		t.Fatal("holding = nil, want venue holding")
	}
	if got.Holding.QueryKey != relatedHoldingVenue {
		t.Errorf("QueryKey = %q, want %q", got.Holding.QueryKey, relatedHoldingVenue)
	}
	if got.Holding.QueryValue != relatedID("loc", 1) {
		t.Errorf("QueryValue = %q, want location id", got.Holding.QueryValue)
	}
	// current + Shared A + Shared B share the location; Elsewhere does not.
	if got.Holding.Count != 3 {
		t.Errorf("Count = %d, want 3", got.Holding.Count)
	}
	assertRelatedTitles(t, got.Works, []string{"Shared A", "Shared B"})

	// A private collection yields no holding and no candidates.
	private := mustFindRecord(t, app, constants.CollectionArtworks, relatedID("work", 3))
	privateGot, err := NewRelatedWorkResolver(app).Resolve(private, RelatedByCollection)
	if err != nil {
		t.Fatalf("resolve private: %v", err)
	}
	if privateGot.Holding != nil {
		t.Errorf("private holding = %+v, want nil", privateGot.Holding)
	}
	if len(privateGot.Works) != 0 {
		t.Errorf("private works = %d, want 0", len(privateGot.Works))
	}
}

func TestRelatedHoldingPeriodCountAndNilForMissing(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true, "filing_name": "Dürer, Albrecht",
	})
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other", "slug": "other", "published": true, "filing_name": "Other, Artist",
	})

	saveRelatedRecord(t, app, constants.CollectionArtworks, "current00000001", map[string]any{
		"title": "Current", "author": []string{"artistone000001"}, "published": true,
		"art_period_id": "period1",
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 1), map[string]any{
		"title": "Same Period", "author": []string{"artisttwo000001"}, "published": true,
		"art_period_id": "period1",
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 2), map[string]any{
		"title": "Other Period", "author": []string{"artisttwo000001"}, "published": true,
		"art_period_id": "period2",
	})
	saveRelatedRecord(t, app, constants.CollectionArtworks, relatedID("work", 3), map[string]any{
		"title": "No Period", "author": []string{"artisttwo000001"}, "published": true,
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got.Holding == nil {
		t.Fatal("holding = nil, want period holding")
	}
	if got.Holding.QueryKey != relatedHoldingPeriod {
		t.Errorf("QueryKey = %q, want %q", got.Holding.QueryKey, relatedHoldingPeriod)
	}
	if got.Holding.QueryValue != "period1" {
		t.Errorf("QueryValue = %q, want period1", got.Holding.QueryValue)
	}
	// current + Same Period share the art period; Other Period does not.
	if got.Holding.Count != 2 {
		t.Errorf("Count = %d, want 2 (current + same period)", got.Holding.Count)
	}
	// The period sample is date-window based; current has no date, so no works.
	if len(got.Works) != 0 {
		t.Errorf("works = %d, want 0 (no date_start on current)", len(got.Works))
	}

	// An artwork without an art period has a nil holding.
	noPeriod := mustFindRecord(t, app, constants.CollectionArtworks, relatedID("work", 3))
	noPeriodGot, err := NewRelatedWorkResolver(app).Resolve(noPeriod, RelatedByPeriod)
	if err != nil {
		t.Fatalf("resolve no-period: %v", err)
	}
	if noPeriodGot.Holding != nil {
		t.Errorf("no-period holding = %+v, want nil", noPeriodGot.Holding)
	}
}

func TestRelatedHoldingNilForPaletteAndUnusable(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	saveRelatedRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Dürer", "slug": "durer", "published": true, "filing_name": "Dürer, Albrecht",
	})

	resolver := NewRelatedWorkResolver(app)

	// Palette holding is always nil, even with a valid signature.
	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(),
	})
	paletteCurrent := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	palette, err := resolver.Resolve(paletteCurrent, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve palette: %v", err)
	}
	if palette.Holding != nil {
		t.Errorf("palette holding = %+v, want nil", palette.Holding)
	}

	// Artist with no author -> nil holding.
	saveRelatedWork(t, app, "noauthor0000001", "No Author", []string{}, true)
	noAuthor := mustFindRecord(t, app, constants.CollectionArtworks, "noauthor0000001")
	noAuthorGot, err := resolver.Resolve(noAuthor, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve no-author: %v", err)
	}
	if noAuthorGot.Holding != nil {
		t.Errorf("no-author holding = %+v, want nil", noAuthorGot.Holding)
	}

	// Artist with an empty filing name -> nil holding.
	saveRelatedRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "No Filing", "slug": "no-filing", "published": true,
	})
	saveRelatedWork(t, app, "emptyfil0000001", "Empty Filing", []string{"artisttwo000001"}, true)
	emptyFiling := mustFindRecord(t, app, constants.CollectionArtworks, "emptyfil0000001")
	emptyFilingGot, err := resolver.Resolve(emptyFiling, RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve empty-filing: %v", err)
	}
	if emptyFilingGot.Holding != nil {
		t.Errorf("empty-filing holding = %+v, want nil", emptyFilingGot.Holding)
	}
}

func mustFindRecord(t *testing.T, app *tests.TestApp, collection string, id string) *core.Record {
	t.Helper()
	record, err := app.FindRecordById(collection, id)
	if err != nil {
		t.Fatalf("find %s %s: %v", collection, id, err)
	}
	return record
}

func assertRelatedTitles(t *testing.T, works []*core.Record, want []string) {
	t.Helper()
	if len(works) != len(want) {
		t.Fatalf("works = %d, want %d (%v)", len(works), len(want), want)
	}
	for i, work := range works {
		if got := work.GetString("title"); got != want[i] {
			t.Errorf("work[%d] = %q, want %q", i, got, want[i])
		}
	}
}
