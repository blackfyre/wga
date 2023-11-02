package search

import (
	"fmt"
	"net/http"
	"strconv"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"blackfyre.ninja/wga/utils/jsonld"
	"blackfyre.ninja/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func search(app *pocketbase.PocketBase, e *core.ServeEvent, c echo.Context) error {
	//setup request variables
	htmx := utils.IsHtmxRequest(c)
	currentUrl := c.Request().URL.String()
	limit := 16
	page := 1

	//build filters
	filters := buildFilters(app, c)

	//set page
	if c.QueryParam("page") != "" {
		err := error(nil)
		page, err = strconv.Atoi(c.QueryParam("page"))

		if err != nil {
			return apis.NewBadRequestError("Invalid page", err)
		}
	}

	//set offset
	offset := page * limit

	//set cache key
	cacheKey := "search:" + strconv.Itoa(offset) + ":" + strconv.Itoa(limit) + ":" + strconv.Itoa(page) + ":" + filters.FingerPrint()

	if htmx {
		cacheKey = cacheKey + ":htmx"
	}

	if filters.AnyFilterActive() {
		cacheKey = cacheKey + ":search"
	}

	//check cache
	if app.Cache().Has(cacheKey) {
		html := app.Cache().Get(cacheKey).(string)
		return c.HTML(http.StatusOK, html)
	} else {

		td := assets.NewRenderData(app)

		td["Artworks"] = []any{}

		filterString, filterParams := filters.BuildFilter()

		records, err := app.Dao().FindRecordsByFilter(
			"artworks",
			filterString,
			"+title",
			limit,
			offset,
			filterParams,
		)

		if err != nil {
			fmt.Println(err)
			return apis.NewBadRequestError("Invalid page", err)
		}

		// this could be replaced with a dedicated sql query, but this is more convinient
		totalRecords, err := app.Dao().FindRecordsByFilter(
			"artworks",
			filterString,
			"",
			0,
			0,
			filterParams,
		)

		if err != nil {
			return apis.NewBadRequestError("Invalid page", err)
		}

		recordsCount := len(totalRecords)

		for _, v := range records {

			artistIds := v.GetStringSlice("author")

			if len(artistIds) == 0 {
				// wating for the promised logging system by @pocketbase
				continue
			}

			artist, err := models.GetArtistById(app.Dao(), artistIds[0])

			if err != nil {
				// wating for the promised logging system by @pocketbase
				continue
			}

			jsonLd := jsonld.GenerateVisualArtworkJsonLdContent(v, c)

			jsonLd["image"] = url.GenerateFileUrl(app, "artworks", v.GetString("id"), v.GetString("image"))
			// jsonLd["url"] = fullUrl + "/" + v.GetString("id")
			jsonLd["creator"] = jsonld.GenerateArtistJsonLdContent(artist, c)
			// jsonLd["creator"].(map[string]any)["sameAs"] = fullUrl
			jsonLd["thumbnailUrl"] = url.GenerateThumbUrl(app, "artworks", v.GetString("id"), v.GetString("image"), "320x240")

			td["Artworks"] = append(td["Artworks"].([]any), map[string]any{
				"Id":         v.GetId(),
				"Title":      v.GetString("title"),
				"Comment":    v.GetString("comment"),
				"Technique":  v.GetString("technique"),
				"Image":      jsonLd["image"].(string),
				"Thumb":      jsonLd["thumbnailUrl"].(string),
				"ArtistSlug": artist.Slug,
				"Jsonld":     jsonLd,
			})
		}

		td["ArtFormOptions"], _ = getArtFormOptions(app)
		td["ArtTypeOptions"], _ = getArtTypesOptions(app)
		td["ArtSchoolOptions"], _ = getArtSchoolOptions(app)
		td["ArtistNameList"], _ = getArtistNameList(app)
		td["ActiveFilterValues"] = filters

		pagination := utils.NewPagination(recordsCount, limit, page, currentUrl, "artwork-search-results")

		td["Pagination"] = pagination.Render()

		blockToRender := "search:content"

		if htmx {
			blockToRender = "search:search-results"
		}

		html, err := assets.Render(assets.Renderable{
			IsHtmx: htmx,
			Block:  blockToRender,
			Data:   td,
		})

		if err != nil {
			return apis.NewNotFoundError("", err)
		}

		c.Response().Header().Set("HX-Push-Url", currentUrl)

		return c.HTML(http.StatusOK, html)
	}
}

// RegisterSearchHandlers registers search handlers to the given PocketBase app.
func RegisterSearchHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/search", func(c echo.Context) error {
			return search(app, e, c)
		})
		return nil
	})
}
