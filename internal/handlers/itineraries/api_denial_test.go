package itineraries

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// TestAnonymousAPIDenial proves that an anonymous PocketBase API client cannot
// list, view, create, update, or delete itinerary records. Both collections
// keep nil API rules, so every verb returns 403 without exposing data.
func TestAnonymousAPIDenialForItineraryCollections(t *testing.T) {
	app, mux := newItineraryMux(t)

	itinerary, stop := seedItineraryRecords(t, app)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"itineraries list", http.MethodGet, "/api/collections/itineraries/records", ""},
		{"itineraries view", http.MethodGet, "/api/collections/itineraries/records/" + itinerary.Id, ""},
		{"itineraries create", http.MethodPost, "/api/collections/itineraries/records", `{"owner":"hijack","status":"draft"}`},
		{"itineraries update", http.MethodPatch, "/api/collections/itineraries/records/" + itinerary.Id, `{"title":"hijacked"}`},
		{"itineraries delete", http.MethodDelete, "/api/collections/itineraries/records/" + itinerary.Id, ""},
		{"stops list", http.MethodGet, "/api/collections/itinerary_stops/records", ""},
		{"stops view", http.MethodGet, "/api/collections/itinerary_stops/records/" + stop.Id, ""},
		{"stops create", http.MethodPost, "/api/collections/itinerary_stops/records", `{"itinerary":"` + itinerary.Id + `"}`},
		{"stops update", http.MethodPatch, "/api/collections/itinerary_stops/records/" + stop.Id, `{"narration":"hijacked"}`},
		{"stops delete", http.MethodDelete, "/api/collections/itinerary_stops/records/" + stop.Id, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			if response.Code != http.StatusForbidden {
				t.Errorf("%s %s status = %d, want 403", tc.method, tc.path, response.Code)
			}
		})
	}
}

func seedItineraryRecords(t *testing.T, app *pocketbase.PocketBase) (*core.Record, *core.Record) {
	t.Helper()

	itineraries, err := app.FindCollectionByNameOrId(itineraryworkflow.CollectionItineraries)
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}
	itinerary := core.NewRecord(itineraries)
	itinerary.Set("owner", "seed-owner-digest")
	itinerary.Set("status", "draft")
	if err := app.Save(itinerary); err != nil {
		t.Fatalf("save itinerary: %v", err)
	}

	stops, err := app.FindCollectionByNameOrId(itineraryworkflow.CollectionItineraryStops)
	if err != nil {
		t.Fatalf("find itinerary_stops: %v", err)
	}
	stop := core.NewRecord(stops)
	stop.Set("itinerary", itinerary.Id)
	stop.Set("artwork", testArtworkID)
	stop.Set("position", 0)
	if err := app.Save(stop); err != nil {
		t.Fatalf("save stop: %v", err)
	}

	return itinerary, stop
}
