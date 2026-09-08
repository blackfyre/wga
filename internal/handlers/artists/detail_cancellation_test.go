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
