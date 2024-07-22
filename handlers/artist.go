package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	wgaModels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/jsonld"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

const (
	Yes           string = "yes"
	No            string = "no"
	NotApplicable string = "n/a"
)

// normalizedBioExcerpt returns a normalized biography excerpt for the given record.
// It includes the person's year and place of birth and death (if available).
func normalizedBioExcerpt(r *models.Record) string {
	var s []string

	yob := r.GetInt("year_of_birth")
	eyob := r.GetString("exact_year_of_birth")
	pob := r.GetString("place_of_birth")
	kpob := r.GetString("known_place_of_birth")

	yod := r.GetInt("year_of_death")
	eyod := r.GetString("exact_year_of_death")
	pod := r.GetString("place_of_death")
	kpod := r.GetString("known_place_of_death")

	if yob > 0 {

		var c []string

		prefix := "b."

		c = append(c, prefix)
		y := strconv.Itoa(yob)

		if eyob == No {
			y = "~" + y
		}

		c = append(c, y)

		if kpob == No {
			pob = pob + "?"
		}

		c = append(c, pob)

		s = append(s, strings.Join(c, " "))

	}

	if yod > 0 {

		var c []string

		prefix := "d."

		c = append(c, prefix)
		y := strconv.Itoa(yod)

		if eyod == No {
			y = "~" + y
		}

		c = append(c, y)

		if kpod == No {
			pod = pod + "?"
		}

		c = append(c, pod)

		s = append(s, strings.Join(c, " "))

	}

	return strings.Join(s, ", ")
}

// findArtworksByAuthorId retrieves a list of artworks by the given author ID.
// It uses the provided PocketBase instance to query the database and returns
// a slice of Record pointers and an error, if any.
func findArtworksByAuthorId(app *pocketbase.PocketBase, authorId string) ([]*models.Record, error) {
	return app.Dao().FindRecordsByFilter("artworks", "author = '"+authorId+"'", "+title", 100, 0)
}

// renderSchoolNames takes an instance of PocketBase and a slice of school IDs,
// and returns a string containing the names of the schools corresponding to the given IDs.
// If a school is not found, it logs an error and continues to the next ID.
func renderSchoolNames(app *pocketbase.PocketBase, schoolIds []string) string {
	var schoolCollector []string

	for _, s := range schoolIds {
		r, err := app.Dao().FindRecordById("schools", s)

		if err != nil {
			app.Logger().Error("school not found", err)
			continue
		}

		schoolCollector = append(schoolCollector, r.GetString("name"))

	}

	return strings.Join(schoolCollector, ", ")
}

