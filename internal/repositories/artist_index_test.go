package repositories

import (
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newArtistIndexTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	schools := core.NewBaseCollection("Schools")
	schools.Id = "test_schools"
	schools.MarkAsNew()
	schools.Fields.Add(
		&core.TextField{Id: "school_name", Name: "name", Required: true},
		&core.TextField{Id: "school_slug", Name: "slug"},
	)
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools collection: %v", err)
	}

	periods := core.NewBaseCollection("Art_periods")
	periods.Id = "test_art_periods"
	periods.MarkAsNew()
	periods.Fields.Add(
		&core.TextField{Id: "period_name", Name: "name", Required: true},
		&core.NumberField{Id: "period_start", Name: "start"},
		&core.NumberField{Id: "period_end", Name: "end"},
	)
	if err := app.Save(periods); err != nil {
		t.Fatalf("save art periods collection: %v", err)
	}

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_filing_name", Name: "filing_name"},
		&core.TextField{Id: "artist_short_name", Name: "short_name"},
		&core.NumberField{Id: "artist_yob", Name: "year_of_birth"},
		&core.NumberField{Id: "artist_yod", Name: "year_of_death"},
		&core.TextField{Id: "artist_profession", Name: "profession"},
		&core.TextField{Id: "artist_portrait", Name: "portrait"},
		&core.NumberField{Id: "artist_portrait_width", Name: "biography_image_width"},
		&core.RelationField{Id: "artist_school", Name: "school", CollectionId: schools.Id, MinSelect: 0, MaxSelect: 10},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "test_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Id: "artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	return app
}

func saveArtistIndexSchool(t *testing.T, app *tests.TestApp, id, slug, name string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("schools")
	if err != nil {
		t.Fatalf("find schools: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("slug", slug)
	if err := app.Save(record); err != nil {
		t.Fatalf("save school %s: %v", id, err)
	}
}

func saveArtistIndexPeriod(t *testing.T, app *tests.TestApp, id, name string, start, end int) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("art_periods")
	if err != nil {
		t.Fatalf("find art_periods: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("start", start)
	record.Set("end", end)
	if err := app.Save(record); err != nil {
		t.Fatalf("save period %s: %v", id, err)
	}
}

type artistIndexArtistSeed struct {
	id          string
	name        string
	filingName  string
	shortName   string
	blankFiling bool
	blankShort  bool
	birth       int
	death       int
	profession  string
	portrait    string
	schools     []string
	published   bool
}

func saveArtistIndexArtist(t *testing.T, app *tests.TestApp, seed artistIndexArtistSeed) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = seed.id
	record.Set("name", seed.name)
	filingName := seed.filingName
	if filingName == "" && !seed.blankFiling {
		filingName = seed.name
	}
	shortName := seed.shortName
	if shortName == "" && !seed.blankShort {
		shortName = seed.name
	}
	record.Set("filing_name", filingName)
	record.Set("short_name", shortName)
	record.Set("year_of_birth", seed.birth)
	record.Set("year_of_death", seed.death)
	record.Set("profession", seed.profession)
	if seed.portrait != "" {
		record.Set("portrait", seed.portrait)
	}
	record.Set("school", seed.schools)
	record.Set("published", seed.published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist %s: %v", seed.id, err)
	}
}

func saveArtistIndexArtwork(t *testing.T, app *tests.TestApp, id string, authors []string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", "Work "+id)
	record.Set("author", authors)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork %s: %v", id, err)
	}
}

// newArtistIndexLegacyTestApp builds a pre-migration app whose artists
// collection lacks one or both authoritative identity fields, mirroring the
// malformed instances the fail-closed guard must tolerate.
func newArtistIndexLegacyTestApp(t *testing.T, withFilingName bool) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.NumberField{Id: "artist_yob", Name: "year_of_birth"},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if withFilingName {
		artists.Fields.Add(&core.TextField{Id: "artist_filing_name", Name: "filing_name"})
	}
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	return app
}

