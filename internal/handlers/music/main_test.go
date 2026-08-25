package music

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestParsePlayerRequestAcceptsStoredSongIdentifier(t *testing.T) {
	request, err := parsePlayerRequest(url.Values{
		"song": {"72d6bb922f76aea"},
	})
	if err != nil {
		t.Fatalf("parse valid request: %v", err)
	}
	if request.SongID != "72d6bb922f76aea" {
		t.Errorf("song id = %q", request.SongID)
	}
}

func TestParsePlayerRequestRejectsUnsafeOrUnboundedInput(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{name: "missing song", values: url.Values{}},
		{name: "short song", values: url.Values{"song": {"short"}}},
		{name: "uppercase song", values: url.Values{"song": {"72D6BB922F76AEA"}}},
		{name: "punctuated song", values: url.Values{"song": {"72d6bb922f76ae-"}}},
		{name: "caller display value", values: url.Values{"song": {"72d6bb922f76aea"}, "piece": {"Fabricated"}}},
		{name: "caller source value", values: url.Values{"song": {"72d6bb922f76aea"}, "src": {"https://example.com/song.mp3"}}},
		{name: "duplicate song", values: url.Values{"song": {"72d6bb922f76aea", "aaaaaaaaaaaaaaa"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parsePlayerRequest(test.values); !errors.Is(err, errInvalidPlayerRequest) {
				t.Fatalf("error = %v, want invalid player request", err)
			}
		})
	}
}

func TestPlayerRouteDeniesUnpublishedSongOrComposer(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	composers := core.NewBaseCollection("Music_composer")
	composers.Id = "music_composer"
	composers.MarkAsNew()
	composers.Fields.Add(
		&core.TextField{Id: "composer_name", Name: "name", Required: true},
		&core.BoolField{Id: "composer_published", Name: "published"},
	)
	if err := app.Save(composers); err != nil {
		t.Fatalf("save composer collection: %v", err)
	}
	publishedComposer := core.NewRecord(composers)
	publishedComposer.Id = "composer0000001"
	publishedComposer.Set("name", "Sweelinck")
	publishedComposer.Set("published", true)
	if err := app.Save(publishedComposer); err != nil {
		t.Fatalf("save published composer: %v", err)
	}
	unpublishedComposer := core.NewRecord(composers)
	unpublishedComposer.Id = "composer0000002"
	unpublishedComposer.Set("name", "Hidden Composer")
	unpublishedComposer.Set("published", false)
	if err := app.Save(unpublishedComposer); err != nil {
		t.Fatalf("save unpublished composer: %v", err)
	}

	songs := core.NewBaseCollection("Music_song")
	songs.Id = "music_song"
	songs.MarkAsNew()
	songs.Fields.Add(
		&core.TextField{Id: "music_title", Name: "title", Required: true},
		&core.RelationField{Id: "music_composer", Name: "composer", CollectionId: composers.Id, MinSelect: 1, MaxSelect: 20},
		&core.TextField{Id: "music_source", Name: "source", Required: true},
		&core.BoolField{Id: "music_published", Name: "published"},
	)
	if err := app.Save(songs); err != nil {
		t.Fatalf("save music collection: %v", err)
	}
	unpublishedSong := core.NewRecord(songs)
	unpublishedSong.Id = "72d6bb922f76aea"
	unpublishedSong.Set("title", "Secret Study")
	unpublishedSong.Set("composer", []string{publishedComposer.Id})
	unpublishedSong.Set("source", "secret.mp3")
	unpublishedSong.Set("published", false)
	if err := app.Save(unpublishedSong); err != nil {
		t.Fatalf("save unpublished song: %v", err)
	}
	unpublishedComposerSong := core.NewRecord(songs)
	unpublishedComposerSong.Id = "72d6bb922f76beb"
	unpublishedComposerSong.Set("title", "Hidden Composer Piece")
	unpublishedComposerSong.Set("composer", []string{unpublishedComposer.Id})
	unpublishedComposerSong.Set("source", "hidden.mp3")
	unpublishedComposerSong.Set("published", true)
	if err := app.Save(unpublishedComposerSong); err != nil {
		t.Fatalf("save song with unpublished composer: %v", err)
	}

	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		for _, songID := range []string{unpublishedSong.Id, unpublishedComposerSong.Id} {
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/player?song="+songID, nil))
			if recorder.Code != http.StatusNotFound {
				t.Errorf("unpublished song %s status = %d, want 404", songID, recorder.Code)
			}
			body := recorder.Body.String()
			for _, leaked := range []string{"Secret Study", "Hidden Composer Piece", "Hidden Composer", "secret.mp3", "hidden.mp3", "/api/files/music_song"} {
				if strings.Contains(body, leaked) {
					t.Errorf("unpublished song %s response leaks %q: %s", songID, leaked, body)
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestPlayerRouteResolvesStoredRecordingAndNeverAutoplays(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	composers := core.NewBaseCollection("Music_composer")
	composers.Id = "music_composer"
	composers.MarkAsNew()
	composers.Fields.Add(&core.TextField{Id: "composer_name", Name: "name", Required: true})
	composers.Fields.Add(&core.BoolField{Id: "composer_published", Name: "published"})
	if err := app.Save(composers); err != nil {
		t.Fatalf("save composer collection: %v", err)
	}
	composer := core.NewRecord(composers)
	composer.Id = "composer0000001"
	composer.Set("name", "Sweelinck")
	composer.Set("published", true)
	if err := app.Save(composer); err != nil {
		t.Fatalf("save composer: %v", err)
	}

	collection := core.NewBaseCollection("Music_song")
	collection.Id = "music_song"
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Id: "music_title", Name: "title", Required: true},
		&core.RelationField{Id: "music_composer", Name: "composer", CollectionId: composers.Id, MinSelect: 1, MaxSelect: 20},
		&core.TextField{Id: "music_source", Name: "source", Required: true},
		&core.BoolField{Id: "music_published", Name: "published"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save music collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = "72d6bb922f76aea"
	record.Set("title", "Fantasia")
	record.Set("composer", []string{composer.Id})
	record.Set("source", "fantasia.mp3")
	record.Set("published", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("save music record: %v", err)
	}

	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		valid := httptest.NewRecorder()
		mux.ServeHTTP(valid, httptest.NewRequest(http.MethodGet, "/player?song=72d6bb922f76aea", nil))
		if valid.Code != http.StatusOK {
			t.Fatalf("valid status = %d, want 200", valid.Code)
		}
		body := valid.Body.String()
		for _, expected := range []string{"Sweelinck", "Fantasia", `src="/api/files/music_song/72d6bb922f76aea/fantasia.mp3"`, `controls`, `preload="metadata"`, `data-wga-music-player`} {
			if !strings.Contains(body, expected) {
				t.Errorf("player body does not contain %q", expected)
			}
		}
		if strings.Contains(strings.ToLower(body), "autoplay") {
			t.Error("player must not contain autoplay")
		}
		if got := valid.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q", got)
		}

		invalid := httptest.NewRecorder()
		mux.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/player?song=https%3A%2F%2Fexample.com", nil))
		if invalid.Code != http.StatusBadRequest {
			t.Errorf("invalid status = %d, want 400", invalid.Code)
		}

		missing := httptest.NewRecorder()
		mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/player?song=aaaaaaaaaaaaaaa", nil))
		if missing.Code != http.StatusNotFound {
			t.Errorf("missing status = %d, want 404", missing.Code)
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
