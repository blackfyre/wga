package artists

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// RenderArtistContent renders the content of an artist by generating a DTO (Data Transfer Object) that contains
// information about the artist, their works, and JSON-LD metadata. It takes the PocketBase application instance,
// the Echo context, and the artist record as input parameters. It returns the DTO representing the artist content
// and an error if any occurred during the process.
func RenderArtistContent(app *pocketbase.PocketBase, c *core.RequestEvent, artist *core.Record, hxTarget string) (dto.Artist, error) {
	id := artist.GetString("id")
	expectedSlug := utils.GenerateArtistSlug(artist)

	works, err := findArtworksByAuthorId(app, id)

	if err != nil {
		app.Logger().Error("Error finding artworks: ", "error", err.Error())
		return dto.Artist{}, errs.ErrArtistNotFound
	}

	schools := utils.RenderSchoolNames(app, artist.GetStringSlice("school"))

	content := dto.Artist{
		Name: artist.GetString("name"),
		Bio:  artist.GetString("bio"),
		BioExcerpt: normalizedBioExcerpt(BioExcerptDTO{
			YearOfBirth:       artist.GetInt("year_of_birth"),
			ExactYearOfBirth:  artist.GetString("exact_year_of_birth"),
			PlaceOfBirth:      artist.GetString("place_of_birth"),
			KnownPlaceOfBirth: artist.GetString("known_place_of_birth"),
			YearOfDeath:       artist.GetInt("year_of_death"),
			ExactYearOfDeath:  artist.GetString("exact_year_of_death"),
			PlaceOfDeath:      artist.GetString("place_of_death"),
			KnownPlaceOfDeath: artist.GetString("known_place_of_death"),
		}),
		Schools:    schools,
		Profession: artist.GetString("profession"),
		Works:      dto.ImageGrid{},
		Url:        "/artists/" + expectedSlug,
		HxTarget:   hxTarget,
	}

	JsonLd := jsonld.ArtistJsonLd(artist)

	marshalled, err := json.Marshal(JsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artist jsonld for"+id, "error", err.Error())
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	app.Logger().Debug("Rendering artist content", "artistId", id, "artistName", artist.GetString("name"), "worksCount", len(works))

	for _, w := range works {

		artJsonLd := jsonld.ArtworkJsonLd(w, artist)

		marshalled, err := json.Marshal(artJsonLd)

		if err != nil {
			app.Logger().Error("Error marshalling artwork jsonld for"+w.GetString("id"), "error", err.Error())
		}

		content.Works = append(content.Works, dto.Image{
			Id:        w.GetString("id"),
			Title:     w.GetString("title"),
			Comment:   w.GetString("comment"),
			Technique: w.GetString("technique"),
			Image:     url.GenerateFileUrl("artworks", w.GetString("id"), w.GetString("image"), ""),
			Thumb:     url.GenerateThumbUrl("artworks", w.GetString("id"), w.GetString("image"), "320x240", ""),
			Url:       utils.AssetUrl(utils.Slugify(w.GetString("title")) + "-" + w.GetString("id")),
			Jsonld:    fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled),
			HxTarget:  hxTarget,
		})
	}

	return content, nil
}

// processArtist is a handler function that processes the artist request.
// It retrieves the artist information from the database, including their works,
// and renders the HTML template with the artist data.
// If the artist is not found, it returns a not found error.
// If the rendering fails, it returns an error.
// It also sets the HX-Push-Url header to enable Htmx push for the artist page.
func processArtist(c *core.RequestEvent, app *pocketbase.PocketBase) error {
	slug := c.Request.PathValue("name")

	id := utils.ExtractIdFromString(slug)

	fullUrl := utils.GenerateCurrentPageUrl(c)
	artist, err := app.FindRecordById("artists", id)

	app.Logger().Info("Processing artist", "slug", slug)

	if err != nil {
		app.Logger().Error("Artist not found: ", slug, err)
		return utils.NotFoundError(c)
	}

	expectedSlug := utils.GenerateArtistSlug(artist)

	if slug != expectedSlug {
		return c.Redirect(http.StatusMovedPermanently, "/artists/"+expectedSlug)
	}

	content, err := RenderArtistContent(app, c, artist, "#mc-area")

	if err != nil {
		return err
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", content.Name, content.BioExcerpt))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, content.Bio)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, fullUrl)
	if len(content.Works) > 0 {
		ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, utils.AssetUrl(content.Works[0].Image))
	}

	// c.Response.Header().Set("HX-Push-Url", fullUrl)

	var buff bytes.Buffer

	err = pages.ArtistPage(content).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artist page", "error", err.Error())

		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}