// saveLegacyArtistIndexArtist persists a published artist using only the fields
// present in the legacy (identity-less) artists schema.
func saveLegacyArtistIndexArtist(t *testing.T, app *tests.TestApp, id, name string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("year_of_birth", 1606)
	record.Set("published", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("save legacy artist %s: %v", id, err)
	}
}

func TestArtistIndexRepositoryFiltersUnpublishedArtists(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artistpub100000", name: "Public Artist", birth: 1500, death: 1560, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artisthid100000", name: "Hidden Artist", birth: 1500, death: 1560, published: false})

	repo := NewArtistIndexRepository(app)

	count, err := repo.CountArtists(ArtistIndexFilter{Limit: 100})
	if err != nil {
		t.Fatalf("count artists: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (unpublished excluded)", count)
	}

	artists, err := repo.ListArtists(ArtistIndexFilter{Limit: 100})
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}
	if len(artists) != 1 || artists[0].Record.Id != "artistpub100000" {
		t.Errorf("list = %#v, want only published artist", artists)
	}

	letters, err := repo.ListAvailableLetters(ArtistIndexFilter{})
	if err != nil {
		t.Fatalf("list letters: %v", err)
	}
	if len(letters) != 1 || letters[0] != "P" {
		t.Errorf("letters = %v, want [P]", letters)
	}

	minYear, maxYear, err := repo.BirthYearBounds()
	if err != nil {
		t.Fatalf("birth year bounds: %v", err)
	}
	if minYear != 1500 || maxYear != 1500 {
		t.Errorf("bounds = (%d, %d), want (1500, 1500)", minYear, maxYear)
	}
}

func TestArtistIndexRepositoryDerivesAvailability(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artavail1000000", name: "Available Artist", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artunavail10000", name: "Unavailable Artist", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artsecond100000", name: "Second Position Artist", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artother1000000", name: "Co-author Artist", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artempty1000000", name: "No Work Artist", published: true})

	// Direct published author relation confers availability.
	saveArtistIndexArtwork(t, app, "workdirect10000", []string{"artavail1000000"}, true)
	// A published artwork confers availability on every artist in its author
	// array, regardless of position.
	saveArtistIndexArtwork(t, app, "worksecond10000", []string{"artother1000000", "artsecond100000"}, true)
	// Unpublished artwork never confers availability.
	saveArtistIndexArtwork(t, app, "workhidden10000", []string{"artunavail10000"}, false)

	repo := NewArtistIndexRepository(app)
	artists, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameAsc, Limit: 100})
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}

	available := map[string]bool{}
	for _, artist := range artists {
		available[artist.Record.Id] = artist.Available
	}
	if !available["artavail1000000"] {
		t.Errorf("artavail1000000 should be available via direct author relation")
	}
	if !available["artsecond100000"] {
		t.Errorf("artsecond100000 should be available via second-position author relation")
	}
	if !available["artother1000000"] {
		t.Errorf("artother1000000 should be available via first-position author relation")
	}
	if available["artunavail10000"] {
		t.Errorf("artunavail10000 should be unavailable (only unpublished artworks)")
	}
	if available["artempty1000000"] {
		t.Errorf("artempty1000000 should be unavailable (no artwork at all)")
	}
}

func TestArtistIndexRepositoryListsAvailableLetters(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artalice1000000", name: "Alice", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artbob100000000", name: "Bob", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artcarol1000000", name: "Carol", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artalice2000000", name: "Alice Two", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artvan100000000", name: "van Gogh", published: true})

	repo := NewArtistIndexRepository(app)
	letters, err := repo.ListAvailableLetters(ArtistIndexFilter{})
	if err != nil {
		t.Fatalf("list letters: %v", err)
	}

	want := []string{"A", "B", "C", "V"}
	if len(letters) != len(want) {
		t.Fatalf("letters = %v, want %v", letters, want)
	}
	for i := range want {
		if letters[i] != want[i] {
			t.Errorf("letters[%d] = %q, want %q", i, letters[i], want[i])
		}
	}
}

