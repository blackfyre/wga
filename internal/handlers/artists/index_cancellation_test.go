package artists

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/blackfyre/wga/internal/requestprotection"
)

func TestArtistSearchCancellationStopsSubsequentRepositoryStage(t *testing.T) {
	app := newArtistRecordApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	stages := []string{}
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artists.search.periods" {
			cancel()
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	if _, _, err := buildArtistIndexViewContext(ctx, app, url.Values{}, checkpoint); !errors.Is(err, context.Canceled) {
		t.Fatalf("buildArtistIndexViewContext() error = %v, want original cancellation cause", err)
	}
	want := []string{"artists.search.schools", "artists.search.periods"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestArtistSearchCancellationSkipsRender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := renderArtistIndex(ctx, func() error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("renderArtistIndex() error = %v, want original cancellation cause", err)
	}
	if called {
		t.Fatal("artist renderer was invoked after cancellation")
	}
}
