package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/assets"
	wgamodels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/jsonld"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerArtists(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/artists", func(c echo.Context) error {

			limit := 30
			page := 1
			searchExpression := ""
			searchExpressionPresent := false
			confirmedHtmxRequest := utils.IsHtmxRequest(c)
			currentUrl := c.Request().URL.String()
			c.Response().Header().Set("HX-Push-Url", currentUrl)

			if c.QueryParam("page") != "" {
				err := error(nil)
				page, err = strconv.Atoi(c.QueryParam("page"))

				if err != nil {
					app.Logger().Error("Invalid page: ", c.QueryParam("page"), err)
					return apis.NewBadRequestError("Invalid page", err)
				}
			}

			if c.QueryParam("q") != "" {
				searchExpression = c.QueryParam("q")
			}

			if c.QueryParams().Has("q") {
				searchExpressionPresent = true
			}

			offset := (page - 1) * limit

			cacheKey := "artists:" + strconv.Itoa(offset) + ":" + strconv.Itoa(limit) + ":" + strconv.Itoa(page) + ":" + searchExpression

			if confirmedHtmxRequest {
				cacheKey = cacheKey + ":htmx"
			}

			if searchExpressionPresent {
				cacheKey = cacheKey + ":search"
			}

			if app.Store().Has(cacheKey) {
				html := app.Store().Get(cacheKey).(string)
				return c.HTML(http.StatusOK, html)
			} else {

				filter := "published = true"

				if searchExpression != "" {
					filter = filter + " && name ~ {:searchExpression}"
				}

				records, err := app.Dao().FindRecordsByFilter(
					"artists",
					filter,
					"+name",
					limit,
					offset,
					dbx.Params{
						"searchExpression": searchExpression,
					},
				)

				if err != nil {
					app.Logger().Error("Failed to get artist records: ", err)
					return apis.NewBadRequestError("Invalid page", err)
				}

				totalRecords, err := app.Dao().FindRecordsByFilter(
					"artists",
					filter,
					"+name",
					0,
					0,
					dbx.Params{
						"searchExpression": searchExpression,
					},
				)

				if err != nil {
					app.Logger().Error("Failed to get total records: ", err)
					return apis.NewBadRequestError("Invalid page", err)
				}

				recordsCount := len(totalRecords)

				preRendered := []map[string]any{}

				for _, m := range records {

					// TODO: handle aka

					school := m.GetStringSlice("school")

					schoolCollector := []string{}

					for _, s := range school {
						r, err := app.Dao().FindRecordById("schools", s)

						if err != nil {
							log.Print("school not found")
							continue
						}

						schoolCollector = append(schoolCollector, r.GetString("name"))

					}

					row := map[string]any{
						"Name":       m.GetString("name"),
						"Url":        artistUrl(m.GetString("slug")),
						"Profession": m.GetString("profession"),
						"BornDied":   normalizedBirthDeathActivity(m),
						"Schools":    strings.Join(schoolCollector, ", "),
						"Jsonld": jsonld.GenerateArtistJsonLdContent(&wgamodels.Artist{
							Name:         m.GetString("name"),
							Slug:         m.GetString("slug"),
							Bio:          m.GetString("bio"),
							YearOfBirth:  m.GetInt("year_of_birth"),
							YearOfDeath:  m.GetInt("year_of_death"),
							PlaceOfBirth: m.GetString("place_of_birth"),
							PlaceOfDeath: m.GetString("place_of_death"),
							Published:    m.GetBool("published"),
							School:       m.GetString("school"),
							Profession:   m.GetString("profession"),
						}, c),
					}

					preRendered = append(preRendered, row)
				}

				data := assets.NewRenderData(app)

				data["Content"] = preRendered
				data["Count"] = recordsCount

				pagination := utils.NewPagination(recordsCount, limit, page, "/artists?q="+searchExpression, "search-results", "")

				data["Pagination"] = pagination.Render()

				html := ""
				blockToRender := "artists:content"

				if confirmedHtmxRequest {
					if searchExpression != "" || searchExpressionPresent {
						blockToRender = "artists:search-results"
					}
				}

				html, err = assets.Render(assets.Renderable{
					IsHtmx: confirmedHtmxRequest,
					Block:  blockToRender,
					Data:   data,
				})

				if err != nil {
					// or redirect to a dedicated 404 HTML page
					app.Logger().Error("Failed to render artists: ", err)
					return apis.NewApiError(500, err.Error(), err)
				}

				app.Store().Set(cacheKey, html)

				return c.HTML(http.StatusOK, html)
			}
		})

		return nil
	})
}
