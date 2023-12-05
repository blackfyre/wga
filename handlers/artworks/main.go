package artworks

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

func searchPage(app *pocketbase.PocketBase, e *core.ServeEvent, c echo.Context) error {
	//setup request variables
	htmx := utils.IsHtmxRequest(c)
	// currentUrl := c.Request().URL.String()
	currentUrlPath := c.Request().URL.Path

	//build filters
	filters := buildFilters(app, c)

	//set page

	//check cache
	td := assets.NewRenderData(app)

	td["ArtFormOptions"], _ = getArtFormOptions(app)
	td["ArtTypeOptions"], _ = getArtTypesOptions(app)
	td["ArtSchoolOptions"], _ = getArtSchoolOptions(app)
	td["ArtistNameList"], _ = getArtistNameList(app)
	td["ActiveFilterValues"] = filters
	td["NewFilterValues"] = filters.BuildFilterString()

	html, err := assets.Render(assets.Renderable{
		IsHtmx: htmx,
		Block:  "artworks:content",
		Data:   td,
	})

	if err != nil {
		return apis.NewNotFoundError("", err)
	}

	c.Response().Header().Set("HX-Push-Url", currentUrlPath+"?"+filters.BuildFilterString())

	return c.HTML(http.StatusOK, html)
}

func search(app *pocketbase.PocketBase, e *core.ServeEvent, c echo.Context) error {

	htmx := utils.IsHtmxRequest(c)

	if !htmx {
		return c.Redirect(http.StatusFound, "/artworks")
	}

	limit := 16
	page := 0
	offset := 0

	if c.QueryParam("page") != "" {
		err := error(nil)
		page, err = strconv.Atoi(c.QueryParam("page"))

		if err != nil {
			return apis.NewBadRequestError("Invalid page", err)
		}

		page = page - 1
	}

	offset = page * limit

	//build filters
	filters := buildFilters(app, c)

	filterString, filterParams := filters.BuildFilter()

	td := assets.NewRenderData(app)

	records, err := app.Dao().FindRecordsByFilter(
		"artworks",
		filterString,
		"+title",
		limit,
		offset,
		filterParams,
	)

	if err != nil {
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

	td["Artworks"] = []any{}

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

		jsonLd["image"] = url.GenerateFileUrl("artworks", v.GetString("id"), v.GetString("image"), "")
		jsonLd["creator"] = jsonld.GenerateArtistJsonLdContent(artist, c)
		jsonLd["thumbnailUrl"] = url.GenerateThumbUrl("artworks", v.GetString("id"), v.GetString("image"), "320x240", "")

		row := map[string]any{
			"Id":         v.GetId(),
			"Title":      v.GetString("title"),
			"Comment":    v.GetString("comment"),
			"Technique":  v.GetString("technique"),
			"Image":      jsonLd["image"].(string),
			"Thumb":      jsonLd["thumbnailUrl"].(string),
			"ArtistSlug": artist.Slug,
			"Jsonld":     jsonLd,
		}

		td["Artworks"] = append(td["Artworks"].([]any), row)
	}

	pUrl := "/artworks?" + filters.BuildFilterString()
	pHtmxUrl := "/artworks/results?" + filters.BuildFilterString()

	fmt.Printf("pUrl: %s\n", pUrl)
	fmt.Printf("pHtmxUrl: %s\n", pHtmxUrl)

	pagination := utils.NewPagination(recordsCount, limit, page+1, pUrl, "artwork-search-results", pHtmxUrl)

	td["Pagination"] = pagination.Render()

	html, err := assets.Render(assets.Renderable{
		IsHtmx: htmx,
		Block:  "artworks:results",
		Data:   td,
	})

	if err != nil {
		return apis.NewNotFoundError("", err)
	}

	// c.Response().Header().Set("HX-Push-Url", currentUrl)

	return c.HTML(http.StatusOK, html)
}

// RegisterArtworksHandlers registers search handlers to the given PocketBase app.
func RegisterArtworksHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/artworks", func(c echo.Context) error {
			return searchPage(app, e, c)
		})

		e.Router.GET("/artworks/results", func(c echo.Context) error {
			return search(app, e, c)
		})
		return nil
	})
}
