package url

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestDeliveryProfilesUseApprovedWidths(t *testing.T) {
	for _, test := range []struct {
		name    string
		profile DeliveryProfile
		want    string
		width   int
	}{
		{name: "itinerary tray", profile: DeliveryProfileItineraryTray, want: "120x0", width: 120},
		{name: "search row", profile: DeliveryProfileSearchRow, want: "200x0", width: 200},
		{name: "related timeline card", profile: DeliveryProfileRelatedTimelineCard, want: "400x0", width: 400},
		{name: "card and artist index", profile: DeliveryProfileCardAndArtistIndex, want: "500x0", width: 500},
		{name: "portrait record and work fallback", profile: DeliveryProfilePortraitRecordAndWorkFallback, want: "600x0", width: 600},
		{name: "postcard small Dual Mode plate", profile: DeliveryProfilePostcardSmallDualPlate, want: "700x0", width: 700},
		{name: "guided tour card", profile: DeliveryProfileGuidedTourCard, want: "800x0", width: 800},
		{name: "feature", profile: DeliveryProfileFeature, want: "900x0", width: 900},
		{name: "tour title plate", profile: DeliveryProfileTourTitlePlate, want: "1000x0", width: 1000},
		{name: "Dual Mode medium plate", profile: DeliveryProfileDualMediumPlate, want: "1100x0", width: 1100},
		{name: "artwork record tour page", profile: DeliveryProfileArtworkRecordTourPage, want: "1400x0", width: 1400},
		{name: "Dual Mode large plate", profile: DeliveryProfileDualLargePlate, want: "1600x0", width: 1600},
		{name: "viewer", profile: DeliveryProfileViewer, want: "2000x0", width: 2000},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := string(test.profile); got != test.want {
				t.Fatalf("profile = %q, want %q", got, test.want)
			}
			if got := test.profile.width(); got != test.width {
				t.Fatalf("profile width = %d, want %d", got, test.width)
			}
		})
	}
}

