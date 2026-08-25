package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newArtistRecordTestApp(t *testing.T) *tests.TestApp {
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
		&core.TextField{Id: "artist_slug", Name: "slug"},
		&core.NumberField{Id: "artist_yob", Name: "year_of_birth"},
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

	composers := core.NewBaseCollection("Music_composer")
	composers.Id = "test_music_composer"
	composers.MarkAsNew()
	composers.Fields.Add(
		&core.TextField{Id: "composer_name", Name: "name", Required: true},
		&core.SelectField{Id: "composer_century", Name: "century", Values: []string{"12", "13", "14", "15", "16", "17", "18", "19", "20", "21"}, MaxSelect: 1},
		&core.BoolField{Id: "composer_published", Name: "published"},
	)
	if err := app.Save(composers); err != nil {
		t.Fatalf("save music_composer collection: %v", err)
	}

	songs := core.NewBaseCollection("Music_song")
	songs.Id = "test_music_song"
	songs.MarkAsNew()
	songs.Fields.Add(
		&core.TextField{Id: "song_title", Name: "title", Required: true},
		&core.RelationField{Id: "song_composer", Name: "composer", CollectionId: composers.Id, MinSelect: 1, MaxSelect: 20},
		&core.FileField{Id: "song_source", Name: "source", MaxSelect: 1},
		&core.BoolField{Id: "song_published", Name: "published"},
	)
	if err := app.Save(songs); err != nil {
		t.Fatalf("save music_song collection: %v", err)
	}

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
		t.Fatalf("save art_periods collection: %v", err)
	}

	return app
}

func saveArtistRecordArtist(t *testing.T, app *tests.TestApp, id string, birth int, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", "Artist "+id)
	record.Set("slug", "artist-"+id)
	record.Set("year_of_birth", birth)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist %s: %v", id, err)
	}
}

func saveArtistRecordWork(t *testing.T, app *tests.TestApp, id string, title string, authors []string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", title)
	record.Set("author", authors)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork %s: %v", id, err)
	}
}

func saveArtistRecordComposer(t *testing.T, app *tests.TestApp, id, name, century string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("music_composer")
	if err != nil {
		t.Fatalf("find music_composer: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("century", century)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save composer %s: %v", id, err)
	}
}

func saveArtistRecordSong(t *testing.T, app *tests.TestApp, id, title string, composers []string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find music_song: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", title)
	record.Set("composer", composers)
	record.Set("source", "song-"+id+".mp3")
	record.Set("published", published)
	if err := app.SaveNoValidate(record); err != nil {
		t.Fatalf("save song %s: %v", id, err)
	}
}

func TestArtistRecordRepositoryFindsPublishedArtistOnly(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true)
	saveArtistRecordArtist(t, app, "artisthid100000", 1606, false)

	repo := NewArtistRecordRepository(app)

	artist, err := repo.FindPublishedArtist("artistpub100000")
	if err != nil {
		t.Fatalf("find published artist: %v", err)
	}
	if artist.Id != "artistpub100000" {
		t.Errorf("artist = %q, want artistpub100000", artist.Id)
	}

	if _, err := repo.FindPublishedArtist("artisthid100000"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("unpublished artist error = %v, want sql.ErrNoRows", err)
	}
	if _, err := repo.FindPublishedArtist("artistmiss00000"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("missing artist error = %v, want sql.ErrNoRows", err)
	}
}

func TestArtistRecordRepositoryCountsAndListsPublishedWorks(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true)
	saveArtistRecordArtist(t, app, "artistother1000", 1606, true)
	saveArtistRecordWork(t, app, "workzeta0000001", "Zeta", []string{"artistpub100000"}, true)
	saveArtistRecordWork(t, app, "workalpha000001", "Alpha", []string{"artistpub100000"}, true)
	saveArtistRecordWork(t, app, "workhidden00001", "Hidden", []string{"artistpub100000"}, false)
	saveArtistRecordWork(t, app, "workother000001", "Other", []string{"artistother1000"}, true)

	repo := NewArtistRecordRepository(app)

	count, err := repo.CountPublishedWorks("artistpub100000")
	if err != nil {
		t.Fatalf("count works: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (unpublished and other-author excluded)", count)
	}

	works, err := repo.ListPublishedWorks("artistpub100000", 0)
	if err != nil {
		t.Fatalf("list works: %v", err)
	}
	if len(works) != 2 {
		t.Fatalf("works = %d, want 2", len(works))
	}
	if works[0].GetString("title") != "Alpha" || works[1].GetString("title") != "Zeta" {
		t.Errorf("works order = [%s, %s], want [Alpha, Zeta]", works[0].GetString("title"), works[1].GetString("title"))
	}
}

