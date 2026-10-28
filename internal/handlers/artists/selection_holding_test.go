package artists

import (
	"net/http"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// addSelectionRouteHoldings gives the selection route fixture a locations
// collection and a current-location relation, stores a holding for one work,
// and adds a second work without one to the first selection.
func addSelectionRouteHoldings(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()

	locations := core.NewBaseCollection(constants.CollectionLocations)
	locations.Fields.Add(&core.TextField{Name: "name"})
	if err := app.Save(locations); err != nil {
		t.Fatalf("save locations: %v", err)
	}
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	artworks.Fields.Add(&core.RelationField{Name: "current_location_id", CollectionId: locations.Id, MaxSelect: 1})
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks relation: %v", err)
	}

	saveSelectionRouteRecord(t, app, constants.CollectionLocations, "locationone0001", map[string]any{
		"name": "Kunsthistorisches Museum, Vienna",
	})
	work, err := app.FindRecordById("artworks", "workone00000001")
	if err != nil {
		t.Fatalf("find work: %v", err)
	}
	work.Set("current_location_id", "locationone0001")
	if err := app.Save(work); err != nil {
		t.Fatalf("save work holding: %v", err)
	}
	saveSelectionRouteRecord(t, app, "artworks", "worktwo00000001", map[string]any{
		"title": "An Unheld Study", "author": []string{"artistone000001"}, "published": true, "technique": "Chalk",
	})
	selection, err := app.FindRecordById("art_selections", "rselect00000001")
	if err != nil {
		t.Fatalf("find selection: %v", err)
	}
	selection.Set("artworks", []string{"workone00000001", "worktwo00000001"})
	if err := app.Save(selection); err != nil {
		t.Fatalf("save selection works: %v", err)
	}
}

func TestSelectionRouteShowsStoredHoldingOnlyWhenPresent(t *testing.T) {
	app, request := newSelectionRouteApp(t)
	addSelectionRouteHoldings(t, app)

	response := request("/artists/synthetic-artist-artistone000001/selections/rselect00000001")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if got := strings.Count(body, "data-plate-location"); got != 1 {
		t.Fatalf("location lines = %d, want 1 (only the held work)", got)
	}
	held := strings.Index(body, "A Painting</h2>")
	location := strings.Index(body, "data-plate-location>Kunsthistorisches Museum, Vienna</p>")
	unheld := strings.Index(body, "An Unheld Study</h2>")
	if held < 0 || location < held || unheld < location {
		t.Error("the stored holding must render on the held work's plate and nowhere else")
	}
}

func TestArtistRecordSelectionPreviewsShowStoredHolding(t *testing.T) {
	app, request := newSelectionRouteApp(t)
	addSelectionRouteHoldings(t, app)

	response := request("/artists/synthetic-artist-artistone000001")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "data-plate-location>Kunsthistorisches Museum, Vienna</p>") {
		t.Error("selection previews must show the stored holding")
	}
	if !strings.Contains(body, "data-selection-plate-grid") {
		t.Error("selection previews must use the shared selection plate grid")
	}
}
