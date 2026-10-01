package postcards

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/blackfyre/wga/internal/artworks"
	"github.com/blackfyre/wga/internal/utils"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

// ErrArtistIdentityUnavailable indicates that the postcard's artwork author
// lacks the authoritative identity fields needed to attribute the work.
var ErrArtistIdentityUnavailable = errors.New("postcard artwork artist identity is unavailable")

// ReceivedCard is the recipient-facing projection of a postcard's selected
// work: what the card shows and where its links lead.
type ReceivedCard struct {
	// Artwork is the published artwork record the card describes.
	Artwork          *core.Record
	Title            string
	ArtistFilingName string
	// Technique excludes the dimensions when the record repeats them there.
	Technique  string
	Dimensions string
	// Location is the work's holding location, preferring the current-location
	// record over the catalogue summary.
	Location string
	Comment  string
	Image    string
	// RecordURL is the canonical public artwork record.
	RecordURL string
	// ComposeURL opens a new postcard composer for the same work.
	ComposeURL string
}

// ResolveReceivedCard builds the received-card projection for a postcard. It
// returns ErrArtworkUnavailable when the selected work is no longer published
// and ErrArtistIdentityUnavailable when its author cannot be attributed.
func ResolveReceivedCard(app core.App, postcard *core.Record) (*ReceivedCard, error) {
	artwork, err := artworks.FindPublished(app, postcard.GetString("image_id"))
	if err != nil {
		return nil, ErrArtworkUnavailable
	}
	if errs := app.ExpandRecord(artwork, []string{"author"}, nil); len(errs) > 0 {
		return nil, fmt.Errorf("expand postcard artwork author: %v", errs)
	}
	author := artwork.ExpandedOne("author")
	if !HasCompleteArtistIdentity(author) {
		return nil, ErrArtistIdentityUnavailable
	}

	location, dimensions := artworks.LocationAndDimensions(artwork.GetString("comment"))
	if current := artworks.CurrentLocation(app, artwork); current != "" {
		location = current
	}
	technique := strings.TrimSpace(artwork.GetString("technique"))
	if dimensions != "" {
		technique = strings.TrimSpace(strings.TrimSuffix(technique, ", "+dimensions))
	}
	image := utils.AssetUrl("/assets/images/no-image.png")
	if artwork.GetString("image") != "" {
		image = urlutils.GenerateArtworkImageURL(artwork, urlutils.DeliveryProfilePostcardSmallDualPlate, "")
	}

	return &ReceivedCard{
		Artwork:          artwork,
		Title:            artwork.GetString("title"),
		ArtistFilingName: author.GetString("filing_name"),
		Technique:        technique,
		Dimensions:       dimensions,
		Location:         location,
		Comment:          artwork.GetString("comment"),
		Image:            image,
		RecordURL: urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
			ArtistName:   author.GetString("name"),
			ArtistId:     author.Id,
			ArtworkTitle: artwork.GetString("title"),
			ArtworkId:    artwork.Id,
		}),
		ComposeURL: "/postcard/send?" + url.Values{"awid": {artwork.Id}}.Encode(),
	}, nil
}

// HasCompleteArtistIdentity reports whether an artist record carries both
// authoritative identity fields. Prior-bootstrap artists have blank fields and
// must fail closed rather than render reconstructed or blank identity.
func HasCompleteArtistIdentity(artist *core.Record) bool {
	return artist != nil &&
		strings.TrimSpace(artist.GetString("filing_name")) != "" &&
		strings.TrimSpace(artist.GetString("short_name")) != ""
}
