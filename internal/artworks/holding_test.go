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

func TestLocationAndDimensionsWithoutDimensions(t *testing.T) {
	location, dimensions := LocationAndDimensions("<p>1480 · Uffizi, Florence</p>")
	if location != "Uffizi, Florence" || dimensions != "" {
		t.Errorf("LocationAndDimensions() = %q, %q; want the location and no dimensions", location, dimensions)
	}
}

func TestLocationAndDimensionsDecodesEntities(t *testing.T) {
	location, dimensions := LocationAndDimensions("<p>1850 · Victoria &amp; Albert Museum, London · 20 &times; 30 cm</p>")
	if location != "Victoria & Albert Museum, London" || dimensions != "20 × 30 cm" {
		t.Errorf("LocationAndDimensions() = %q, %q; want decoded text", location, dimensions)
	}
}

func TestCurrentLocations(t *testing.T) {
	app := testutils.NewTestApp(t)
	locations := core.NewBaseCollection(constants.CollectionLocations)
	locations.Fields.Add(&core.TextField{Name: "name"})
	if err := app.Save(locations); err != nil {
		t.Fatalf("create locations: %v", err)
	}
	named := core.NewRecord(locations)
	named.Set("name", " Uffizi, Florence ")
	blank := core.NewRecord(locations)
	blank.Set("name", "  ")
	for _, record := range []*core.Record{named, blank} {
		if err := app.Save(record); err != nil {
			t.Fatalf("create location: %v", err)
		}
	}
	artworks := core.NewBaseCollection(constants.CollectionArtworks)
	artworks.Fields.Add(&core.RelationField{Name: "current_location_id", CollectionId: locations.Id, MaxSelect: 1})
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks: %v", err)
	}

	newArtwork := func(id string, locationID string) *core.Record {
		record := core.NewRecord(artworks)
		record.Id = id
		if locationID != "" {
			record.Set("current_location_id", locationID)
		}
		return record
	}
	got := CurrentLocations(app, []*core.Record{
		newArtwork("heldone00000001", named.Id),
		newArtwork("heldtwo00000001", named.Id),
		newArtwork("unheld000000001", ""),
		newArtwork("dangling0000001", "missingloc00000"),
		newArtwork("blankname000001", blank.Id),
		nil,
	})

	want := map[string]string{"heldone00000001": "Uffizi, Florence", "heldtwo00000001": "Uffizi, Florence"}
	if len(got) != len(want) {
		t.Fatalf("CurrentLocations() = %v, want %v", got, want)
	}
	for id, name := range want {
		if got[id] != name {
			t.Errorf("CurrentLocations()[%q] = %q, want %q", id, got[id], name)
		}
	}
	if empty := CurrentLocations(app, nil); len(empty) != 0 {
		t.Errorf("CurrentLocations(nil) = %v, want empty", empty)
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
