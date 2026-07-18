package statistics

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
)

func TestMarshalStatsCoalescesConcurrentFetches(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	const workers = 8

	var fetches atomic.Int32
	ready := make(chan struct{}, workers)
	start := make(chan struct{})
	results := make(chan error, workers)

	for range workers {
		go func() {
			ready <- struct{}{}
			<-start

			rows, jsonData, err := marshalStats(app, t.Name(), func() ([]int, error) {
				fetches.Add(1)
				time.Sleep(20 * time.Millisecond)
				return []int{1, 2, 3}, nil
			})
			if err != nil {
				results <- err
				return
			}
			if len(rows) != 3 || jsonData != "[1,2,3]" {
				results <- fmt.Errorf("unexpected cached statistics: rows=%v json=%s", rows, jsonData)
				return
			}

			results <- nil
		}()
	}

	for range workers {
		<-ready
	}
	close(start)

	for range workers {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}

	if got := fetches.Load(); got != 1 {
		t.Errorf("expected one aggregate fetch, got %d", got)
	}
}
