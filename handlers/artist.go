package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"blackfyre.ninja/wga/assets"
	wgamodels "blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"blackfyre.ninja/wga/utils/jsonld"
	"blackfyre.ninja/wga/utils/url"
	"github.com/joho/godotenv"
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

// registerArtist registers the artist routes to the PocketBase app.
// It adds two routes to the app router:
// 1. GET /artists/:name - returns the artist page with the given name
// 2. GET /artists/:name/:awid - returns the artwork page with the given name and artwork id
// It also caches the HTML response for each route to improve performance.
func registerArtist(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("artists/:name", func(c echo.Context) error {
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

				fullUrl := os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + c.Request().URL.String()
				artist, err := app.Dao().FindRecordsByFilter("artists", "slug = '"+slug+"'", "+name", 1, 0)

				if err != nil {
					return apis.NewNotFoundError("", err)
				}

				works, err := app.Dao().FindRecordsByFilter("artworks", "author = '"+artist[0].GetString("id")+"'", "+title", 100, 0)

				if err != nil {
					return apis.NewNotFoundError("", err)
				}

				data := assets.NewRenderData(app)

				data["Name"] = artist[0].GetString("name")
				data["Bio"] = artist[0].GetString("bio")
				data["Works"] = []map[string]any{}
				data["Slug"] = slug
				data["BioExcerpt"] = normalizedBioExcerpt(artist[0])
				data["CurrentUrl"] = fullUrl
				data["Profession"] = artist[0].GetString("profession")
				data["YearOfBirth"] = artist[0].GetString("year_of_birth")
				data["YearOfDeath"] = artist[0].GetString("year_of_death")
				data["PlaceOfBirth"] = artist[0].GetString("place_of_birth")
				data["PlaceOfDeath"] = artist[0].GetString("place_of_death")
				data["Jsonld"] = jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
					Name:         artist[0].GetString("name"),
					Slug:         artist[0].GetString("slug"),
					Bio:          artist[0].GetString("bio"),
					YearOfBirth:  artist[0].GetInt("year_of_birth"),
					YearOfDeath:  artist[0].GetInt("year_of_death"),
					PlaceOfBirth: artist[0].GetString("place_of_birth"),
					PlaceOfDeath: artist[0].GetString("place_of_death"),
					Published:    artist[0].GetBool("published"),
					School:       artist[0].GetString("school"),
					Profession:   artist[0].GetString("profession"),
				}, c)

				for _, w := range works {

					jsonLd := jsonld.GenerateVisualArtworkJsonLdContent(w, c)

					jsonLd["image"] = url.GenerateFileUrl("artworks", w.GetString("id"), w.GetString("image"), "")
					jsonLd["url"] = fullUrl + "/" + w.GetString("id")
					jsonLd["creator"] = jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
						Name:         artist[0].GetString("name"),
						Slug:         artist[0].GetString("slug"),
						Bio:          artist[0].GetString("bio"),
						YearOfBirth:  artist[0].GetInt("year_of_birth"),
						YearOfDeath:  artist[0].GetInt("year_of_death"),
						PlaceOfBirth: artist[0].GetString("place_of_birth"),
						PlaceOfDeath: artist[0].GetString("place_of_death"),
						Published:    artist[0].GetBool("published"),
						School:       artist[0].GetString("school"),
						Profession:   artist[0].GetString("profession"),
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
						"Url":       c.Request().URL.String() + "/" + w.GetString("id"),
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
		})

		e.Router.GET("artists/:name/:awid", func(c echo.Context) error {

			htmx := utils.IsHtmxRequest(c)

			slug := c.PathParam("name")
			awid := c.PathParam("awid")
			cacheKey := "artist:" + slug + awid

			if htmx {
				cacheKey = cacheKey + "-htmx"
			}

			html := ""

			found := app.Store().Has(cacheKey)
			// found := false

			if found {
				html = app.Store().Get(cacheKey).(string)
			} else {

				err := godotenv.Load()

				if err != nil {
					return apis.NewBadRequestError("Error loading .env file", err)
				}

				artist, err := app.Dao().FindRecordsByFilter("artists", "slug = '"+slug+"'", "+name", 1, 0)

				if err != nil {
					return apis.NewNotFoundError("", err)
				}

				aw, err := app.Dao().FindRecordById("artworks", awid)

				if err != nil {
					return apis.NewNotFoundError("", err)
				}

				data := assets.NewRenderData(app)

				data["ArtistName"] = artist[0].GetString("name")
				data["ArtistUrl"] = "/artists/" + slug
				data["AwId"] = awid
				data["AwImage"] = url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), "")
				data["AwTitle"] = aw.GetString("title")
				data["AwComment"] = aw.GetString("comment")
				data["AwTechnique"] = aw.GetString("technique")

				fullUrl := os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + c.Request().URL.String()
				jsonLd := jsonld.GenerateVisualArtworkJsonLdContent(aw, c)

				jsonLd["image"] = url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), "")
				jsonLd["url"] = fullUrl
				jsonLd["creator"] = jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
					Name:         artist[0].GetString("name"),
					Slug:         artist[0].GetString("slug"),
					Bio:          artist[0].GetString("bio"),
					YearOfBirth:  artist[0].GetInt("year_of_birth"),
					YearOfDeath:  artist[0].GetInt("year_of_death"),
					PlaceOfBirth: artist[0].GetString("place_of_birth"),
					PlaceOfDeath: artist[0].GetString("place_of_death"),
					Published:    artist[0].GetBool("published"),
					School:       artist[0].GetString("school"),
					Profession:   artist[0].GetString("profession"),
				}, c)
				jsonLd["creator"].(map[string]any)["sameAs"] = os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + "/artists/" + slug
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

			c.Response().Header().Set("HX-Push-Url", "/artists/"+slug+"/"+awid)

			return c.HTML(http.StatusOK, html)
		})
		return nil
	})
}
