package statistics

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/repositories"
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

func TestSummarizeSchoolPeriodRowsBuildsPeriodMatrix(t *testing.T) {
	rows := summarizeSchoolPeriodRows([]repositories.SchoolPeriodRow{
		{PeriodStart: 1500, School: "Italian", Count: 3},
		{PeriodStart: 1500, School: "French", Count: 2},
		{PeriodStart: 1500, School: "Other", Count: 1},
		{PeriodStart: 1550, School: "Dutch", Count: 4},
	})

	if len(rows) != 2 {
		t.Fatalf("period rows = %d, want 2", len(rows))
	}
	if rows[0].Period != "1500–1549" || rows[0].Italian != 3 || rows[0].French != 2 || rows[0].Other != 1 || rows[0].Total != 6 {
		t.Errorf("first period = %#v", rows[0])
	}
	if rows[1].Period != "1550–1599" || rows[1].Dutch != 4 || rows[1].Total != 4 {
		t.Errorf("second period = %#v", rows[1])
	}
}
