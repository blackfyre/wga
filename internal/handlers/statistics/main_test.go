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

func TestSummarizeArtFormRowsPreservesNameAndCount(t *testing.T) {
	rows := summarizeArtFormRows([]repositories.ArtFormDistribution{
		{Name: "Painting", Count: 7},
		{Name: "Sculpture", Count: 3},
	})

	if len(rows) != 2 {
		t.Fatalf("art form rows = %d, want 2", len(rows))
	}
	if rows[0].Name != "Painting" || rows[0].Count != 7 {
		t.Errorf("first art form = %#v", rows[0])
	}
	if rows[1].Name != "Sculpture" || rows[1].Count != 3 {
		t.Errorf("second art form = %#v", rows[1])
	}
}

func TestSummarizeSchoolPeriodRowsFormatsPeriodLabels(t *testing.T) {
	rows := summarizeSchoolPeriodRows([]repositories.SchoolPeriodRow{
		{PeriodStart: 1500, School: "Italian", Count: 3},
		{PeriodStart: 1500, School: "French", Count: 2},
		{PeriodStart: 1500, School: "Other", Count: 1},
		{PeriodStart: 1550, School: "Dutch", Count: 4},
	})

	if len(rows) != 4 {
		t.Fatalf("period rows = %d, want 4", len(rows))
	}

	want := []struct {
		period string
		school string
		count  int
	}{
		{"1500–1549", "Italian", 3},
		{"1500–1549", "French", 2},
		{"1500–1549", "Other", 1},
		{"1550–1599", "Dutch", 4},
	}
	for i, row := range rows {
		if row.Period != want[i].period || row.School != want[i].school || row.Count != want[i].count {
			t.Errorf("row %d = %#v, want %#v", i, row, want[i])
		}
	}
}
