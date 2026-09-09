package artists

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestArtistSearchCancellationIsNotRecordedAsServerFault(t *testing.T) {
	app := newArtistRecordApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/artists", nil).WithContext(ctx)
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: httptest.NewRecorder()}}

	if err := processArtists(app, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("processArtists() error = %v, want context.Canceled", err)
	}
	if _, ok := utils.ServerFailureFrom(event); ok {
		t.Fatal("artist search cancellation recorded a server fault")
	}
}

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
