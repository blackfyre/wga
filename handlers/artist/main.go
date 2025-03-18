package artist

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

const (
	Yes           string = "yes"
	No            string = "no"
	NotApplicable string = "n/a"
)

type BioExcerptDTO struct {
	YearOfBirth       int
	ExactYearOfBirth  string
	PlaceOfBirth      string
	KnownPlaceOfBirth string
	YearOfDeath       int
	ExactYearOfDeath  string
	PlaceOfDeath      string
	KnownPlaceOfDeath string
}

// generateBioSection generates a bio section based on the provided parameters.
// It takes a prefix string, year integer, exactYear string, place string, and knownPlace string as input.
// It returns a string representing the generated bio section.
func generateBioSection(prefix string, year int, exactYear string, place string, knownPlace string) string {
	var c []string

	c = append(c, prefix)
	y := strconv.Itoa(year)

	if exactYear == No {
		y = "~" + y
	}

	c = append(c, y)

	if knownPlace == No {
		place += "?"
	}

	c = append(c, place)

	return strings.Join(c, " ")
}

// normalizedBioExcerpt returns a normalized biography excerpt for the given record.
// It includes the person's year and place of birth and death (if available).
func normalizedBioExcerpt(d BioExcerptDTO) string {
	var s []string

	s = append(s, generateBioSection("b.", d.YearOfBirth, d.ExactYearOfBirth, d.PlaceOfBirth, d.KnownPlaceOfBirth))
	s = append(s, generateBioSection("d.", d.YearOfDeath, d.ExactYearOfDeath, d.PlaceOfDeath, d.KnownPlaceOfDeath))

	return strings.Join(s, ", ")
}

// findArtworksByAuthorId retrieves a list of artworks by the given author ID.
// It uses the provided PocketBase instance to query the database and returns
// a slice of Record pointers and an error, if any.
func findArtworksByAuthorId(app *pocketbase.PocketBase, authorId string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("artworks", "author = '"+authorId+"'", "+title", 100, 0)
}

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

	c.Response.Header().Set("HX-Push-Url", fullUrl)

	var buff bytes.Buffer

	err = pages.ArtistPage(content).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artist page", "error", err.Error())

		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

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

	// split the slug on the last dash and use the last part as the artist id
	artistSlugParts := strings.Split(artistSlug, "-")
	artistId := artistSlugParts[len(artistSlugParts)-1]

	artist, err := app.FindRecordById("artists", artistId)

	// if the artist is not found, return a not found error
	if err != nil {
		app.Logger().Error("Artist not found: ", artistSlug, err)
		return errs.ErrArtistNotFound
	}

	// generate the expected slug for the artist
	expectedArtistSlug := artist.GetString("slug") + "-" + artist.GetString("id")

	// split the slug on the last dash and use the last part as the artwork id
	artworkSlugParts := strings.Split(artworkSlug, "-")
	artworkId := artworkSlugParts[len(artworkSlugParts)-1]

	// find the artwork by id
	aw, err := app.FindRecordById("artworks", artworkId)

	if err != nil {
		app.Logger().Error("Error finding artwork: ", artworkSlug, err)
		return errs.ErrArtworkNotFound
	}

	// generate the expected slug for the artwork
	expectedArtworkSlug := utils.Slugify(aw.GetString("title")) + "-" + aw.GetString("id")

	// redirect to the correct URL if either slug is not correct
	if artistSlug != expectedArtistSlug || artworkSlug != expectedArtworkSlug {
		return c.Redirect(http.StatusMovedPermanently, "/artists/"+expectedArtistSlug+"/"+expectedArtworkSlug)
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
		Image: dto.Image{
			Id:        aw.GetString("id"),
			Title:     aw.GetString("title"),
			Comment:   aw.GetString("comment"),
			Technique: aw.GetString("technique"),
			Image:     url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), ""),
			Thumb:     url.GenerateThumbUrl("artworks", aw.GetString("id"), aw.GetString("image"), "320x240", ""),
		},
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

	fullUrl := c.Request.URL.Scheme + "://" + c.Request.URL.Host + c.Request.URL.String()

	school := artist.GetStringSlice("school")

	var schoolCollector []string

	for _, s := range school {
		r, err := app.FindRecordById("schools", s)

		if err != nil {
			app.Logger().Error("school not found", "error", err.Error())
			continue
		}

		schoolCollector = append(schoolCollector, r.GetString("name"))

	}

	jsonLd := jsonld.ArtworkJsonLd(aw, artist)

	marshalled, err := json.Marshal(jsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artwork jsonld for"+aw.GetString("id"), "error", err.Error())
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", content.Title, content.Artist.Name))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, content.Comment)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, utils.AssetUrl(content.Image.Image))

	c.Response.Header().Set("HX-Push-Url", fullUrl)

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
	var artist *core.Record

	if artistId != "" {
		artist, err := app.FindRecordById("Artists", artistId)

		if err != nil {
			app.Logger().Error(fmt.Sprintf("Error finding artist (%s) related to artwork (%s)", artistId, &artwork.Id), "error", err.Error())
		}

		artworkUrl = url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
			ArtistName:   artist.GetString("name"),
			ArtistId:     artist.GetString("id"),
			ArtworkId:    artwork.GetString("id"),
			ArtworkTitle: artwork.GetString("title"),
		})

	} else {
		artworkUrl = url.GenerateArtworkUrl(url.ArtworkUrlDTO{
			ArtworkId:    artwork.GetString("id"),
			ArtworkTitle: artwork.GetString("title"),
		})
	}

	content := dto.Artwork{
		Id:        artwork.GetString("id"),
		Title:     artwork.GetString("title"),
		Comment:   artwork.GetString("comment"),
		Technique: artwork.GetString("technique"),
		Url:       artworkUrl,
		Image: dto.Image{
			Id:        artwork.GetString("id"),
			Title:     artwork.GetString("title"),
			Comment:   artwork.GetString("comment"),
			Technique: artwork.GetString("technique"),
			Image:     url.GenerateFileUrl("artworks", artwork.GetString("id"), artwork.GetString("image"), ""),
		},
		HxTarget: hxTarget,
	}

	// Check if artist pointer is nil
	if artist != nil {
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
	}

	return content, nil
}

// RegisterHandlers registers the artist routes to the PocketBase app.
// It adds two routes to the app router:
// 1. GET /artists/:name - returns the artist page with the given name
// 2. GET /artists/:name/:awid - returns the artwork page with the given name and artwork id
// It also caches the HTML response for each route to improve performance.
func RegisterHandlers(app *pocketbase.PocketBase) {

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		ag := se.Router.Group("artists")

		ag.GET("/:name", func(e *core.RequestEvent) error {
			return processArtist(e, app)
		})

		ag.GET("/:name/:awid", func(e *core.RequestEvent) error {
			return processArtwork(e, app)
		})
		return nil
	})
}
