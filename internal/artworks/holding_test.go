package artworks

import (
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
)

func TestLocationAndDimensions(t *testing.T) {
	location, dimensions := LocationAndDimensions("<p>1902 · Synthetic Gallery, Test City · 101 x 201 cm</p>")
	if location != "Synthetic Gallery, Test City" {
		t.Errorf("location = %q, want %q", location, "Synthetic Gallery, Test City")
	}
	if dimensions != "101 x 201 cm" {
		t.Errorf("dimensions = %q, want %q", dimensions, "101 x 201 cm")
	}
}

func TestLocationAndDimensionsWithoutCatalogueSummary(t *testing.T) {
	location, dimensions := LocationAndDimensions("<p>Commentary without catalogue metadata.</p>")
	if location != "" || dimensions != "" {
		t.Errorf("LocationAndDimensions() = %q, %q; want empty values", location, dimensions)
	}
}

func TestCurrentLocation(t *testing.T) {
	app := testutils.NewTestApp(t)
	locations := core.NewBaseCollection(constants.CollectionLocations)
	locations.Fields.Add(&core.TextField{Name: "name"})
	if err := app.Save(locations); err != nil {
		t.Fatalf("create locations: %v", err)
	}
	location := core.NewRecord(locations)
	location.Set("name", "  Mauritshuis, The Hague ")
	if err := app.Save(location); err != nil {
		t.Fatalf("create location: %v", err)
	}
	artworks := core.NewBaseCollection(constants.CollectionArtworks)
	artworks.Fields.Add(&core.RelationField{Name: "current_location_id", CollectionId: locations.Id, MaxSelect: 1})
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks: %v", err)
	}

	held := core.NewRecord(artworks)
	held.Set("current_location_id", location.Id)
	if got := CurrentLocation(app, held); got != "Mauritshuis, The Hague" {
		t.Errorf("CurrentLocation() = %q, want %q", got, "Mauritshuis, The Hague")
	}
	if got := CurrentLocation(app, core.NewRecord(artworks)); got != "" {
		t.Errorf("CurrentLocation() without relation = %q, want empty", got)
	}
	dangling := core.NewRecord(artworks)
	dangling.Set("current_location_id", "missingloc00000")
	if got := CurrentLocation(app, dangling); got != "" {
		t.Errorf("CurrentLocation() with unresolvable relation = %q, want empty", got)
	}
}
