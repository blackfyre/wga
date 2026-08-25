package url

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
)

const (
	DeliveryProfileItineraryTray                 DeliveryProfile = "120x0"
	DeliveryProfileSearchRow                     DeliveryProfile = "200x0"
	DeliveryProfileRelatedTimelineCard           DeliveryProfile = "400x0"
	DeliveryProfileCardAndArtistIndex            DeliveryProfile = "500x0"
	DeliveryProfilePortraitRecordAndWorkFallback DeliveryProfile = "600x0"
	DeliveryProfilePostcardSmallDualPlate        DeliveryProfile = "700x0"
	DeliveryProfileGuidedTourCard                DeliveryProfile = "800x0"
	DeliveryProfileFeature                       DeliveryProfile = "900x0"
	DeliveryProfileTourTitlePlate                DeliveryProfile = "1000x0"
	DeliveryProfileDualMediumPlate               DeliveryProfile = "1100x0"
	DeliveryProfileArtworkRecordTourPage         DeliveryProfile = "1400x0"
	DeliveryProfileDualLargePlate                DeliveryProfile = "1600x0"
	DeliveryProfileViewer                        DeliveryProfile = "2000x0"
)

// DeliveryProfile is an approved source-eligible PocketBase thumbnail size.
type DeliveryProfile string

func GenerateFileUrl(collection string, collectionId string, fileName string, token string) string {

	url := fmt.Sprintf(
		"/api/files/%s/%s/%s",
		collection,
		collectionId,
		fileName,
	)

	if token != "" {
		url += fmt.Sprintf("?token=%s", token)
	}

	return url
}

func generateThumbURL(collection string, collectionId string, fileName string, thumbSize string, token string) string {

	url := fmt.Sprintf(
		"/api/files/%s/%s/%s?thumb=%s",
		collection,
		collectionId,
		fileName,
		thumbSize,
	)

	if token != "" {
		url += fmt.Sprintf("&token=%s", token)
	}

	return url
}

// GenerateDeliveryURL returns the original file unless the source is wider than
// the assigned profile. This prevents PocketBase from generating an upscale.
func GenerateDeliveryURL(collection string, collectionID string, fileName string, sourceWidth int, profile DeliveryProfile, token string) string {
	if fileName == "" {
		return ""
	}

	targetWidth := profile.width()
	if sourceWidth <= targetWidth || sourceWidth <= 0 || targetWidth <= 0 {
		return GenerateFileUrl(collection, collectionID, fileName, token)
	}

	return generateThumbURL(collection, collectionID, fileName, string(profile), token)
}

func (p DeliveryProfile) width() int {
	switch p {
	case DeliveryProfileItineraryTray:
		return 120
	case DeliveryProfileSearchRow:
		return 200
	case DeliveryProfileRelatedTimelineCard:
		return 400
	case DeliveryProfileCardAndArtistIndex:
		return 500
	case DeliveryProfilePortraitRecordAndWorkFallback:
		return 600
	case DeliveryProfilePostcardSmallDualPlate:
		return 700
	case DeliveryProfileGuidedTourCard:
		return 800
	case DeliveryProfileFeature:
		return 900
	case DeliveryProfileTourTitlePlate:
		return 1000
	case DeliveryProfileDualMediumPlate:
		return 1100
	case DeliveryProfileArtworkRecordTourPage:
		return 1400
	case DeliveryProfileDualLargePlate:
		return 1600
	case DeliveryProfileViewer:
		return 2000
	default:
		return 0
	}
}

func GenerateArtworkImageURL(r *core.Record, profile DeliveryProfile, token string) string {
	if r == nil {
		return ""
	}

	return GenerateDeliveryURL(
		constants.CollectionArtworks,
		r.Id,
		r.GetString("image"),
		r.GetInt("image_width"),
		profile,
		token,
	)
}

// GenerateArtworkSourceURL returns the original artwork file URL with no
// thumbnail query. It is used only for the deliberate source-file download, so
// a record's original filename always resolves to the source itself rather than
// to a source-eligible rendition.
func GenerateArtworkSourceURL(r *core.Record) string {
	if r == nil {
		return ""
	}

	filename := r.GetString("image")
	if filename == "" {
		return ""
	}

	return GenerateFileUrl(constants.CollectionArtworks, r.Id, filename, "")
}

type ArtworkUrlDTO struct {
	ArtistName   string
	ArtistId     string
	ArtworkTitle string
	ArtworkId    string
}

func GenerateFullArtworkUrl(d ArtworkUrlDTO) string {
	return fmt.Sprintf("/artists/%v-%v/%v-%v", utils.Slugify(d.ArtistName), d.ArtistId, utils.Slugify(d.ArtworkTitle), d.ArtworkId)
}

func GenerateArtworkUrl(d ArtworkUrlDTO) string {
	return fmt.Sprintf("/artworks/%v-%v", utils.Slugify(d.ArtworkTitle), d.ArtworkId)
}

func GenerateArtistUrlFromRecord(r *core.Record) string {
	return GenerateArtistUrl(ArtistUrlDTO{
		ArtistName: r.GetString("name"),
		ArtistId:   r.GetString("id"),
	})
}

func GenerateArtistPortraitURL(r *core.Record) string {
	portrait := r.GetString("portrait")
	if portrait == "" {
		return ""
	}

	return GenerateFileUrl(constants.CollectionArtists, r.Id, portrait, "")
}

func GenerateArtistPortraitImageURL(r *core.Record, profile DeliveryProfile, token string) string {
	if r == nil {
		return ""
	}

	return GenerateDeliveryURL(
		constants.CollectionArtists,
		r.Id,
		r.GetString("portrait"),
		r.GetInt("biography_image_width"),
		profile,
		token,
	)
}

type ArtistUrlDTO struct {
	ArtistName string
	ArtistId   string
	BaseUrl    string
}

func GenerateArtistUrl(d ArtistUrlDTO) string {
	return fmt.Sprintf("%v/artists/%v-%v", d.BaseUrl, utils.Slugify(d.ArtistName), d.ArtistId)
}

func GenerateDualModeUrl() url.URL {
	return url.URL{
		Path: "/dual-mode",
	}
}

func GetRequiredQueryParam(c *echo.Context, param string) (string, error) {
	p := c.QueryParam(param)

	if p == "" {
		return "", fmt.Errorf("missing required query parameter: %v", param)
	}

	return p, nil
}

func GenerateCurrentPageUrl(c *core.RequestEvent) string {
	if c == nil || c.Request == nil {
		return ""
	}

	var urlParts []string

	if c.Request.URL.Scheme != "" && c.Request.URL.Host != "" {
		urlParts = append(urlParts, c.Request.URL.Scheme+"://"+c.Request.URL.Host)
	}
	if c.Request.URL.Path != "" {
		urlParts = append(urlParts, c.Request.URL.Path)
	}
	if c.Request.URL.Fragment != "" {
		urlParts = append(urlParts, "#"+c.Request.URL.Fragment)
	}
	if c.Request.URL.RawQuery != "" {
		urlParts = append(urlParts, "?"+c.Request.URL.RawQuery)
	}

	return strings.Join(urlParts, "")
}