// renderArtistContent renders the content of an artist by generating a DTO (Data Transfer Object) that contains
// information about the artist, their works, and JSON-LD metadata. It takes the PocketBase application instance,
// the Echo context, and the artist record as input parameters. It returns the DTO representing the artist content
// and an error if any occurred during the process.
func renderArtistContent(app *pocketbase.PocketBase, c echo.Context, artist *models.Record) (dto.Artist, error) {
	id := artist.GetId()
	expectedSlug := generateArtistSlug(artist)

	works, err := findArtworksByAuthorId(app, id)

	if err != nil {
		app.Logger().Error("Error finding artworks: ", err)

		return dto.Artist{}, utils.NotFoundError(c)
	}

	schools := renderSchoolNames(app, artist.GetStringSlice("school"))

	content := dto.Artist{
		Name:       artist.GetString("name"),
		Bio:        artist.GetString("bio"),
		BioExcerpt: normalizedBioExcerpt(artist),
		Schools:    schools,
		Profession: artist.GetString("profession"),
		Works:      dto.ImageGrid{},
		Url:        "/artists/" + expectedSlug,
	}

	artistReferenceModel := &wgaModels.Artist{
		Id:           artist.GetId(),
		Name:         artist.GetString("name"),
		Slug:         artist.GetString("slug"),
		Bio:          artist.GetString("bio"),
		YearOfBirth:  artist.GetInt("year_of_birth"),
		YearOfDeath:  artist.GetInt("year_of_death"),
		PlaceOfBirth: artist.GetString("place_of_birth"),
		PlaceOfDeath: artist.GetString("place_of_death"),
		Published:    artist.GetBool("published"),
		School:       schools,
		Profession:   artist.GetString("profession"),
	}

	JsonLd := jsonld.ArtistJsonLd(artistReferenceModel, c)

	marshalled, err := json.Marshal(JsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artist jsonld for"+id, err)
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	for _, w := range works {

		artJsonLd := jsonld.ArtworkJsonLd(w, artistReferenceModel, c)

		marshalled, err := json.Marshal(artJsonLd)

		if err != nil {
			app.Logger().Error("Error marshalling artwork jsonld for"+w.GetId(), err)
		}

		content.Works = append(content.Works, dto.Image{
			Id:        w.GetId(),
			Title:     w.GetString("title"),
			Comment:   w.GetString("comment"),
			Technique: w.GetString("technique"),
			Image:     url.GenerateFileUrl("artworks", w.GetString("id"), w.GetString("image"), ""),
			Thumb:     url.GenerateThumbUrl("artworks", w.GetString("id"), w.GetString("image"), "320x240", ""),
			Url:       c.Request().URL.String() + "/" + utils.Slugify(w.GetString("title")) + "-" + w.GetString("id"),
			Jsonld:    fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled),
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
func processArtist(c echo.Context, app *pocketbase.PocketBase) error {
	slug := c.PathParam("name")

	id := utils.ExtractIdFromString(slug)

	fullUrl := generateCurrentPageUrl(c)
	artist, err := app.Dao().FindRecordById("artists", id)

	if err != nil {
		app.Logger().Error("Artist not found: ", slug, err)
		return utils.NotFoundError(c)
	}

	expectedSlug := generateArtistSlug(artist)

	if slug != expectedSlug {
		return c.Redirect(http.StatusMovedPermanently, "/artists/"+expectedSlug)
	}

	content, err := renderArtistContent(app, c, artist)

	if err != nil {
		return err
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", content.Name, content.BioExcerpt))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, content.Bio)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, fullUrl)
	if len(content.Works) > 0 {
		ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, c.Scheme()+"://"+c.Request().Host+content.Works[0].Image)
	}

	c.Response().Header().Set("HX-Push-Url", fullUrl)
	err = pages.ArtistPage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artist page", err)

		return utils.ServerFaultError(c)
	}

	return nil
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
func processArtwork(c echo.Context, app *pocketbase.PocketBase) error {
	artistSlug := c.PathParam("name")
	artworkSlug := c.PathParam("awid")

	// split the slug on the last dash and use the last part as the artist id
	artistSlugParts := strings.Split(artistSlug, "-")
	artistId := artistSlugParts[len(artistSlugParts)-1]

	artist, err := app.Dao().FindRecordById("artists", artistId)

	// if the artist is not found, return a not found error
	if err != nil {
		app.Logger().Error("Artist not found: ", artistSlug, err)
		return utils.NotFoundError(c)
	}

	// generate the expected slug for the artist
	expectedArtistSlug := artist.GetString("slug") + "-" + artist.GetString("id")

	// split the slug on the last dash and use the last part as the artwork id
	artworkSlugParts := strings.Split(artworkSlug, "-")
	artworkId := artworkSlugParts[len(artworkSlugParts)-1]

	// find the artwork by id
	aw, err := app.Dao().FindRecordById("artworks", artworkId)

	if err != nil {
		app.Logger().Error("Error finding artwork: ", artworkSlug, err)
		return utils.NotFoundError(c)
	}

	// generate the expected slug for the artwork
	expectedArtworkSlug := utils.Slugify(aw.GetString("title")) + "-" + aw.GetString("id")

	// redirect to the correct URL if either slug is not correct
	if artistSlug != expectedArtistSlug || artworkSlug != expectedArtworkSlug {
		return c.Redirect(http.StatusMovedPermanently, "/artists/"+expectedArtistSlug+"/"+expectedArtworkSlug)
	}

	content := dto.Artwork{
		Id:        aw.GetId(),
		Title:     aw.GetString("title"),
		Comment:   aw.GetString("comment"),
		Technique: aw.GetString("technique"),
		Url:       url.GenerateArtworkUrl(url.ArtworkUrlDTO{
			ArtistName: artist.GetString("name"),
			ArtistId: artist.Id,
			ArtworkId: aw.GetId(),
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
			Id:         artist.GetId(),
			Name:       artist.GetString("name"),
			Bio:        artist.GetString("bio"),
			Profession: artist.GetString("profession"),
			Url:        url.GenerateArtistUrl(url.ArtistUrlDTO{
				ArtistId: artist.GetId(),
				ArtistName: artist.GetString("name"),
			}),
		},
	}

	fullUrl := c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()

	school := artist.GetStringSlice("school")

	var schoolCollector []string

	for _, s := range school {
		r, err := app.Dao().FindRecordById("schools", s)

		if err != nil {
			app.Logger().Error("school not found", err)
			continue
		}

		schoolCollector = append(schoolCollector, r.GetString("name"))

	}

	jsonLd := jsonld.ArtworkJsonLd(aw, &wgaModels.Artist{
		Id:           artist.GetId(),
		Name:         artist.GetString("name"),
		Slug:         artist.GetString("slug"),
		Bio:          artist.GetString("bio"),
		YearOfBirth:  artist.GetInt("year_of_birth"),
		YearOfDeath:  artist.GetInt("year_of_death"),
		PlaceOfBirth: artist.GetString("place_of_birth"),
		PlaceOfDeath: artist.GetString("place_of_death"),
		Published:    artist.GetBool("published"),
		School:       strings.Join(schoolCollector, ", "),
		Profession:   artist.GetString("profession"),
	}, c)

	marshalled, err := json.Marshal(jsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artwork jsonld for"+aw.GetId(), err)
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", content.Title, content.Artist.Name))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, content.Comment)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, c.Scheme()+"://"+c.Request().Host+content.Image.Image)

	c.Response().Header().Set("HX-Push-Url", fullUrl)
	err = pages.ArtworkPage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artwork page", err)
		return c.String(http.StatusInternalServerError, "failed to render response template")
	}

	return nil
}

// registerArtist registers the artist routes to the PocketBase app.
// It adds two routes to the app router:
// 1. GET /artists/:name - returns the artist page with the given name
// 2. GET /artists/:name/:awid - returns the artwork page with the given name and artwork id
// It also caches the HTML response for each route to improve performance.
func registerArtist(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("artists/:name", func(c echo.Context) error {
			return processArtist(c, app)
		})

		e.Router.GET("artists/:name/:awid", func(c echo.Context) error {
			return processArtwork(c, app)
		})
		return nil
	})
}