func TestGenerateDeliveryURLUsesThumbnailOnlyForWiderSources(t *testing.T) {
	for _, test := range []struct {
		name        string
		sourceWidth int
		want        string
	}{
		{name: "wider", sourceWidth: 601, want: "/api/files/artworks/artwork-id/work.jpg?thumb=600x0&token=token"},
		{name: "equal", sourceWidth: 600, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
		{name: "narrower", sourceWidth: 599, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
		{name: "zero", sourceWidth: 0, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
		{name: "invalid", sourceWidth: -1, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GenerateDeliveryURL("artworks", "artwork-id", "work.jpg", test.sourceWidth, DeliveryProfilePortraitRecordAndWorkFallback, "token")
			if got != test.want {
				t.Fatalf("GenerateDeliveryURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGenerateDeliveryURLReturnsOriginalForUnknownProfile(t *testing.T) {
	for _, profile := range []DeliveryProfile{
		"300x0",
		"600x1",
		"600",
		"invalid",
	} {
		got := GenerateDeliveryURL("artworks", "artwork-id", "work.jpg", 1000, profile, "")
		want := "/api/files/artworks/artwork-id/work.jpg"
		if got != want {
			t.Fatalf("GenerateDeliveryURL() = %q, want %q", got, want)
		}
	}
}

func TestImageURLRecordHelpersOnlyThumbnailWiderSources(t *testing.T) {
	for _, test := range []struct {
		name          string
		collection    string
		id            string
		filenameField string
		widthField    string
		filename      string
		width         int
		generate      func(*core.Record, DeliveryProfile, string) string
		want          string
	}{
		{name: "artwork wider", collection: "Artworks", id: "artwork-id", filenameField: "image", widthField: "image_width", filename: "work.jpg", width: 601, generate: GenerateArtworkImageURL, want: "/api/files/artworks/artwork-id/work.jpg?thumb=600x0&token=token"},
		{name: "artwork equal", collection: "Artworks", id: "artwork-id", filenameField: "image", widthField: "image_width", filename: "work.jpg", width: 600, generate: GenerateArtworkImageURL, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
		{name: "artwork narrower", collection: "Artworks", id: "artwork-id", filenameField: "image", widthField: "image_width", filename: "work.jpg", width: 599, generate: GenerateArtworkImageURL, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
		{name: "artwork zero width", collection: "Artworks", id: "artwork-id", filenameField: "image", widthField: "image_width", filename: "work.jpg", width: 0, generate: GenerateArtworkImageURL, want: "/api/files/artworks/artwork-id/work.jpg?token=token"},
		{name: "artwork missing filename", collection: "Artworks", id: "artwork-id", filenameField: "image", widthField: "image_width", width: 601, generate: GenerateArtworkImageURL, want: ""},
		{name: "portrait wider", collection: "Artists", id: "artist-id", filenameField: "portrait", widthField: "biography_image_width", filename: "portrait.jpg", width: 601, generate: GenerateArtistPortraitImageURL, want: "/api/files/artists/artist-id/portrait.jpg?thumb=600x0&token=token"},
		{name: "portrait equal", collection: "Artists", id: "artist-id", filenameField: "portrait", widthField: "biography_image_width", filename: "portrait.jpg", width: 600, generate: GenerateArtistPortraitImageURL, want: "/api/files/artists/artist-id/portrait.jpg?token=token"},
		{name: "portrait narrower", collection: "Artists", id: "artist-id", filenameField: "portrait", widthField: "biography_image_width", filename: "portrait.jpg", width: 599, generate: GenerateArtistPortraitImageURL, want: "/api/files/artists/artist-id/portrait.jpg?token=token"},
		{name: "portrait zero width", collection: "Artists", id: "artist-id", filenameField: "portrait", widthField: "biography_image_width", filename: "portrait.jpg", width: 0, generate: GenerateArtistPortraitImageURL, want: "/api/files/artists/artist-id/portrait.jpg?token=token"},
		{name: "portrait missing filename", collection: "Artists", id: "artist-id", filenameField: "portrait", widthField: "biography_image_width", width: 601, generate: GenerateArtistPortraitImageURL, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			record := core.NewRecord(core.NewBaseCollection(test.collection))
			record.Id = test.id
			record.Set(test.filenameField, test.filename)
			record.Set(test.widthField, test.width)
			got := test.generate(record, DeliveryProfilePortraitRecordAndWorkFallback, "token")
			if got != test.want {
				t.Fatalf("image URL = %q, want %q", got, test.want)
			}
		})
	}
}

func TestArtworkSourceURLIsOriginalRegardlessOfWidth(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("Artworks"))
	record.Id = "artwork-id"
	record.Set("image", "work.jpg")

	for _, width := range []int{3054, 2000, 1999, 1400, 0} {
		record.Set("image_width", width)
		if got := GenerateArtworkSourceURL(record); got != "/api/files/artworks/artwork-id/work.jpg" {
			t.Fatalf("source URL for width %d = %q, want original with no thumb query", width, got)
		}
	}

	record.Set("image", "")
	if got := GenerateArtworkSourceURL(record); got != "" {
		t.Fatalf("source URL for missing filename = %q, want empty", got)
	}
	if got := GenerateArtworkSourceURL(nil); got != "" {
		t.Fatalf("source URL for nil record = %q, want empty", got)
	}
}

func TestArtworkRecordURLsDistinguishSourceFromRenditions(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("Artworks"))
	record.Id = "artwork-id"
	record.Set("image", "work.jpg")
	record.Set("image_width", 3054)

	display := GenerateArtworkImageURL(record, DeliveryProfileArtworkRecordTourPage, "")
	viewer := GenerateArtworkImageURL(record, DeliveryProfileViewer, "")
	source := GenerateArtworkSourceURL(record)

	if display != "/api/files/artworks/artwork-id/work.jpg?thumb=1400x0" {
		t.Errorf("display = %q, want 1400px thumbnail", display)
	}
	if viewer != "/api/files/artworks/artwork-id/work.jpg?thumb=2000x0" {
		t.Errorf("viewer = %q, want 2000px thumbnail", viewer)
	}
	if source != "/api/files/artworks/artwork-id/work.jpg" {
		t.Errorf("source = %q, want original with no thumb query", source)
	}
}

func TestArtworkSourceURLHonestForNarrowSources(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("Artworks"))
	record.Id = "artwork-id"
	record.Set("image", "work.jpg")
	record.Set("image_width", 1500)

	// A source at or below the 2000px viewer profile must not be relabelled as a
	// rendition: both the viewer and the source resolve to the original.
	viewer := GenerateArtworkImageURL(record, DeliveryProfileViewer, "")
	source := GenerateArtworkSourceURL(record)
	if viewer != "/api/files/artworks/artwork-id/work.jpg" {
		t.Errorf("viewer for a <=2000 source = %q, want original", viewer)
	}
	if source != "/api/files/artworks/artwork-id/work.jpg" {
		t.Errorf("source = %q, want original", source)
	}
}
