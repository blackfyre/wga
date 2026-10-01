package postcards

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestSendPostcardRendersPageAndHtmxFragment(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installComposeArtwork(t, app, "Selected work")
	changedArtworkID := installComposeArtwork(t, app, "Changed work")
	for _, test := range []struct {
		name string
		htmx bool
	}{
		{name: "page"},
		{name: "fragment", htmx: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/postcard/send?awid="+artworkID, nil)
			if test.htmx {
				request.Header.Set("HX-Request", "true")
			}
			event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: recorder}}
			if err := sendPostcard(app, event, config.Captcha{}); err != nil {
				t.Fatal(err)
			}
			body := recorder.Body.String()
			if recorder.Code != http.StatusOK || !strings.Contains(body, "Selected work") || !strings.Contains(body, "id=\"postcard-compose\"") {
				t.Fatalf("compose response status=%d body=%q", recorder.Code, body)
			}
			if test.htmx && strings.Contains(body, "wga-dialog-panel") {
				t.Fatal("HTMX composer response retained dialog markup")
			}
		})
	}
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: httptest.NewRequest(http.MethodGet, "/postcard/send?awid="+changedArtworkID, nil), Response: recorder}}
	if err := sendPostcard(app, event, config.Captcha{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recorder.Body.String(), "Changed work") {
		t.Fatal("changed artwork did not replace compose selection")
	}
}

func TestSendPostcardRejectsMissingArtwork(t *testing.T) {
	app := testutils.NewTestApp(t)
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: httptest.NewRequest(http.MethodGet, "/postcard/send?awid=missing", nil), Response: recorder}}
	_ = sendPostcard(app, event, config.Captcha{})
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", recorder.Code)
	}
}

func installComposeArtwork(t *testing.T, app core.App, title string) string {
	t.Helper()
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		artists = core.NewBaseCollection("artists")
		artists.Fields.Add(&core.BoolField{Name: "published"}, &core.TextField{Name: "filing_name"}, &core.TextField{Name: "short_name"})
		if err := app.Save(artists); err != nil {
			t.Fatal(err)
		}
	}
	artist := core.NewRecord(artists)
	artist.Set("published", true)
	artist.Set("filing_name", "Artist, Test")
	artist.Set("short_name", "Test Artist")
	if err := app.Save(artist); err != nil {
		t.Fatal(err)
	}
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		artworks = core.NewBaseCollection("artworks")
		artworks.Fields.Add(&core.BoolField{Name: "published"}, &core.TextField{Name: "title"}, &core.TextField{Name: "technique"}, &core.TextField{Name: "image"}, &core.RelationField{Name: "author", CollectionId: artists.Id, MaxSelect: 1})
		if err := app.Save(artworks); err != nil {
			t.Fatal(err)
		}
	}
	artwork := core.NewRecord(artworks)
	artwork.Set("published", true)
	artwork.Set("title", title)
	artwork.Set("technique", "Oil on canvas")
	artwork.Set("author", []string{artist.Id})
	if err := app.Save(artwork); err != nil {
		t.Fatal(err)
	}
	return artwork.Id
}

// TestSendPostcardAttributesThePublishedAuthor verifies the composer uses the
// same published-author resolution as the received card, so SEND YOUR OWN
// opens for a co-authored work whose first author is unpublished.
func TestSendPostcardAttributesThePublishedAuthor(t *testing.T) {
	app := testutils.NewTestApp(t)
	artists := core.NewBaseCollection("artists")
	artists.Fields.Add(&core.BoolField{Name: "published"}, &core.TextField{Name: "filing_name"}, &core.TextField{Name: "short_name"})
	if err := app.Save(artists); err != nil {
		t.Fatal(err)
	}
	authorIDs := make([]string, 0, 2)
	for _, values := range []map[string]any{
		{"published": false, "filing_name": "ARTIST, Hidden", "short_name": "Hidden"},
		{"published": true, "filing_name": "ARTIST, Public", "short_name": "Public"},
	} {
		artist := core.NewRecord(artists)
		for key, value := range values {
			artist.Set(key, value)
		}
		if err := app.Save(artist); err != nil {
			t.Fatal(err)
		}
		authorIDs = append(authorIDs, artist.Id)
	}
	artworks := core.NewBaseCollection("artworks")
	artworks.Fields.Add(&core.BoolField{Name: "published"}, &core.TextField{Name: "title"}, &core.TextField{Name: "technique"}, &core.TextField{Name: "image"}, &core.RelationField{Name: "author", CollectionId: artists.Id, MaxSelect: 2})
	if err := app.Save(artworks); err != nil {
		t.Fatal(err)
	}
	artwork := core.NewRecord(artworks)
	artwork.Set("published", true)
	artwork.Set("title", "Co-authored work")
	artwork.Set("author", authorIDs)
	if err := app.Save(artwork); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: httptest.NewRequest(http.MethodGet, "/postcard/send?awid="+artwork.Id, nil), Response: recorder}}
	if err := sendPostcard(app, event, config.Captcha{}); err != nil {
		t.Fatal(err)
	}
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, "ARTIST, Public") {
		t.Fatalf("status=%d, want the composer attributed to the published author", recorder.Code)
	}
	if strings.Contains(body, "ARTIST, Hidden") {
		t.Fatal("composer exposed the unpublished author")
	}
}

// TestRenderFormCountsTheRedisplayedMessage verifies the no-JavaScript counter
// charges a redisplayed message for its visible text, as validation does, and
// not for its formatting markup.
func TestRenderFormCountsTheRedisplayedMessage(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installComposeArtwork(t, app, "Selected work")
	for _, tc := range []struct {
		name    string
		message string
		counter string
	}{
		{name: "new composer", counter: ">300 CHARACTERS LEFT<"},
		{name: "formatted message", message: "<p><b>Hello</b> world</p>", counter: ">289 CHARACTERS LEFT<"},
		{name: "over the limit", message: "<p>" + strings.Repeat("é", 302) + "</p>", counter: ">2 CHARACTERS OVER THE LIMIT<"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			event := &core.RequestEvent{App: app, Event: router.Event{Request: httptest.NewRequest(http.MethodPost, "/postcards", nil), Response: recorder}}
			values := pages.PostcardComposeView{Message: tc.message}
			if err := renderForm(artworkID, values, "Check the message", http.StatusUnprocessableEntity, app, event, config.Captcha{}); err != nil {
				t.Fatal(err)
			}
			if body := recorder.Body.String(); !strings.Contains(body, tc.counter) {
				t.Fatalf("composer counter missing %q", tc.counter)
			}
		})
	}
}
