package artists

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestArtistLookupErrorReturnsNotFoundForMissingRecord(t *testing.T) {
	app, _ := newArtworkRouteApp(t)
	event, response := newArtistRequestEvent()

	if err := artistLookupError(event, app, "missing-artist", sql.ErrNoRows); err != nil {
		t.Fatalf("artist lookup error = %v", err)
	}
	if response.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if _, ok := utils.ServerFailureFrom(event); ok {
		t.Fatal("missing artist must not record a server failure")
	}
}

func TestArtistLookupErrorPreservesOperationalCause(t *testing.T) {
	app, _ := newArtworkRouteApp(t)
	event, response := newArtistRequestEvent()
	cause := errors.New("artist query failed")

	if err := artistLookupError(event, app, "affected-artist", cause); err != nil {
		t.Fatalf("artist lookup error = %v", err)
	}
	if response.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	failure, ok := utils.ServerFailureFrom(event)
	if !ok {
		t.Fatal("expected artist lookup failure metadata")
	}
	if failure.Category != "artist_lookup" {
		t.Errorf("failure category = %q, want artist_lookup", failure.Category)
	}
	if !errors.Is(failure.Cause, cause) {
		t.Errorf("failure cause = %v, want %v", failure.Cause, cause)
	}
}

func newArtistRequestEvent() (*core.RequestEvent, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodGet, "/artists/affected-artist", nil)
	response := httptest.NewRecorder()
	return &core.RequestEvent{Event: router.Event{Request: request, Response: response}}, response
}
