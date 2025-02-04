package artworks

import (
	"context"
	"net/http"
	"strconv"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func searchPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	fullUrl := utils.AssetUrl(c.Request.URL.String())
	filters := buildFilters(c)

	if filters.AnyFilterActive() {
		// redirect to the search results page
		return c.Redirect(http.StatusFound, "/artworks/results?"+filters.BuildFilterString())
	}

	content := dto.ArtworkSearchDTO{
		ActiveFilterValues: &dto.ArtworkSearchFilterValues{
			Title:         filters.Title,
			SchoolString:  filters.SchoolString,
			ArtFormString: filters.ArtFormString,
			ArtTypeString: filters.ArtTypeString,
			ArtistString:  filters.ArtistString,
		},
	}

	content.ArtFormOptions, _ = getArtFormOptions(app)
	content.ArtTypeOptions, _ = getArtTypesOptions(app)
	content.ArtSchoolOptions, _ = getArtSchoolOptions(app)
	content.ArtistNameList, _ = GetArtistNameList(app)
	content.NewFilterValues = filters.BuildFilterString()

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Artworks Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "On this page you can search for artworks by title, artist, art form, art type and art school!")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, fullUrl)

	c.Response().Header().Set("HX-Push-Url", fullUrl)
	err := pages.ArtworkSearchPage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil

}

func search(app *pocketbase.PocketBase, c echo.Context) error {

	limit := 16
	page := 1
	offset := 0

	if c.QueryParam("page") != "" {
		err := error(nil)
		page, err = strconv.Atoi(c.QueryParam("page"))

		if err != nil {
			return apis.NewBadRequestError("Invalid page", err)
		}
	}

	offset = (page - 1) * limit

	//build filters
	filters := buildFilters(c)

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

	content := dto.ArtworkSearchDTO{
		Results: dto.ArtworkSearchResultDTO{
			Artworks: dto.ImageGrid{},
		},
		ActiveFilterValues: &dto.ArtworkSearchFilterValues{
			Title:         filters.Title,
			SchoolString:  filters.SchoolString,
			ArtFormString: filters.ArtFormString,
			ArtTypeString: filters.ArtTypeString,
			ArtistString:  filters.ArtistString,
		},
	}

	content.ArtFormOptions, _ = getArtFormOptions(app)
	content.ArtTypeOptions, _ = getArtTypesOptions(app)
	content.ArtSchoolOptions, _ = getArtSchoolOptions(app)
	content.ArtistNameList, _ = GetArtistNameList(app)
	content.NewFilterValues = filters.BuildFilterString()
	content.Results.ActiveFiltering = filters.AnyFilterActive()

	for _, v := range records {

		artistIds := v.GetStringSlice("author")

		if len(artistIds) == 0 {
			// waiting for the promised logging system by @pocketbase
			continue
		}

		artist, err := models.GetArtistById(app.Dao(), artistIds[0])

		if err != nil {
			// waiting for the promised logging system by @pocketbase
			continue
		}

		content.Results.Artworks = append(content.Results.Artworks, dto.Image{
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   artist.Name,
				ArtistId:     artist.Id,
				ArtworkTitle: v.GetString("title"),
				ArtworkId:    v.GetId(),
			}),
			Image:     url.GenerateFileUrl("artworks", v.GetString("id"), v.GetString("image"), ""),
			Thumb:     url.GenerateThumbUrl("artworks", v.GetString("id"), v.GetString("image"), "320x240", ""),
			Comment:   v.GetString("comment"),
			Title:     v.GetString("title"),
			Technique: v.GetString("technique"),
			Id:        v.GetId(),
			Artist: dto.Artist{
				Id:   artist.Id,
				Name: artist.Name,
				Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
					ArtistId:   artist.Id,
					ArtistName: artist.Name,
				}),
				Profession: artist.Profession,
			},
		})
	}

	pUrl := "/artworks?" + filters.BuildFilterString()
	pHtmxUrl := "/artworks/results?" + filters.BuildFilterString()

	pagination := utils.NewPagination(recordsCount, limit, page, pUrl, "", pHtmxUrl)

	content.Results.Pagination = string(pagination.Render())

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Artworks Search")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "On this page you can search for artworks by title, artist, art form, art type and art school!")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, pHtmxUrl)

	c.Response().Header().Set("HX-Push-Url", pHtmxUrl)
	err = pages.ArtworkSearchPage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil
}

// RegisterArtworksHandlers registers search handlers to the given PocketBase app.
func RegisterArtworksHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/artworks", func(*core.RequestEvent) error {
			return searchPage(app, c)
		})

		e.Router.GET("/artworks/results", func(*core.RequestEvent) error {
			return search(app, c)
		})
		return nil
	})
}
