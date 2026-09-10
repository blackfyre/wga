package artworks

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

func TestArtworkSearchCancellationIsNotRecordedAsServerFault(t *testing.T) {
	app := newArtworkSearchApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/artworks", nil).WithContext(ctx)
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: httptest.NewRecorder()}}

	if err := search(app, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("search() error = %v, want context.Canceled", err)
	}
	if _, ok := utils.ServerFailureFrom(event); ok {
		t.Fatal("artwork search cancellation recorded a server fault")
	}
}

func TestArtworkSearchCancellationStopsSubsequentRepositoryStage(t *testing.T) {
	app := newArtworkSearchApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	stages := []string{}
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artworks.search.records" {
			cancel()
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	if _, _, err := buildArtworkSearchViewContext(ctx, app, url.Values{}, 1, artworkSearchPageSize, checkpoint); !errors.Is(err, context.Canceled) {
		t.Fatalf("buildArtworkSearchViewContext() error = %v, want original cancellation cause", err)
	}
	want := []string{"artworks.search.count", "artworks.search.records"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestArtworkSearchResultsWorkflowChecksCancellationBeforeProjection(t *testing.T) {
	app := newArtworkSearchApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	stages := []string{}
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artworks.search.projection" {
			cancel()
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	if _, err := buildArtworkSearchResultsViewContext(ctx, app, url.Values{}, 1, artworkSearchPageSize, checkpoint); !errors.Is(err, context.Canceled) {
		t.Fatalf("buildArtworkSearchResultsViewContext() error = %v, want original cancellation cause", err)
	}
	want := []string{"artworks.search.count", "artworks.search.records", "artworks.search.projection"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestArtworkSearchCancellationSkipsRender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := renderArtworkSearch(ctx, func() error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("renderArtworkSearch() error = %v, want original cancellation cause", err)
	}
	if called {
		t.Fatal("artwork renderer was invoked after cancellation")
	}
}
