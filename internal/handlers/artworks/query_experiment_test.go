package artworks

import (
	"fmt"
	"os"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// listArtworkRecords is the pre-task-6.1 baseline retained only for the
// evidence benchmark and equivalence test.
func listArtworkRecords(app *pocketbase.PocketBase, f *filters, limit int, offset int) ([]*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return nil, err
	}
	query := app.RecordQuery(collection)
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)
	if err := attachArtworkConditions(query, resolver, f); err != nil {
		return nil, err
	}
	if err := attachArtworkSort(query, resolver, f); err != nil {
		return nil, err
	}
	if err := resolver.UpdateQuery(query); err != nil {
		return nil, err
	}
	if offset > 0 {
		query.Offset(int64(offset))
	}
	if limit > 0 {
		query.Limit(int64(limit))
	}
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}
	return records, nil
}

func countArtworkRecords(app *pocketbase.PocketBase, f *filters) (int, error) {
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return 0, err
	}
	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	query := app.RecordQuery(collection).Select("COUNT(DISTINCT " + baseID + ")")
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)
	if err := attachArtworkConditions(query, resolver, f); err != nil {
		return 0, err
	}
	if err := resolver.UpdateQuery(query); err != nil {
		return 0, err
	}
	var count int
	if err := query.Row(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// combinedArtworkPage is the benchmark adapter for the retained task 6.1 path.
// It evaluates the filtered, ordered page and total in one window query, then
// hydrates only the bounded result IDs.
func combinedArtworkPage(app *pocketbase.PocketBase, f *filters, limit int, offset int) ([]*core.Record, int, error) {
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return nil, 0, err
	}
	rows, err := listArtworkPageRowsForCollection(app, collection, f, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return nil, 0, nil
	}
	records, err := listArtworkRecordsByPageRowsForCollection(app, collection, rows)
	return records, rows[0].Total, err
}

func TestCombinedArtworkPageMatchesSeparateQueries(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "combineartist01", "Alpha Artist")
	saveSearchArtist(t, app, "combineartist02", "Beta Artist")
	saveSearchLocation(t, app, "combineloc00001", "Alpha Museum")
	saveSearchLocation(t, app, "combineloc00002", "Beta Museum")

	seeds := []searchArtworkSeed{
		{id: "combinework0001", title: "Madonna Alpha", authors: []string{"combineartist01"}, location: "combineloc00001", year: 1500, sourceRow: 1, published: true},
		{id: "combinework0002", title: "Portrait", authors: []string{"combineartist02", "combineartist01"}, location: "combineloc00002", year: 1550, sourceRow: 2, published: true},
		{id: "combinework0003", title: "Madonna Beta", authors: []string{"combineartist02"}, location: "combineloc00001", year: 1650, sourceRow: 3, published: true},
		{id: "combinework0004", title: "Hidden Madonna", authors: []string{"combineartist01"}, location: "combineloc00001", year: 1500, sourceRow: 4, published: false},
	}
	for _, seed := range seeds {
		saveSearchArtwork(t, app, seed)
	}

	for _, tc := range []struct {
		name   string
		filter filters
	}{
		{name: "unfiltered"},
		{name: "text", filter: filters{Query: "Madonna"}},
		{name: "exact artist including coauthor", filter: filters{ArtistID: "combineartist01"}},
		{name: "venue", filter: filters{VenueString: "combineloc00001"}},
		{name: "date range", filter: filters{YearFrom: "1500", YearTo: "1600"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.filter
			f.Sort = sortCatalogue
			f.SortDir = sortAsc
			separateTotal, err := countArtworkRecords(app, &f)
			if err != nil {
				t.Fatalf("count separate records: %v", err)
			}
			separate, err := listArtworkRecords(app, &f, 2, 0)
			if err != nil {
				t.Fatalf("list separate records: %v", err)
			}
			combined, combinedTotal, err := combinedArtworkPage(app, &f, 2, 0)
			if err != nil {
				t.Fatalf("load combined page: %v", err)
			}
			if combinedTotal != separateTotal {
				t.Fatalf("combined total = %d, want %d", combinedTotal, separateTotal)
			}
			if got, want := experimentRecordIDs(combined), experimentRecordIDs(separate); fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("combined IDs = %v, want %v", got, want)
			}
		})
	}
}

