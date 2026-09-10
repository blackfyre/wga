package repositories

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestCollectionHoldingsInvalidationDuringLoadDoesNotRestoreStaleProjection(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	started := make(chan struct{})
	release := make(chan struct{})
	staleResult := make(chan []CollectionHolding, 1)
	go func() {
		value, loadErr := collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
			close(started)
			<-release
			return []CollectionHolding{{Value: "stale", Label: "Stale", Count: 1}}, nil
		})
		if loadErr != nil {
			staleResult <- nil
			return
		}
		staleResult <- value
	}()

	<-started
	InvalidateCollectionHoldings(app)
	fresh := []CollectionHolding{{Value: "fresh", Label: "Fresh", Count: 2}}
	loaded, err := collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
		return fresh, nil
	})
	if err != nil {
		t.Fatalf("load fresh projection: %v", err)
	}
	if len(loaded) != 1 || loaded[0] != fresh[0] {
		t.Fatalf("fresh projection = %#v", loaded)
	}

	close(release)
	if stale := <-staleResult; len(stale) != 1 || stale[0].Value != "stale" {
		t.Fatalf("original caller result = %#v, want its completed stale load", stale)
	}

	cached, err := collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
		t.Fatal("obsolete load replaced the fresh cached projection")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("read cached projection: %v", err)
	}
	if len(cached) != 1 || cached[0] != fresh[0] {
		t.Fatalf("cached projection = %#v, want fresh projection", cached)
	}
}
