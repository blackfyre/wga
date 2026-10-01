package postcards

import (
	"errors"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
)

type receivedFixture struct {
	app       core.App
	artists   *core.Collection
	artworks  *core.Collection
	locations *core.Collection
	postcards *core.Collection
}

func newReceivedFixture(t *testing.T) *receivedFixture {
	t.Helper()
	app := testutils.NewTestApp(t)
	f := &receivedFixture{app: app}

	f.artists = core.NewBaseCollection(constants.CollectionArtists)
	f.artists.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "filing_name"}, &core.TextField{Name: "short_name"})
	f.locations = core.NewBaseCollection(constants.CollectionLocations)
	f.locations.Fields.Add(&core.TextField{Name: "name"})
	for _, collection := range []*core.Collection{f.artists, f.locations} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("create %s: %v", collection.Name, err)
		}
	}
	f.artworks = core.NewBaseCollection(constants.CollectionArtworks)
	f.artworks.Fields.Add(
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.TextField{Name: "title"},
		&core.TextField{Name: "comment"},
		&core.TextField{Name: "technique"},
		&core.RelationField{Name: "author", CollectionId: f.artists.Id, MaxSelect: 1},
		&core.RelationField{Name: "current_location_id", CollectionId: f.locations.Id, MaxSelect: 1},
	)
	f.postcards = core.NewBaseCollection("received_postcards_fixture")
	f.postcards.Fields.Add(&core.TextField{Name: "image_id"})
	for _, collection := range []*core.Collection{f.artworks, f.postcards} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("create %s: %v", collection.Name, err)
		}
	}
	return f
}

func (f *receivedFixture) save(t *testing.T, collection *core.Collection, values map[string]any) *core.Record {
	t.Helper()
	record := core.NewRecord(collection)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := f.app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection.Name, err)
	}
	return record
}

func (f *receivedFixture) artist(t *testing.T) *core.Record {
	return f.save(t, f.artists, map[string]any{"name": "Johannes Vermeer", "filing_name": "VERMEER, Johannes", "short_name": "Vermeer"})
}

func (f *receivedFixture) postcardFor(t *testing.T, artwork *core.Record) *core.Record {
	return f.save(t, f.postcards, map[string]any{"image_id": artwork.Id})
}

func TestResolveReceivedCardLinksToTheRecordAndComposer(t *testing.T) {
	f := newReceivedFixture(t)
	artist := f.artist(t)
	holding := f.save(t, f.locations, map[string]any{"name": "Mauritshuis, The Hague"})
	artwork := f.save(t, f.artworks, map[string]any{
		"published": true, "title": "Girl with a Pearl Earring", "author": artist.Id, "current_location_id": holding.Id,
		"technique": "Oil on canvas, 46,5 x 40 cm",
		"comment":   "<p>c. 1665 · Catalogue Gallery · 46,5 x 40 cm</p>",
	})

	card, err := ResolveReceivedCard(f.app, f.postcardFor(t, artwork))
	if err != nil {
		t.Fatalf("ResolveReceivedCard() error = %v", err)
	}
	wantRecord := "/artists/johannes-vermeer-" + artist.Id + "/girl-with-a-pearl-earring-" + artwork.Id
	if card.RecordURL != wantRecord {
		t.Errorf("RecordURL = %q, want %q", card.RecordURL, wantRecord)
	}
	if want := "/postcard/send?awid=" + artwork.Id; card.ComposeURL != want {
		t.Errorf("ComposeURL = %q, want %q", card.ComposeURL, want)
	}
	if card.ArtistFilingName != "VERMEER, Johannes" || card.Title != "Girl with a Pearl Earring" {
		t.Errorf("identity = %q / %q", card.ArtistFilingName, card.Title)
	}
	if card.Technique != "Oil on canvas" || card.Dimensions != "46,5 x 40 cm" {
		t.Errorf("technique, dimensions = %q, %q; want the dimensions split from the technique", card.Technique, card.Dimensions)
	}
	if card.Location != "Mauritshuis, The Hague" {
		t.Errorf("Location = %q, want the current-location record", card.Location)
	}
	if !strings.HasSuffix(card.Image, "/assets/images/no-image.png") {
		t.Errorf("Image = %q, want the no-image fallback for a record without an image", card.Image)
	}
	if card.Artwork == nil || card.Artwork.Id != artwork.Id {
		t.Error("card must carry the resolved artwork record")
	}
}

func TestResolveReceivedCardFallsBackToTheCatalogueLocation(t *testing.T) {
	f := newReceivedFixture(t)
	artwork := f.save(t, f.artworks, map[string]any{
		"published": true, "title": "Work", "author": f.artist(t).Id, "technique": "Oil on canvas",
		"comment": "<p>1902 · Synthetic Gallery, Test City · 101 x 201 cm</p>",
	})

	card, err := ResolveReceivedCard(f.app, f.postcardFor(t, artwork))
	if err != nil {
		t.Fatalf("ResolveReceivedCard() error = %v", err)
	}
	if card.Location != "Synthetic Gallery, Test City" || card.Dimensions != "101 x 201 cm" {
		t.Errorf("location, dimensions = %q, %q", card.Location, card.Dimensions)
	}
}

func TestResolveReceivedCardOmitsAbsentRecordDetails(t *testing.T) {
	f := newReceivedFixture(t)
	artwork := f.save(t, f.artworks, map[string]any{
		"published": true, "title": "Work", "author": f.artist(t).Id, "technique": "Fresco",
		"comment": "<p>Commentary without a catalogue summary.</p>",
	})

	card, err := ResolveReceivedCard(f.app, f.postcardFor(t, artwork))
	if err != nil {
		t.Fatalf("ResolveReceivedCard() error = %v", err)
	}
	if card.Dimensions != "" || card.Location != "" {
		t.Errorf("dimensions, location = %q, %q; want both omitted", card.Dimensions, card.Location)
	}
	if card.Technique != "Fresco" {
		t.Errorf("Technique = %q, want %q", card.Technique, "Fresco")
	}
}

func TestResolveReceivedCardRejectsUnavailableWorks(t *testing.T) {
	f := newReceivedFixture(t)
	unpublished := f.save(t, f.artworks, map[string]any{"published": false, "title": "Hidden", "author": f.artist(t).Id})
	if _, err := ResolveReceivedCard(f.app, f.postcardFor(t, unpublished)); !errors.Is(err, ErrArtworkUnavailable) {
		t.Errorf("unpublished artwork error = %v, want ErrArtworkUnavailable", err)
	}

	blank := f.save(t, f.artists, map[string]any{"name": "Unknown"})
	unattributed := f.save(t, f.artworks, map[string]any{"published": true, "title": "Work", "author": blank.Id})
	if _, err := ResolveReceivedCard(f.app, f.postcardFor(t, unattributed)); !errors.Is(err, ErrArtistIdentityUnavailable) {
		t.Errorf("incomplete artist identity error = %v, want ErrArtistIdentityUnavailable", err)
	}
}