func TestArtistIndexRepositoryUsesAuthoritativeFilingName(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artistalpha0001", name: "Zulu Route Name", filingName: "Alpha, Filing", shortName: "Alpha", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artistzulu00001", name: "Alpha Route Name", filingName: "Zulu, Filing", shortName: "Zulu", published: true})

	repo := NewArtistIndexRepository(app)
	artists, err := repo.ListArtists(ArtistIndexFilter{Query: "alpha", Letter: "A", Sort: ArtistSortNameAsc, Limit: 100})
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}
	if got := names(artists); len(got) != 1 || got[0] != "Alpha, Filing" {
		t.Fatalf("filing-name filter result = %v, want [Alpha, Filing]", got)
	}

	all, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameAsc, Limit: 100})
	if err != nil {
		t.Fatalf("list all artists: %v", err)
	}
	if got := names(all); len(got) != 2 || got[0] != "Alpha, Filing" || got[1] != "Zulu, Filing" {
		t.Fatalf("filing-name sort = %v", got)
	}

	letters, err := repo.ListAvailableLetters(ArtistIndexFilter{})
	if err != nil {
		t.Fatalf("list letters: %v", err)
	}
	if len(letters) != 2 || letters[0] != "A" || letters[1] != "Z" {
		t.Fatalf("filing-name letters = %v, want [A Z]", letters)
	}
}

