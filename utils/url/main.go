package url

import (
	"fmt"
    "github.com/blackfyre/wga/utils"
    "github.com/pocketbase/pocketbase/models"
    "net/url"
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
    ArtistName string
	ArtistId string
	ArtworkTitle string
	ArtworkId string
	BaseUrl string
}

func GenerateArtworkUrl (d ArtworkUrlDTO) string {
	return fmt.Sprintf("%v/artists/%v-%v/artworks/%v-%v", d.BaseUrl, utils.Slugify(d.ArtistName), d.ArtistId, utils.Slugify(d.ArtistName), d.ArtworkId)
}

func GenerateArtistUrlFromRecord(r *models.Record) string {
	return  GenerateArtistUrl(ArtistUrlDTO{
		ArtistName: r.GetString("name"),
		ArtistId: r.Id,
	})
}

type ArtistUrlDTO struct {
	ArtistName string
	ArtistId string
	BaseUrl string
}

func GenerateArtistUrl(d ArtistUrlDTO) string {
	return fmt.Sprintf("%v/artists/%v-%v", d.BaseUrl, utils.Slugify(d.ArtistName), d.ArtistId)
}