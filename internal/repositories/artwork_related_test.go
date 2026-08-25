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

func TestRelatedPaletteCandidateLimitCoversProducerCount(t *testing.T) {
	if relatedPaletteCandidateLimit < 52865 {
		t.Fatalf("relatedPaletteCandidateLimit = %d, must cover the documented 52,865 profiled records", relatedPaletteCandidateLimit)
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

// colourSig returns a producer-shaped colour signature in the expected space.
func colourSig(bins ...int) map[string]any {
	return map[string]any{"space": "oklab-hcl-12x3x4", "bins": bins}
}

func saveRelatedWorkFull(t *testing.T, app *tests.TestApp, id string, title string, authors []string, published bool, extra map[string]any) {
	t.Helper()
	fields := map[string]any{"title": title, "author": authors, "published": published}
	for key, value := range extra {
		fields[key] = value
	}
	saveRelatedRecord(t, app, constants.CollectionArtworks, id, fields)
}

func TestRelatedByPaletteRanksBySignatureDistance(t *testing.T) {
	app := newArtworkRelatedTestApp(t)
	for _, artist := range []struct{ id, name string }{
		{"artistone000001", "Dürer"},
		{"artisttwo000001", "Other"},
		{"artistthr000001", "Third"},
	} {
		saveRelatedRecord(t, app, constants.CollectionArtists, artist.id, map[string]any{
			"name": artist.name, "slug": artist.name, "published": true,
		})
	}

	saveRelatedWorkFull(t, app, "current00000001", "Current Work", []string{"artistone000001"}, true, map[string]any{
		"colour_signature": colourSig(0, 0, 0),
	})
	saveRelatedWorkFull(t, app, relatedID("work", 1), "Near Work", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(1, 0, 0),
	})
	saveRelatedWorkFull(t, app, relatedID("work", 2), "Far Work", []string{"artistthr000001"}, true, map[string]any{
		"colour_signature": colourSig(2, 0, 0),
	})
	saveRelatedWorkFull(t, app, relatedID("work", 3), "Farthest Work", []string{"artisttwo000001"}, true, map[string]any{
		"colour_signature": colourSig(0, 3, 0),
	})

	current := mustFindRecord(t, app, constants.CollectionArtworks, "current00000001")
	got, err := NewRelatedWorkResolver(app).Resolve(current, RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	assertRelatedTitles(t, got.Works, []string{"Near Work", "Far Work", "Farthest Work"})
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