func TestArtistRecordRepositoryBoundsWorkPreview(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true)
	for i := 0; i < 20; i++ {
		saveArtistRecordWork(t, app, "workmany"+fmt.Sprintf("%07d", i), fmt.Sprintf("Work %02d", i), []string{"artistpub100000"}, true)
	}

	repo := NewArtistRecordRepository(app)

	count, err := repo.CountPublishedWorks("artistpub100000")
	if err != nil {
		t.Fatalf("count works: %v", err)
	}
	if count != 20 {
		t.Errorf("count = %d, want 20", count)
	}

	works, err := repo.ListPublishedWorks("artistpub100000", 0)
	if err != nil {
		t.Fatalf("list works: %v", err)
	}
	if len(works) != 12 {
		t.Errorf("preview = %d works, want bounded 12", len(works))
	}
}

func TestArtistRecordRepositoryMatchesDeterministicPeriodSong(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true) // 17th century
	saveArtistRecordComposer(t, app, "composer1700000", "Sweelinck", "17", true)
	saveArtistRecordComposer(t, app, "composer1900000", "Chopin", "19", true)
	saveArtistRecordSong(t, app, "songzeta0000000", "Zeta Piece", []string{"composer1700000"}, true)
	saveArtistRecordSong(t, app, "songalpha000000", "Alpha Piece", []string{"composer1700000"}, true)
	saveArtistRecordSong(t, app, "songother000000", "Chopin Piece", []string{"composer1900000"}, true)

	repo := NewArtistRecordRepository(app)

	song, err := repo.MatchPeriodSong(1606)
	if err != nil {
		t.Fatalf("match song: %v", err)
	}
	if song == nil {
		t.Fatal("expected a matching 17th-century song")
	}
	if song.Record.GetString("title") != "Alpha Piece" {
		t.Errorf("song = %q, want deterministic Alpha Piece", song.Record.GetString("title"))
	}
	if song.Composer != "Sweelinck" {
		t.Errorf("composer = %q, want Sweelinck", song.Composer)
	}
}

func TestArtistRecordRepositoryOmitsSongWithoutMatch(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1801, true) // 19th century
	saveArtistRecordComposer(t, app, "composer1700000", "Sweelinck", "17", true)
	saveArtistRecordSong(t, app, "songonly0000000", "Only Piece", []string{"composer1700000"}, true)

	repo := NewArtistRecordRepository(app)

	song, err := repo.MatchPeriodSong(1801)
	if err != nil {
		t.Fatalf("match song: %v", err)
	}
	if song != nil {
		t.Errorf("song = %#v, want nil for unmatched century", song)
	}

	song, err = repo.MatchPeriodSong(0)
	if err != nil {
		t.Fatalf("match song unknown year: %v", err)
	}
	if song != nil {
		t.Errorf("song = %#v, want nil for unknown birth year", song)
	}
}

func TestArtistRecordRepositoryOmitsUnpublishedSong(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true) // 17th century
	saveArtistRecordComposer(t, app, "composer1700000", "Sweelinck", "17", true)
	saveArtistRecordSong(t, app, "songhidden00000", "Hidden Piece", []string{"composer1700000"}, false)

	repo := NewArtistRecordRepository(app)

	song, err := repo.MatchPeriodSong(1606)
	if err != nil {
		t.Fatalf("match song: %v", err)
	}
	if song != nil {
		t.Errorf("song = %#v, want nil for an unpublished song", song)
	}
}

func TestArtistRecordRepositoryOmitsUnpublishedComposer(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true) // 17th century
	saveArtistRecordComposer(t, app, "composerhidden0", "Hidden Composer", "17", false)
	saveArtistRecordSong(t, app, "songpublished00", "Published Piece", []string{"composerhidden0"}, true)

	repo := NewArtistRecordRepository(app)

	song, err := repo.MatchPeriodSong(1606)
	if err != nil {
		t.Fatalf("match song: %v", err)
	}
	if song != nil {
		t.Errorf("song = %#v, want nil for an unpublished composer", song)
	}
}

