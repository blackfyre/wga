package artists

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestArtistDetailCancellationStopsRelatedContent(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	artist, err := repositories.NewArtistRecordRepository(app).FindPublishedArtist("artistone000001")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stages := []string{}
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artist.detail.related_content" {
			cancel()
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	if _, err := buildArtistRecordViewContext(ctx, app, artist, app.Logger(), checkpoint); !errors.Is(err, context.Canceled) {
		t.Fatalf("buildArtistRecordViewContext() error = %v, want original cause", err)
	}
	want := []string{"artist.detail.work_count", "artist.detail.works", "artist.detail.related_content"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestArtistSelectionPreviewCancellationStopsLoopExpansion(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	for _, selection := range []struct {
		id    string
		title string
	}{
		{id: "selectionone001", title: "First selection"},
		{id: "selectiontwo001", title: "Second selection"},
	} {
		saveRecordRecord(t, app, "art_selections", selection.id, map[string]any{
			"artist": []string{"artistone000001"}, "title": selection.title, "display_title": selection.title,
			"artworks": []string{"artworkone00001"}, "source_path": selection.id, "source_hash": selection.id,
			"content_hash": selection.id, "published": true,
		})
	}
	artist, err := repositories.NewArtistRecordRepository(app).FindPublishedArtist("artistone000001")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stages := []string{}
	workStages := 0
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artist.detail.selection_works" {
			workStages++
			if workStages == 2 {
				cancel()
			}
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	if _, err := buildSelectionPreviewsContext(ctx, app, artist, 2, checkpoint); !errors.Is(err, context.Canceled) {
		t.Fatalf("buildSelectionPreviewsContext() error = %v, want cancellation", err)
	}
	want := []string{
		"artist.detail.selection_count",
		"artist.detail.selections",
		"artist.detail.selection_works",
		"artist.detail.selection_works",
	}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestArtworkDetailCancellationStopsProjection(t *testing.T) {
	app, _ := newArtworkRouteApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/artists/synthetic-artist-artistone000001/a-painting-workone00000001", nil).WithContext(ctx)
	request.SetPathValue("name", "synthetic-artist-artistone000001")
	request.SetPathValue("awid", "a-painting-workone00000001")
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: httptest.NewRecorder()}}
	stages := []string{}
	checkpoint := func(ctx context.Context, stage string) error {
		stages = append(stages, stage)
		if stage == "artwork.detail.projection" {
			cancel()
		}
		return requestprotection.Checkpoint(ctx, stage)
	}

	err := processArtworkWithCheckpoint(event, app, config.EnvironmentTest, checkpoint)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("processArtworkWithCheckpoint() response error = %v, want cancelled write", err)
	}
	failure, ok := utils.ServerFailureFrom(event)
	if !ok || !errors.Is(failure.Cause, context.Canceled) {
		t.Fatalf("recorded failure = %+v, want original cancellation cause", failure)
	}
	want := []string{"artwork.detail.artist_lookup", "artwork.detail.artwork_lookup", "artwork.detail.projection"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
}

func TestSelectionDetailCancellationStopsSubsequentStages(t *testing.T) {
	allStages := []string{
		"selection.detail.artist_lookup",
		"selection.detail.selection_lookup",
		"selection.detail.works",
		"selection.detail.related_content",
		"selection.detail.projection",
		"selection.detail.render",
	}
	for stopIndex, stopStage := range allStages {
		t.Run(stopStage, func(t *testing.T) {
			app, _ := newSelectionRouteApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			request := httptest.NewRequest(http.MethodGet, "/artists/synthetic-artist-artistone000001/selections/rselect00000001", nil).WithContext(ctx)
			request.SetPathValue("name", "synthetic-artist-artistone000001")
			request.SetPathValue("selectionID", "rselect00000001")
			event := &core.RequestEvent{Event: router.Event{Request: request, Response: httptest.NewRecorder()}}
			stages := []string{}
			checkpoint := func(ctx context.Context, stage string) error {
				stages = append(stages, stage)
				if stage == stopStage {
					cancel()
				}
				return requestprotection.Checkpoint(ctx, stage)
			}

			err := processSelectionWithCheckpoint(event, app, checkpoint)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("processSelectionWithCheckpoint() error = %v, want original cause", err)
			}
			failure, ok := utils.ServerFailureFrom(event)
			if !ok || !errors.Is(failure.Cause, context.Canceled) {
				t.Fatalf("recorded failure = %+v, want original cancellation cause", failure)
			}
			want := allStages[:stopIndex+1]
			if !reflect.DeepEqual(stages, want) {
				t.Fatalf("started stages = %v, want %v", stages, want)
			}
		})
	}
}