func TestArtistIndexRepositoryFoldsAccentedFilingInitials(t *testing.T) {
	app := newArtistIndexTestApp(t)
	// The four real non-ASCII producer filing initials: three Á and one Ś.
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artaccenta00001", name: "Legacy Álvares", filingName: "ÁLVARES, Baltasar", shortName: "Baltasar Álvares", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artaccenta00002", name: "Legacy Ángel", filingName: "ÁNGEL, Pedro", shortName: "Pedro Ángel", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artaccenta00003", name: "Legacy Cubero", filingName: "ÁLVAREZ Y CUBERO, José", shortName: "José Álvarez Y Cubero", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artaccents00001", name: "Legacy Ślewiński", filingName: "ŚLEWIŃSKI, Władisław", shortName: "Władisław Ślewiński", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artplain0000001", name: "Alice", filingName: "Alice", shortName: "Alice", published: true})

	repo := NewArtistIndexRepository(app)

	letters, err := repo.ListAvailableLetters(ArtistIndexFilter{})
	if err != nil {
		t.Fatalf("list letters: %v", err)
	}
	for _, want := range []string{"A", "S"} {
		if !containsString(letters, want) {
			t.Errorf("letters = %v, want to include %q", letters, want)
		}
	}
	for _, absent := range []string{"Á", "Ś"} {
		if containsString(letters, absent) {
			t.Errorf("letters = %v, must not leak non-ASCII initial %q", letters, absent)
		}
	}

	// The letter filter reaches every accented record and renders it verbatim.
	aArtists, err := repo.ListArtists(ArtistIndexFilter{Letter: "A", Limit: 100})
	if err != nil {
		t.Fatalf("list A artists: %v", err)
	}
	if got := names(aArtists); len(got) != 4 {
		t.Fatalf("A artists = %v, want the three accented filings plus Alice", got)
	} else if got[0] != "Alice" {
		t.Errorf("A artists first = %q, want plain Alice first under A", got[0])
	}
	joined := strings.Join(names(aArtists), "|")
	for _, want := range []string{"ÁLVARES, Baltasar", "ÁNGEL, Pedro", "ÁLVAREZ Y CUBERO, José"} {
		if !strings.Contains(joined, want) {
			t.Errorf("A artists %q missing verbatim filing %q", joined, want)
		}
	}

	sArtists, err := repo.ListArtists(ArtistIndexFilter{Letter: "S", Limit: 100})
	if err != nil {
		t.Fatalf("list S artists: %v", err)
	}
	if got := names(sArtists); len(got) != 1 || got[0] != "ŚLEWIŃSKI, Władisław" {
		t.Errorf("S artists = %v, want [ŚLEWIŃSKI, Władisław] verbatim", got)
	}

	// Count and list stay consistent per letter.
	for _, letter := range []string{"A", "S"} {
		count, err := repo.CountArtists(ArtistIndexFilter{Letter: letter})
		if err != nil {
			t.Fatalf("count %s artists: %v", letter, err)
		}
		listed, err := repo.ListArtists(ArtistIndexFilter{Letter: letter, Limit: 100})
		if err != nil {
			t.Fatalf("list %s artists: %v", letter, err)
		}
		if count != len(listed) {
			t.Errorf("letter %s count = %d, list = %d, want consistent", letter, count, len(listed))
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestFilingLetterExpressionBindsSelectedLetter(t *testing.T) {
	params := dbx.Params{}
	sql := filingLetterExpression("A").Build(nil, params)

	if !strings.Contains(sql, "filing_name LIKE {:letter}") {
		t.Errorf("sql = %q, want a bound {:letter} placeholder for the selected letter", sql)
	}
	if strings.Contains(sql, "LIKE 'A%'") {
		t.Errorf("sql = %q, must not interpolate the selected letter as a literal", sql)
	}
	if got := params["letter"]; got != "A%" {
		t.Errorf("bound letter param = %v, want %q", got, "A%")
	}
	// The fixed producer fold initial stays an embedded safe literal.
	if !strings.Contains(sql, "'Á%'") {
		t.Errorf("sql = %q, want the fixed Á fold literal", sql)
	}
}

func TestArtistIndexRepositoryLetterCannotAlterQuery(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artalice0000000", name: "Alice", published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artbob000000000", name: "Bob", published: true})

	repo := NewArtistIndexRepository(app)

	// A hostile letter that, were it interpolated, would broaden the predicate
	// to every published row must instead be bound as a literal LIKE prefix and
	// match nothing.
	hostile := "X' OR published = true OR filing_name LIKE 'X"
	artists, err := repo.ListArtists(ArtistIndexFilter{Letter: hostile, Limit: 100})
	if err != nil {
		t.Fatalf("list artists with hostile letter: %v", err)
	}
	if len(artists) != 0 {
		t.Fatalf("hostile letter matched %d artists = %v, want 0 (letter must be bound)", len(artists), artistIDs(artists))
	}
}

func TestArtistIndexRepositoryExcludesBlankIdentity(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artcomplete0000", name: "Complete Legacy", filingName: "Complete, Filing", shortName: "Complete", birth: 1500, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artblankfil0000", name: "Blank Filing Legacy", blankFiling: true, shortName: "Blank Filing", birth: 1400, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artblanksho0000", name: "Blank Short Legacy", filingName: "Blank, Short Filing", blankShort: true, birth: 1600, published: true})

	repo := NewArtistIndexRepository(app)

	count, err := repo.CountArtists(ArtistIndexFilter{Limit: 100})
	if err != nil {
		t.Fatalf("count artists: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (blank identity excluded)", count)
	}

	artists, err := repo.ListArtists(ArtistIndexFilter{Limit: 100})
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}
	if got := artistIDs(artists); len(got) != 1 || got[0] != "artcomplete0000" {
		t.Fatalf("list = %v, want only [artcomplete0000] (blank identity excluded)", got)
	}

	letters, err := repo.ListAvailableLetters(ArtistIndexFilter{})
	if err != nil {
		t.Fatalf("list letters: %v", err)
	}
	if len(letters) != 1 || letters[0] != "C" {
		t.Fatalf("letters = %v, want [C] (blank identity excluded)", letters)
	}

	minYear, maxYear, err := repo.BirthYearBounds()
	if err != nil {
		t.Fatalf("birth year bounds: %v", err)
	}
	if minYear != 1500 || maxYear != 1500 {
		t.Errorf("bounds = (%d, %d), want (1500, 1500) (blank identity excluded)", minYear, maxYear)
	}
}

