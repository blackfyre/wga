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
	cause := errors.New("artist search cancelled")
	ctx, cancel := context.WithCancelCause(context.Background())
	stages := []string{}
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artists.search.periods" {
			cancel(cause)
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	if _, _, err := buildArtistIndexViewContext(ctx, app, url.Values{}, checkpoint); !errors.Is(err, cause) {
		t.Fatalf("buildArtistIndexViewContext() error = %v, want original cancellation cause", err)
	}
	want := []string{"artists.search.schools", "artists.search.periods"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestArtistSearchCancellationSkipsRender(t *testing.T) {
	cause := errors.New("artist render cancelled")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	called := false
	err := renderArtistIndex(ctx, func() error {
		called = true
		return nil
	})
	if !errors.Is(err, cause) {
		t.Fatalf("renderArtistIndex() error = %v, want original cancellation cause", err)
	}
	if called {
		t.Fatal("artist renderer was invoked after cancellation")
	}
}
