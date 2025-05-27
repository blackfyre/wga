package url

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
)

func GenerateFileUrl(collection string, collectionId string, fileName string, token string) string {

	return fmt.Sprintf(
		"/api/files/%s/%s/%s?token=%s",
		collection,
		collectionId,
		fileName,
		url.QueryEscape(token),
	)
}

func GenerateThumbUrl(collection string, collectionId string, fileName string, thumbSize string, token string) string {

	return fmt.Sprintf(
		"/api/files/%s/%s/%s?token=%s&thumb=%s",
		collection,
		collectionId,
		fileName,
		url.QueryEscape(token),
		thumbSize,
	)
}

type ArtworkUrlDTO struct {
	ArtistName   string
	ArtistId     string
	ArtworkTitle string
	ArtworkId    string
}

func GenerateFullArtworkUrl(d ArtworkUrlDTO) string {
	return fmt.Sprintf("/artists/%v-%v/artworks/%v-%v", utils.Slugify(d.ArtistName), d.ArtistId, utils.Slugify(d.ArtistName), d.ArtworkId)
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

func GetRequiredQueryParam(c echo.Context, param string) (string, error) {
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
