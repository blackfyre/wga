package repositories

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/pocketbase/pocketbase"
)

func TestDualModeReferenceRepositoryCachesAndIsolatesResults(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	var loads atomic.Int32
	repo := &DualModeReferenceRepository{
		app: app,
		ctx: t.Context(),
		load: func(context.Context) (DualModeReference, error) {
			loads.Add(1)
			return DualModeReference{
				Schools: []ArtistSchoolOption{{ID: "school", Name: "Original"}},
				Periods: []ArtistPeriodOption{{ID: "period", Name: "Original"}},
				BornMin: 1400,
				BornMax: 1900,
			}, nil
		},
	}

	first, err := repo.Load()
	if err != nil {
		t.Fatalf("cold load: %v", err)
	}
	first.Schools[0].Name = "Mutated"
	first.Periods[0].Name = "Mutated"
	second, err := repo.Load()
	if err != nil {
		t.Fatalf("cache hit: %v", err)
	}
	if loads.Load() != 1 {
		t.Fatalf("loads = %d, want 1", loads.Load())
	}
	if second.Schools[0].Name != "Original" || second.Periods[0].Name != "Original" {
		t.Fatalf("cached projection was mutated: %#v", second)
	}
}

func TestDualModeReferenceRepositorySharesColdLoad(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
		started := make(chan struct{})
		release := make(chan struct{})
		var loads atomic.Int32
		repo := &DualModeReferenceRepository{
			app: app,
			ctx: t.Context(),
			load: func(context.Context) (DualModeReference, error) {
				if loads.Add(1) == 1 {
					close(started)
				}
				<-release
				return DualModeReference{BornMin: 1400}, nil
			},
		}

		results := make(chan error, 2)
		go func() { _, err := repo.Load(); results <- err }()
		<-started
		go func() { _, err := repo.Load(); results <- err }()
		synctest.Wait()
		close(release)
		for range 2 {
			if err := <-results; err != nil {
				t.Fatalf("shared load: %v", err)
			}
		}
		if loads.Load() != 1 {
			t.Fatalf("loads = %d, want 1", loads.Load())
		}
	})
}

func TestDualModeReferenceRepositoryRetriesFailure(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	wantErr := errors.New("load failed")
	var loads atomic.Int32
	repo := &DualModeReferenceRepository{
		app: app,
		ctx: t.Context(),
		load: func(context.Context) (DualModeReference, error) {
			if loads.Add(1) == 1 {
				return DualModeReference{}, wantErr
			}
			return DualModeReference{BornMax: 1900}, nil
		},
	}
	if _, err := repo.Load(); !errors.Is(err, wantErr) {
		t.Fatalf("first load error = %v, want %v", err, wantErr)
	}
	if value, err := repo.Load(); err != nil || value.BornMax != 1900 {
		t.Fatalf("retry = (%#v, %v)", value, err)
	}
}

func TestDualModeReferenceSharedWaitHonoursCancellation(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	started := make(chan struct{})
	release := make(chan struct{})
	owner := &DualModeReferenceRepository{
		app: app,
		ctx: t.Context(),
		load: func(context.Context) (DualModeReference, error) {
			close(started)
			<-release
			return DualModeReference{}, nil
		},
	}
	ownerDone := make(chan error, 1)
	go func() { _, err := owner.Load(); ownerDone <- err }()
	<-started

	waiterCtx, cancel := context.WithCancel(t.Context())
	waiter := &DualModeReferenceRepository{app: app, ctx: waiterCtx, load: owner.load}
	cancel()
	waiterDone := make(chan error, 1)
	go func() { _, err := waiter.Load(); waiterDone <- err }()
	select {
	case err := <-waiterDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiter error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled waiter remained blocked")
	}

	close(release)
	if err := <-ownerDone; err != nil {
		t.Fatalf("owner load: %v", err)
	}
}

func TestDualModeReferenceInvalidationDuringLoadDoesNotRestoreStaleProjection(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	started := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	repo := &DualModeReferenceRepository{
		app: app,
		ctx: t.Context(),
		load: func(context.Context) (DualModeReference, error) {
			if loads.Add(1) == 1 {
				close(started)
				<-release
				return DualModeReference{BornMin: 1400}, nil
			}
			return DualModeReference{BornMin: 1500}, nil
		},
	}

	firstResult := make(chan DualModeReference, 1)
	go func() {
		value, _ := repo.Load()
		firstResult <- value
	}()
	<-started
	InvalidateDualModeReference(app)
	close(release)
	if stale := <-firstResult; stale.BornMin != 1400 {
		t.Fatalf("in-flight result = %#v, want its completed generation", stale)
	}
	fresh, err := repo.Load()
	if err != nil {
		t.Fatalf("fresh load: %v", err)
	}
	if fresh.BornMin != 1500 || loads.Load() != 2 {
		t.Fatalf("fresh result = %#v after %d loads", fresh, loads.Load())
	}
}