func TestArtistIndexRepositoryFailsClosedWithoutIdentityFields(t *testing.T) {
	for _, tc := range []struct {
		name           string
		withFilingName bool
	}{
		{name: "neither identity field", withFilingName: false},
		{name: "missing short_name only", withFilingName: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newArtistIndexLegacyTestApp(t, tc.withFilingName)
			saveLegacyArtistIndexArtist(t, app, "artlegacy000000", "Legacy Artist")

			repo := NewArtistIndexRepository(app)

			count, err := repo.CountArtists(ArtistIndexFilter{})
			if err != nil {
				t.Fatalf("count artists: %v", err)
			}
			if count != 0 {
				t.Errorf("count = %d, want 0 (missing identity fields)", count)
			}

			artists, err := repo.ListArtists(ArtistIndexFilter{Limit: 100})
			if err != nil {
				t.Fatalf("list artists: %v", err)
			}
			if len(artists) != 0 {
				t.Errorf("artists = %#v, want empty (missing identity fields)", artists)
			}

			letters, err := repo.ListAvailableLetters(ArtistIndexFilter{})
			if err != nil {
				t.Fatalf("list letters: %v", err)
			}
			if len(letters) != 0 {
				t.Errorf("letters = %v, want empty (missing identity fields)", letters)
			}

			minYear, maxYear, err := repo.BirthYearBounds()
			if err != nil {
				t.Fatalf("birth year bounds: %v", err)
			}
			if minYear != 0 || maxYear != 0 {
				t.Errorf("bounds = (%d, %d), want (0, 0) (missing identity fields)", minYear, maxYear)
			}
		})
	}
}

