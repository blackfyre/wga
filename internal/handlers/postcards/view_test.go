package postcards

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/testutils"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

// TestViewPostcardSetsRecipientResponseHeaders verifies a successful recipient
// page is not cached and does not leak the bearer token via the Referer header.
func TestViewPostcardSetsRecipientResponseHeaders(t *testing.T) {
	app := testutils.NewTestApp(t)

	artists := core.NewBaseCollection(constants.CollectionArtists)
	artists.Fields.Add(&core.TextField{Name: "name"})
	if err := app.Save(artists); err != nil {
		t.Fatalf("create artists collection: %v", err)
	}
	author := core.NewRecord(artists)
	author.Set("name", "Artist")
	if err := app.Save(author); err != nil {
		t.Fatalf("create artist: %v", err)
	}

	artworks := core.NewBaseCollection(constants.CollectionArtworks)
	artworks.Fields.Add(
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.TextField{Name: "title"},
		&core.TextField{Name: "comment"},
		&core.TextField{Name: "technique"},
		&core.RelationField{Name: "author", CollectionId: artists.Id, MaxSelect: 1},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks collection: %v", err)
	}
	artwork := core.NewRecord(artworks)
	artwork.Set("published", true)
	artwork.Set("title", "Work")
	artwork.Set("author", author.Id)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("create artwork: %v", err)
	}

	postcards := core.NewBaseCollection(constants.CollectionPostcards)
	postcards.Id = constants.CollectionPostcards
	postcards.Fields.Add(
		&core.TextField{Name: "sender_name"},
		&core.TextField{Name: "message"},
		&core.TextField{Name: "image_id"},
		&core.BoolField{Name: "include_music"},
		&core.SelectField{Name: "status", Values: []string{"queued", "sent", "received", "cancelled"}, MaxSelect: 1},
		&core.DateField{Name: "received_at"},
	)
	if err := app.Save(postcards); err != nil {
		t.Fatalf("create postcards collection: %v", err)
	}
	postcard := core.NewRecord(postcards)
	postcard.Set("sender_name", "Sender")
	postcard.Set("message", "Hello")
	postcard.Set("image_id", artwork.Id)
	postcard.Set("status", "sent")
	if err := app.Save(postcard); err != nil {
		t.Fatalf("create postcard: %v", err)
	}

	deliveries := core.NewBaseCollection("tracking_postcard_deliveries")
	deliveries.Id = "tracking_postcard_deliveries"
	deliveries.Fields.Add(
		&core.RelationField{Name: "postcard", CollectionId: postcards.Id},
		&core.TextField{Name: "recipient"},
		&core.SelectField{Name: "status", Values: []string{"pending", "sent", "cancelled"}, MaxSelect: 1},
		&core.TextField{Name: "view_token_envelope"},
		&core.TextField{Name: "view_token_hash"},
		&core.DateField{Name: "view_expires_at"},
	)
	if err := app.Save(deliveries); err != nil {
		t.Fatalf("create deliveries collection: %v", err)
	}
	token := newTestRecipientToken(t)
	delivery := core.NewRecord(deliveries)
	delivery.Set("postcard", postcard.Id)
	delivery.Set("recipient", "recipient@example.test")
	delivery.Set("status", "sent")
	delivery.Set("view_token_hash", postcardworkflow.HashRecipientToken(token))
	delivery.Set("view_expires_at", types.NowDateTime().Add(24*time.Hour))
	if err := app.Save(delivery); err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/postcard?token="+token, nil)
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: recorder,
		},
	}

	if err := viewPostcard(app, event); err != nil {
		t.Fatalf("view postcard: %v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
}

func newTestRecipientToken(t *testing.T) string {
	t.Helper()
	material := make([]byte, 32)
	if _, err := rand.Read(material); err != nil {
		t.Fatalf("generate recipient token: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(material)
}

// TestViewPostcardTokenlessLanding verifies a tokenless GET renders the public
// landing with the correct title, canonical URL, push URL, and an ordinary
// /artworks link, without applying recipient confidentiality headers.
func TestViewPostcardTokenlessLanding(t *testing.T) {
	app := testutils.NewTestApp(t)
	configurePostcardPublicURL(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/postcard", nil)
	event := newPostcardViewEvent(app, request, recorder)

	if err := viewPostcard(app, event); err != nil {
		t.Fatalf("view postcard landing: %v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != "/postcard" {
		t.Fatalf("HX-Push-Url = %q, want /postcard", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("Cache-Control = %q, want empty on tokenless landing", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "" {
		t.Fatalf("Referrer-Policy = %q, want empty on tokenless landing", got)
	}
	for _, fragment := range []string{
		"Postcards - WGA",
		`<link rel="canonical" href="https://gallery.example/postcard"`,
		`href="/artworks"`,
		"Postcards are composed from published artwork pages",
	} {
		if !strings.Contains(recorder.Body.String(), fragment) {
			t.Errorf("response does not contain %q", fragment)
		}
	}
}

// TestViewPostcardRejectsExplicitEmptyAndInvalidToken verifies an explicitly
// empty or malformed token is still denied rather than falling back to the
// public landing, and that the denied recipient request still carries
// confidentiality headers.
func TestViewPostcardRejectsExplicitEmptyAndInvalidToken(t *testing.T) {
	app := testutils.NewTestApp(t)

	for _, rawURL := range []string{"/postcard?token=", "/postcard?token=not-a-valid-token"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, rawURL, nil)
		event := newPostcardViewEvent(app, request, recorder)

		if err := viewPostcard(app, event); err != nil {
			t.Fatalf("%s: %v", rawURL, err)
		}
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d", rawURL, recorder.Code, http.StatusNotFound)
		}
		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s Cache-Control = %q, want no-store", rawURL, got)
		}
		if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("%s Referrer-Policy = %q, want no-referrer", rawURL, got)
		}
	}
}

// TestViewPostcardLandingResponseMatrix verifies the full document, shell
// navigation, and feature-local response contracts for the tokenless landing.
func TestViewPostcardLandingResponseMatrix(t *testing.T) {
	app := testutils.NewTestApp(t)

	full := httptest.NewRecorder()
	if err := viewPostcard(app, newPostcardViewEvent(app, httptest.NewRequest(http.MethodGet, "/postcard", nil), full)); err != nil {
		t.Fatalf("full: %v", err)
	}
	if full.Code != http.StatusOK {
		t.Fatalf("full status = %d, want %d", full.Code, http.StatusOK)
	}
	if !strings.Contains(full.Body.String(), "<html") {
		t.Error("full response must render the full document")
	}
	if got := strings.Count(full.Body.String(), `id="mc-area"`); got != 1 {
		t.Errorf("full rendered %d #mc-area elements, want exactly 1", got)
	}

	for _, target := range []string{"mc-area", "#mc-area"} {
		shell := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/postcard", nil)
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", target)
		if err := viewPostcard(app, newPostcardViewEvent(app, request, shell)); err != nil {
			t.Fatalf("shell(%s): %v", target, err)
		}
		if !strings.Contains(shell.Body.String(), "<html") {
			t.Errorf("shell(%s) must render the full document", target)
		}
		if got := strings.Count(shell.Body.String(), `id="mc-area"`); got != 1 {
			t.Errorf("shell(%s) rendered %d #mc-area elements, want exactly 1", target, got)
		}
	}

	local := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/postcard", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "postcard-landing")
	if err := viewPostcard(app, newPostcardViewEvent(app, request, local)); err != nil {
		t.Fatalf("local: %v", err)
	}
	if local.Code != http.StatusOK {
		t.Fatalf("local status = %d, want %d", local.Code, http.StatusOK)
	}
	if strings.Contains(local.Body.String(), "<html") {
		t.Error("feature-local response must not render the full document")
	}
	if got := strings.Count(local.Body.String(), `id="postcard-landing"`); got != 1 {
		t.Errorf("feature-local rendered %d #postcard-landing elements, want exactly 1", got)
	}
	if strings.Contains(local.Body.String(), `id="mc-area"`) {
		t.Error("feature-local response must not carry #mc-area")
	}
}

func newPostcardViewEvent(app core.App, request *http.Request, recorder *httptest.ResponseRecorder) *core.RequestEvent {
	return &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: recorder,
		},
	}
}

func configurePostcardPublicURL(t *testing.T) {
	t.Helper()
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{
			"WGA_ENV":            "development",
			"WGA_PROTOCOL":       "https",
			"WGA_HOSTNAME":       "gallery.example",
			"WGA_SENDER_NAME":    "WGA",
			"WGA_SENDER_ADDRESS": "sender@example.com",
		}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() { apputils.ConfigurePublicURL(config.PublicURL{}) })
}
