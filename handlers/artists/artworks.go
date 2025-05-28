package artists

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/errs"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/jsonld"
	"github.com/blackfyre/wga/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// processArtwork processes the artwork based on the given context and PocketBase application.
// It retrieves the artist and artwork information, generates JSON-LD content, and renders the HTML template.
// If the artwork is found in the cache, it retrieves the HTML from the cache. Otherwise, it fetches the data from the database,
// generates the HTML, and stores it in the cache for future use.
// It also sets the "HX-Push-Url" header in the response.
// Parameters:
// - c: The echo.Context object representing the HTTP request and response.
// - app: The PocketBase application instance.
// Returns:
// - An error if any error occurs during the processing, or nil if the processing is successful.
func processArtwork(c *core.RequestEvent, app *pocketbase.PocketBase) error {
	artistSlug := c.Request.PathValue("name")
	artworkSlug := c.Request.PathValue("awid")

	// Split the slug on the last dash and use the last part as the artist id
	artistSlugParts := strings.Split(artistSlug, "-")
	artistId := artistSlugParts[len(artistSlugParts)-1]

	artist, err := app.FindRecordById("artists", artistId)

	// If the artist is not found, return a not found error
	if err != nil {
		app.Logger().Error("Artist not found: ", artistSlug, err)
		return errs.ErrArtistNotFound
	}

	// Generate the expected slug for the artist
	expectedArtistSlug := artist.GetString("slug") + "-" + artist.GetString("id")

	// Split the slug on the last dash and use the last part as the artwork id
	artworkSlugParts := strings.Split(artworkSlug, "-")
	artworkId := artworkSlugParts[len(artworkSlugParts)-1]

	// find the artwork by id
	aw, err := app.FindRecordById("artworks", artworkId)

	if err != nil {
		app.Logger().Error("Error finding artwork: ", artworkSlug, err)
		return errs.ErrArtworkNotFound
	}

	// Generate the expected slug for the artwork
	expectedArtworkSlug := utils.Slugify(aw.GetString("title")) + "-" + aw.GetString("id")

	expectedPageUrl := "/artists/" + expectedArtistSlug + "/" + expectedArtworkSlug

	// Redirect to the correct URL if either slug is not correct
	if artistSlug != expectedArtistSlug || artworkSlug != expectedArtworkSlug {
		return c.Redirect(http.StatusMovedPermanently, expectedPageUrl)
	}

	var img dto.Image

	img.Id = aw.GetString("id")
	img.Title = aw.GetString("title")
	img.Comment = aw.GetString("comment")
	img.Technique = aw.GetString("technique")
	if aw.GetString("image") != "" {
		img.Image = url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), "")
		img.Thumb = url.GenerateThumbUrl("artworks", aw.GetString("id"), aw.GetString("image"), "320x240", "")
	} else {
		img.Image = utils.AssetUrl("/assets/images/no-image.png")
		img.Thumb = utils.AssetUrl("/assets/images/no-image.png")
	}

	content := dto.Artwork{
		Id:        aw.GetString("id"),
		Title:     aw.GetString("title"),
		Comment:   aw.GetString("comment"),
		Technique: aw.GetString("technique"),
		Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
			ArtistName:   artist.GetString("name"),
			ArtistId:     artist.Id,
			ArtworkId:    aw.GetString("id"),
			ArtworkTitle: aw.GetString("title"),
		}),
		Image: img,
		Artist: dto.Artist{
			Id:         artist.GetString("id"),
			Name:       artist.GetString("name"),
			Bio:        artist.GetString("bio"),
			Profession: artist.GetString("profession"),
			Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
				ArtistId:   artist.GetString("id"),
				ArtistName: artist.GetString("name"),
			}),
		},
	}

	school := artist.GetStringSlice("school")

	var schoolCollector []string

	for _, s := range school {
		r, err := app.FindRecordById("schools", s)

		if err != nil {
			app.Logger().Error("school not found", "error", err.Error())
			continue
		}

		schoolCollector = append(schoolCollector, r.GetString("name"))

		content.Artist.Schools = strings.Join(schoolCollector, ", ")

	}

	jsonLd := jsonld.ArtworkJsonLd(aw, artist)

	marshalled, err := json.Marshal(jsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artwork jsonld for"+aw.GetString("id"), "error", err.Error())
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", content.Title, content.Artist.Name))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, content.Comment)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, expectedPageUrl)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, utils.AssetUrl(content.Image.Image))

	c.Response.Header().Set("HX-Push-Url", expectedPageUrl)

	var buff bytes.Buffer

	err = pages.ArtworkPage(content).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artwork page", "error", err.Error())
		return c.String(http.StatusInternalServerError, "failed to render response template")
	}

	return c.HTML(http.StatusOK, buff.String())
}

func RenderArtworkContent(app *pocketbase.PocketBase, c *core.RequestEvent, artwork *core.Record, hxTarget string) (dto.Artwork, error) {

	artistId := cmp.Or(artwork.GetStringSlice("author")[0], "")

	var artworkUrl string

	content := dto.Artwork{
		Id:        artwork.GetString("id"),
		Title:     artwork.GetString("title"),
		Comment:   artwork.GetString("comment"),
		Technique: artwork.GetString("technique"),
		Image: dto.Image{
			Id:        artwork.GetString("id"),
			Title:     artwork.GetString("title"),
			Comment:   artwork.GetString("comment"),
			Technique: artwork.GetString("technique"),
			Image:     url.GenerateFileUrl("artworks", artwork.GetString("id"), artwork.GetString("image"), ""),
		},
		HxTarget: hxTarget,
	}

	if artistId != "" {
		var artist *core.Record

		artist, err := app.FindRecordById("Artists", artistId)

		if err != nil {
			app.Logger().Error(fmt.Sprintf("Error finding artist (%s) related to artwork (%s)", artistId, artwork.Id), "error", err.Error())
		}

		artworkUrl = url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
			ArtistName:   artist.GetString("name"),
			ArtistId:     artist.GetString("id"),
			ArtworkId:    artwork.GetString("id"),
			ArtworkTitle: artwork.GetString("title"),
		})

		content.Artist = dto.Artist{
			Id:         artist.GetString("id"),
			Name:       artist.GetString("name"),
			Bio:        artist.GetString("bio"),
			Profession: artist.GetString("profession"),
			Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
				ArtistId:   artist.GetString("id"),
				ArtistName: artist.GetString("name"),
			}),
		}

	} else {
		artworkUrl = url.GenerateArtworkUrl(url.ArtworkUrlDTO{
			ArtworkId:    artwork.GetString("id"),
			ArtworkTitle: artwork.GetString("title"),
		})
	}

	// Set the URL for the artwork
	content.Url = artworkUrl

	return content, nil
}