func experimentRecordIDs(records []*core.Record) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = record.Id
	}
	return ids
}

func TestCombinedArtworkPageProductionEquivalence(t *testing.T) {
	app, ok := newArtworkExperimentApp(t)
	if !ok {
		t.Skip("set WGA_PERF_DATA_DIR to run production-shaped equivalence")
	}

	for _, tc := range artworkExperimentCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.filter
			f.Sort = sortCatalogue
			f.SortDir = sortAsc
			separateTotal, err := countArtworkRecords(app, &f)
			if err != nil {
				t.Fatalf("count separate records: %v", err)
			}
			separate, err := listArtworkRecords(app, &f, artworkSearchPageSize, 0)
			if err != nil {
				t.Fatalf("list separate records: %v", err)
			}
			combined, combinedTotal, err := combinedArtworkPage(app, &f, artworkSearchPageSize, 0)
			if err != nil {
				t.Fatalf("load combined page: %v", err)
			}
			if combinedTotal != separateTotal || fmt.Sprint(experimentRecordIDs(combined)) != fmt.Sprint(experimentRecordIDs(separate)) {
				t.Fatalf("combined total/IDs = %d/%v, want %d/%v", combinedTotal, experimentRecordIDs(combined), separateTotal, experimentRecordIDs(separate))
			}
		})
	}
}

var artworkExperimentRecords []*core.Record
var artworkExperimentTotal int

func BenchmarkArtworkListCountExperiment(b *testing.B) {
	app, ok := newArtworkExperimentApp(b)
	if !ok {
		b.Skip("set WGA_PERF_DATA_DIR to a production-shaped PocketBase data directory")
	}

	for _, tc := range artworkExperimentCases() {
		f := tc.filter
		f.Sort = sortCatalogue
		f.SortDir = sortAsc
		b.Run(tc.name+"/separate", func(b *testing.B) {
			for range b.N {
				total, err := countArtworkRecords(app, &f)
				if err != nil {
					b.Fatal(err)
				}
				records, err := listArtworkRecords(app, &f, artworkSearchPageSize, 0)
				if err != nil {
					b.Fatal(err)
				}
				artworkExperimentRecords = records
				artworkExperimentTotal = total
			}
		})
		b.Run(tc.name+"/combined", func(b *testing.B) {
			for range b.N {
				records, total, err := combinedArtworkPage(app, &f, artworkSearchPageSize, 0)
				if err != nil {
					b.Fatal(err)
				}
				artworkExperimentRecords = records
				artworkExperimentTotal = total
			}
		})
	}
}

func newArtworkExperimentApp(tb testing.TB) (*pocketbase.PocketBase, bool) {
	tb.Helper()
	dataDir := os.Getenv("WGA_PERF_DATA_DIR")
	if dataDir == "" {
		return nil, false
	}
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	if err := app.Bootstrap(); err != nil {
		tb.Fatalf("bootstrap experiment app: %v", err)
	}
	tb.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			tb.Errorf("reset experiment app: %v", err)
		}
	})
	return app, true
}

func artworkExperimentCases() []struct {
	name   string
	filter filters
} {
	return []struct {
		name   string
		filter filters
	}{
		{name: "unfiltered"},
		{name: "text", filter: filters{Query: "Madonna"}},
		{name: "exact_artist", filter: filters{ArtistID: "r3de9890382970b"}},
		{name: "venue", filter: filters{VenueString: "0f7d9a409f32acf"}},
		{name: "date_range", filter: filters{YearFrom: "1500", YearTo: "1600"}},
	}
}
