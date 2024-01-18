package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/assets"
	wgamodels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/jsonld"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
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
	s := []string{}

	yob := r.GetInt("year_of_birth")
	eyob := r.GetString("exact_year_of_birth")
	pob := r.GetString("place_of_birth")
	kpob := r.GetString("known_place_of_birth")

	yod := r.GetInt("year_of_death")
	eyod := r.GetString("exact_year_of_death")
	pod := r.GetString("place_of_death")
	kpod := r.GetString("known_place_of_death")

	if yob > 0 {

		c := []string{}

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

		c := []string{}

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

// processArtist is a handler function that processes the artist request.
// It retrieves the artist information from the database, including their works,
// and renders the HTML template with the artist data.
// If the artist is not found, it returns a not found error.
// If the rendering fails, it returns an error.
// It also sets the HX-Push-Url header to enable Htmx push for the artist page.
func processArtist(c echo.Context, app *pocketbase.PocketBase) error {
	slug := c.PathParam("name")
	cacheKey := "artist:" + slug

	htmx := utils.IsHtmxRequest(c)

	if htmx {
		cacheKey = cacheKey + "-htmx"
	}

	html := ""

	found := app.Store().Has(cacheKey)

	if found {
		html = app.Store().Get(cacheKey).(string)
	} else {

		// split the slug on the last dash and use the last part as the artist id

		slugParts := strings.Split(slug, "-")
		id := slugParts[len(slugParts)-1]

		fullUrl := os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + c.Request().URL.String()
		artist, err := app.Dao().FindRecordById("artists", id)

		if err != nil {
			app.Logger().Error("Artist not found: ", slug, err)
			return apis.NewNotFoundError("", err)
		}

		expectedSlug := artist.GetString("slug") + "-" + artist.GetString("id")

		if slug != expectedSlug {
			return c.Redirect(http.StatusMovedPermanently, "/artists/"+expectedSlug)
		}

		works, err := app.Dao().FindRecordsByFilter("artworks", "author = '"+artist.GetString("id")+"'", "+title", 100, 0)

		if err != nil {
			app.Logger().Error("Error finding artworks: ", err)
			return apis.NewNotFoundError("", err)
		}

		data := assets.NewRenderData(app)

		data["Name"] = artist.GetString("name")
		data["Bio"] = artist.GetString("bio")
		data["Works"] = []map[string]any{}
		data["Slug"] = slug
		data["BioExcerpt"] = normalizedBioExcerpt(artist)
		data["CurrentUrl"] = fullUrl
		data["Profession"] = artist.GetString("profession")
		data["YearOfBirth"] = artist.GetString("year_of_birth")
		data["YearOfDeath"] = artist.GetString("year_of_death")
		data["PlaceOfBirth"] = artist.GetString("place_of_birth")
		data["PlaceOfDeath"] = artist.GetString("place_of_death")
		data["Jsonld"] = jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
			Name:         artist.GetString("name"),
			Slug:         artist.GetString("slug"),
			Bio:          artist.GetString("bio"),
			YearOfBirth:  artist.GetInt("year_of_birth"),
			YearOfDeath:  artist.GetInt("year_of_death"),
			PlaceOfBirth: artist.GetString("place_of_birth"),
			PlaceOfDeath: artist.GetString("place_of_death"),
			Published:    artist.GetBool("published"),
			School:       artist.GetString("school"),
			Profession:   artist.GetString("profession"),
		}, c)

		for _, w := range works {

			jsonLd := jsonld.GenerateVisualArtworkJsonLdContent(w, c)

			jsonLd["image"] = url.GenerateFileUrl("artworks", w.GetString("id"), w.GetString("image"), "")
			jsonLd["url"] = fullUrl + "/" + utils.Slugify(w.GetString("title")) + "-" + w.GetString("id")
			jsonLd["creator"] = jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
				Name:         artist.GetString("name"),
				Slug:         artist.GetString("slug"),
				Bio:          artist.GetString("bio"),
				YearOfBirth:  artist.GetInt("year_of_birth"),
				YearOfDeath:  artist.GetInt("year_of_death"),
				PlaceOfBirth: artist.GetString("place_of_birth"),
				PlaceOfDeath: artist.GetString("place_of_death"),
				Published:    artist.GetBool("published"),
				School:       artist.GetString("school"),
				Profession:   artist.GetString("profession"),
			}, c)
			jsonLd["creator"].(map[string]any)["sameAs"] = fullUrl
			jsonLd["thumbnailUrl"] = url.GenerateThumbUrl("artworks", w.GetString("id"), w.GetString("image"), "320x240", "")

			data["Works"] = append(data["Works"].([]map[string]any), map[string]any{
				"Id":        w.GetId(),
				"Title":     w.GetString("title"),
				"Comment":   w.GetString("comment"),
				"Technique": w.GetString("technique"),
				"Image":     jsonLd["image"].(string),
				"Thumb":     jsonLd["thumbnailUrl"].(string),
				"Jsonld":    jsonLd,
				"Url":       c.Request().URL.String() + "/" + utils.Slugify(w.GetString("title")) + "-" + w.GetString("id"),
			})
		}

		html, err = assets.Render(assets.Renderable{
			IsHtmx: htmx,
			Block:  "artist:content",
			Data:   data,
		})

		if err != nil {
			// or redirect to a dedicated 404 HTML page
			return apis.NewNotFoundError("", err)
		}

		app.Store().Set(cacheKey, html)
	}

	c.Response().Header().Set("HX-Push-Url", "/artists/"+slug)

	return c.HTML(http.StatusOK, html)
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
	htmx := utils.IsHtmxRequest(c)

	artistSlug := c.PathParam("name")
	artworkSlug := c.PathParam("awid")
	cacheKey := "artist:" + artistSlug + artworkSlug

	if htmx {
		cacheKey = cacheKey + "-htmx"
	}

	html := ""

	found := app.Store().Has(cacheKey)
	// found := false

	if found {
		html = app.Store().Get(cacheKey).(string)
	} else {

		// split the slug on the last dash and use the last part as the artist id
		artistSlugParts := strings.Split(artistSlug, "-")
		artistId := artistSlugParts[len(artistSlugParts)-1]

		artist, err := app.Dao().FindRecordById("artists", artistId)

		// if the artist is not found, return a not found error
		if err != nil {
			app.Logger().Error("Artist not found: ", artistSlug, err)
			return apis.NewNotFoundError("", err)
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
			return apis.NewNotFoundError("", err)
		}

		// generate the expected slug for the artwork
		expectedArtworkSlug := utils.Slugify(aw.GetString("title")) + "-" + aw.GetString("id")

		// redirect to the correct URL if either slug is not correct
		if artistSlug != expectedArtistSlug || artworkSlug != expectedArtworkSlug {
			return c.Redirect(http.StatusMovedPermanently, "/artists/"+expectedArtistSlug+"/"+expectedArtworkSlug)
		}

		data := assets.NewRenderData(app)

		data["ArtistName"] = artist.GetString("name")
		data["ArtistUrl"] = "/artists/" + artistSlug
		data["AwId"] = artworkSlug
		data["AwImage"] = url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), "")
		data["AwTitle"] = aw.GetString("title")
		data["AwComment"] = aw.GetString("comment")
		data["AwTechnique"] = aw.GetString("technique")

		fullUrl := os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + c.Request().URL.String()
		jsonLd := jsonld.GenerateVisualArtworkJsonLdContent(aw, c)

		jsonLd["image"] = url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), "")
		jsonLd["url"] = fullUrl
		jsonLd["creator"] = jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
			Name:         artist.GetString("name"),
			Slug:         artist.GetString("slug"),
			Bio:          artist.GetString("bio"),
			YearOfBirth:  artist.GetInt("year_of_birth"),
			YearOfDeath:  artist.GetInt("year_of_death"),
			PlaceOfBirth: artist.GetString("place_of_birth"),
			PlaceOfDeath: artist.GetString("place_of_death"),
			Published:    artist.GetBool("published"),
			School:       artist.GetString("school"),
			Profession:   artist.GetString("profession"),
		}, c)
		jsonLd["creator"].(map[string]any)["sameAs"] = os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + "/artists/" + artistSlug
		jsonLd["thumbnailUrl"] = url.GenerateThumbUrl("artworks", aw.GetString("id"), aw.GetString("image"), "320x240", "")

		data["Jsonld"] = jsonLd

		html, err = assets.Render(assets.Renderable{
			IsHtmx: htmx,
			Block:  "artwork:content",
			Data:   data,
		})

		if err != nil {
			// or redirect to a dedicated 404 HTML page
			return apis.NewNotFoundError("", err)
		}

		app.Store().Set(cacheKey, html)
	}

	c.Response().Header().Set("HX-Push-Url", "/artists/"+artistSlug+"/"+artworkSlug)

	return c.HTML(http.StatusOK, html)
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
