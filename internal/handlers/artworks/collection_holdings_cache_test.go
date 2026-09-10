package artworks

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

func TestCollectionHoldingsColdLoadIsSharedAcrossVenueQueries(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "cacheartist0001", "Cache Artist")
	saveSearchLocation(t, app, "cachelocation01", "Cache Museum")
	saveSearchArtwork(t, app, searchArtworkSeed{
		id:        "cacheartwork001",
		title:     "Cached Work",
		authors:   []string{"cacheartist0001"},
		location:  "cachelocation01",
		published: true,
	})

	queryCount, err := countCollectionHoldingQueries(app, func() error {
		const callers = 16
		start := make(chan struct{})
		errs := make(chan error, callers)
		var group sync.WaitGroup
		group.Add(callers)
		for i := 0; i < callers; i++ {
			go func(index int) {
				defer group.Done()
				<-start
				view, _, loadErr := buildArtworkSearchView(app, url.Values{
					"venue_q": {fmt.Sprintf("museum-%d", index)},
				}, 1, 16)
				if loadErr != nil {
					errs <- loadErr
					return
				}
				if len(view.Facets.Collection.Options) != 0 {
					errs <- fmt.Errorf("query %d options = %#v, want no name match", index, view.Facets.Collection.Options)
				}
			}(i)
		}
		close(start)
		group.Wait()
		close(errs)
		for loadErr := range errs {
			if loadErr != nil {
				return loadErr
			}
		}

		for i := 0; i < 100; i++ {
			if _, loadErr := getVenueOptions(app, fmt.Sprintf("subsequent-%d", i), ""); loadErr != nil {
				return loadErr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("load venue options: %v", err)
	}
	if queryCount != 1 {
		t.Fatalf("collection projection query count = %d, want one shared load across arbitrary venue queries", queryCount)
	}
}

func countCollectionHoldingQueries(app *pocketbase.PocketBase, fn func() error) (int, error) {
	concurrent, ok := app.ConcurrentDB().(*dbx.DB)
	if !ok {
		return 0, fmt.Errorf("ConcurrentDB is %T, want *dbx.DB", app.ConcurrentDB())
	}
	nonconcurrent, _ := app.NonconcurrentDB().(*dbx.DB)

	var count int64
	queryLog := func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
		if strings.Contains(query, "COUNT(DISTINCT artworks.id) AS holding_count") {
			atomic.AddInt64(&count, 1)
		}
	}
	concurrent.QueryLogFunc = queryLog
	if nonconcurrent != nil {
		nonconcurrent.QueryLogFunc = queryLog
	}
	defer func() {
		concurrent.QueryLogFunc = nil
		if nonconcurrent != nil {
			nonconcurrent.QueryLogFunc = nil
		}
	}()

	if err := fn(); err != nil {
		return 0, err
	}
	return int(atomic.LoadInt64(&count)), nil
}