func TestArtistIndexRepositoryCombinesFilters(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexSchool(t, app, "schooldutch1000", "dutch", "Dutch")
	saveArtistIndexSchool(t, app, "schoolital10000", "italian", "Italian")
	saveArtistIndexPeriod(t, app, "periodbaroque10", "Baroque", 1600, 1750)

	// In range, Dutch, name matches.
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artdutch1000000", name: "Rembrandt van Rijn", birth: 1606, death: 1669, schools: []string{"schooldutch1000"}, published: true})
	// In range, Dutch, name does not match query.
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artdutch2000000", name: "Johannes Vermeer", birth: 1632, death: 1675, schools: []string{"schooldutch1000"}, published: true})
	// Out of range, Dutch.
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artdutch3000000", name: "Rembrandt Early", birth: 1580, death: 1650, schools: []string{"schooldutch1000"}, published: true})
	// In range, Italian.
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artital10000000", name: "Rembrandt Italiano", birth: 1650, death: 1700, schools: []string{"schoolital10000"}, published: true})

	repo := NewArtistIndexRepository(app)
	filter := ArtistIndexFilter{
		Query:        "rembrandt",
		School:       "dutch",
		PeriodActive: true,
		PeriodStart:  1600,
		PeriodEnd:    1750,
		BornActive:   true,
		BornFrom:     1600,
		BornTo:       1700,
		Limit:        100,
	}

	artists, err := repo.ListArtists(filter)
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}
	if len(artists) != 1 || artists[0].Record.Id != "artdutch1000000" {
		t.Fatalf("combined filter artists = %v, want only artdutch1000000", artistIDs(artists))
	}

	count, err := repo.CountArtists(filter)
	if err != nil {
		t.Fatalf("count artists: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestArtistIndexRepositorySortsDeterministically(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artbirthmid1000", name: "Middle", birth: 1600, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artbirthlow1000", name: "Earliest", birth: 1500, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artbirthhigh100", name: "Latest", birth: 1700, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artbirthnone100", name: "Unknown Year", birth: 0, published: true})

	repo := NewArtistIndexRepository(app)

	asc, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameAsc, Limit: 100})
	if err != nil {
		t.Fatalf("list az: %v", err)
	}
	if names(asc)[0] != "Earliest" || names(asc)[3] != "Unknown Year" {
		t.Errorf("az order = %v", names(asc))
	}

	desc, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameDesc, Limit: 100})
	if err != nil {
		t.Fatalf("list za: %v", err)
	}
	if names(desc)[0] != "Unknown Year" || names(desc)[3] != "Earliest" {
		t.Errorf("za order = %v", names(desc))
	}

	birth, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortBirth, Limit: 100})
	if err != nil {
		t.Fatalf("list birth: %v", err)
	}
	got := names(birth)
	want := []string{"Earliest", "Middle", "Latest", "Unknown Year"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("birth order[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestArtistIndexRepositoryPaginatesDeterministically(t *testing.T) {
	app := newArtistIndexTestApp(t)
	for i, name := range []string{"Abe", "Bea", "Cyd", "Dee", "Eve", "Fay"} {
		saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artpage0000000" + string(rune(0x30+i)), name: name, birth: 1500 + i, published: true})
	}

	repo := NewArtistIndexRepository(app)

	page1, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameAsc, Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	page2, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameAsc, Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	page3, err := repo.ListArtists(ArtistIndexFilter{Sort: ArtistSortNameAsc, Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}

	if len(page1) != 2 || names(page1)[0] != "Abe" || names(page1)[1] != "Bea" {
		t.Errorf("page1 = %v", names(page1))
	}
	if len(page2) != 2 || names(page2)[0] != "Cyd" || names(page2)[1] != "Dee" {
		t.Errorf("page2 = %v", names(page2))
	}
	if len(page3) != 2 || names(page3)[0] != "Eve" || names(page3)[1] != "Fay" {
		t.Errorf("page3 = %v", names(page3))
	}
}

func TestArtistIndexRepositoryBirthYearBoundsIgnoreUnknown(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artboundlow1000", name: "Low", birth: 1200, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artboundhigh100", name: "High", birth: 1900, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artboundnone100", name: "None", birth: 0, published: true})

	repo := NewArtistIndexRepository(app)
	minYear, maxYear, err := repo.BirthYearBounds()
	if err != nil {
		t.Fatalf("bounds: %v", err)
	}
	if minYear != 1200 || maxYear != 1900 {
		t.Errorf("bounds = (%d, %d), want (1200, 1900)", minYear, maxYear)
	}
}

func TestArtistIndexRepositoryEmptyCollection(t *testing.T) {
	app := newArtistIndexTestApp(t)
	repo := NewArtistIndexRepository(app)

	count, err := repo.CountArtists(ArtistIndexFilter{})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	artists, err := repo.ListArtists(ArtistIndexFilter{Limit: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(artists) != 0 {
		t.Errorf("artists = %#v, want empty", artists)
	}

	minYear, maxYear, err := repo.BirthYearBounds()
	if err != nil {
		t.Fatalf("bounds: %v", err)
	}
	if minYear != 0 || maxYear != 0 {
		t.Errorf("bounds = (%d, %d), want (0, 0)", minYear, maxYear)
	}
}

func TestArtistIndexRepositoryEmptyPageSkipsAvailability(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artsingle100000", name: "Solo", published: true})

	repo := NewArtistIndexRepository(app)
	artists, err := repo.ListArtists(ArtistIndexFilter{Query: "does-not-match-anything", Limit: 100})
	if err != nil {
		t.Fatalf("list empty page: %v", err)
	}
	if len(artists) != 0 {
		t.Errorf("artists = %#v, want empty", artists)
	}
}

func artistIDs(artists []IndexedArtist) []string {
	ids := make([]string, len(artists))
	for i, artist := range artists {
		ids[i] = artist.Record.Id
	}
	return ids
}

func names(artists []IndexedArtist) []string {
	values := make([]string, len(artists))
	for i, artist := range artists {
		values[i] = artist.Record.GetString("filing_name")
	}
	return values
}