func TestArtistRecordRepositoryEmptyRelations(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordArtist(t, app, "artistpub100000", 1606, true)

	repo := NewArtistRecordRepository(app)

	count, err := repo.CountPublishedWorks("artistpub100000")
	if err != nil {
		t.Fatalf("count works: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	works, err := repo.ListPublishedWorks("artistpub100000", 0)
	if err != nil {
		t.Fatalf("list works: %v", err)
	}
	if len(works) != 0 {
		t.Errorf("works = %v, want empty", works)
	}

	song, err := repo.MatchPeriodSong(1606)
	if err != nil {
		t.Fatalf("match song: %v", err)
	}
	if song != nil {
		t.Errorf("song = %#v, want nil without music data", song)
	}
}

func saveArtistRecordSchool(t *testing.T, app *tests.TestApp, id, name, slug string) {
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

func saveArtistRecordPeriod(t *testing.T, app *tests.TestApp, id, name string, start, end int) {
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

func countRecordQueries(app *tests.TestApp, fn func() error) (int, error) {
	concurrent, ok := app.ConcurrentDB().(*dbx.DB)
	if !ok {
		return 0, fmt.Errorf("ConcurrentDB is %T, want *dbx.DB", app.ConcurrentDB())
	}
	nonconcurrent, _ := app.NonconcurrentDB().(*dbx.DB)

	var count int64
	queryLog := func(_ context.Context, _ time.Duration, _ string, _ *sql.Rows, _ error) {
		atomic.AddInt64(&count, 1)
	}
	execLog := func(_ context.Context, _ time.Duration, _ string, _ sql.Result, _ error) {
		atomic.AddInt64(&count, 1)
	}

	concurrent.QueryLogFunc = queryLog
	concurrent.ExecLogFunc = execLog
	if nonconcurrent != nil {
		nonconcurrent.QueryLogFunc = queryLog
		nonconcurrent.ExecLogFunc = execLog
	}
	defer func() {
		concurrent.QueryLogFunc = nil
		concurrent.ExecLogFunc = nil
		if nonconcurrent != nil {
			nonconcurrent.QueryLogFunc = nil
			nonconcurrent.ExecLogFunc = nil
		}
	}()

	if err := fn(); err != nil {
		return 0, err
	}

	return int(atomic.LoadInt64(&count)), nil
}

func TestArtistRecordRepositoryListSchoolNamesResolvesInOneQuery(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordSchool(t, app, "schooldutch0001", "Dutch", "dutch")
	saveArtistRecordSchool(t, app, "schoolital10000", "Italian", "italian")
	saveArtistRecordSchool(t, app, "schoolunused000", "Unused", "unused")

	repo := NewArtistRecordRepository(app)

	var names []string
	count, err := countRecordQueries(app, func() error {
		var err error
		names, err = repo.ListSchoolNames([]string{"schoolital10000", "schooldutch0001"})
		return err
	})
	if err != nil {
		t.Fatalf("list school names: %v", err)
	}
	if count != 1 {
		t.Errorf("query count = %d, want 1 (no N+1 school resolution)", count)
	}
	if len(names) != 2 || names[0] != "Italian" || names[1] != "Dutch" {
		t.Errorf("names = %v, want [Italian Dutch] in relation order", names)
	}
}

func TestArtistRecordRepositoryListSchoolNamesSkipsMissingAndEmpty(t *testing.T) {
	app := newArtistRecordTestApp(t)
	saveArtistRecordSchool(t, app, "schooldutch0001", "Dutch", "dutch")

	repo := NewArtistRecordRepository(app)

	names, err := repo.ListSchoolNames([]string{"schoolmissing000", "schooldutch0001"})
	if err != nil {
		t.Fatalf("list school names: %v", err)
	}
	if len(names) != 1 || names[0] != "Dutch" {
		t.Errorf("names = %v, want [Dutch] (missing id skipped)", names)
	}

	empty, err := repo.ListSchoolNames(nil)
	if err != nil {
		t.Fatalf("list school names empty: %v", err)
	}
	if empty != nil {
		t.Errorf("empty names = %v, want nil", empty)
	}
}

func TestArtistRecordRepositoryListMatchingArtPeriodsDetectsAmbiguity(t *testing.T) {
	app := newArtistRecordTestApp(t)
	// "Early" and "Late" overlap 1580-1600 so 1590 is ambiguous.
	saveArtistRecordPeriod(t, app, "periodeearly000", "Early", 1500, 1600)
	saveArtistRecordPeriod(t, app, "periodelate0000", "Late", 1580, 1700)
	saveArtistRecordPeriod(t, app, "periodemodern00", "Modern", 1800, 1900)

	repo := NewArtistRecordRepository(app)

	periods, err := repo.ListMatchingArtPeriods(1550)
	if err != nil {
		t.Fatalf("list matching periods: %v", err)
	}
	if len(periods) != 1 || periods[0].GetString("name") != "Early" {
		t.Errorf("periods = %v, want single Early match", periodNames(periods))
	}

	ambiguous, err := repo.ListMatchingArtPeriods(1590)
	if err != nil {
		t.Fatalf("list ambiguous periods: %v", err)
	}
	if len(ambiguous) != 2 {
		t.Errorf("ambiguous periods = %d rows, want 2 (bounded ambiguity)", len(ambiguous))
	}

	none, err := repo.ListMatchingArtPeriods(2000)
	if err != nil {
		t.Fatalf("list none periods: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("periods = %v, want none for 2000", periodNames(none))
	}

	unknown, err := repo.ListMatchingArtPeriods(0)
	if err != nil {
		t.Fatalf("list unknown-year periods: %v", err)
	}
	if unknown != nil {
		t.Errorf("unknown-year periods = %v, want nil", periodNames(unknown))
	}
}

func periodNames(records []*core.Record) []string {
	names := make([]string, len(records))
	for i, record := range records {
		names[i] = record.GetString("name")
	}
	return names
}
