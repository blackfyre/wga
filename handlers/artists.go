package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/utils/url"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	wgaModels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/jsonld"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// processArtists is a function that handles the processing of artists in the application.
// It takes a PocketBase instance and an echo.Context as parameters.
// The function retrieves artists based on the provided search expression and pagination parameters.
// It then renders the artists' information in different views based on the request type (HTML or HTMX).
// The function returns an error if there is any issue with retrieving or rendering the artists' information.
func processArtists(app *pocketbase.PocketBase, c echo.Context) error {

	limit := 30
	page := 1
	searchExpression := ""
	searchExpressionPresent := false
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

	if c.QueryParams().Has("q") {
		searchExpressionPresent = true
	}

	if c.QueryParam("q") != "" {
		searchExpression = c.QueryParam("q")
	}

	offset := (page - 1) * limit

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
		app.Logger().Error("Failed to get artist records", "error", err.Error())
		return utils.ServerFaultError(c)
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
		app.Logger().Error("Failed to get total records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	recordsCount := len(totalRecords)

	content := dto.ArtistsView{
		Count: strconv.Itoa(recordsCount),
	}

	if len(searchExpression) > 0 && searchExpressionPresent {
		content.QueryStr = searchExpression
	}

	var jsonLdCollector []jsonld.Person

	for _, m := range records {

		// TODO: handle a.k.a. names

		school := m.GetStringSlice("school")

		var schoolCollector []string

		for _, s := range school {
			r, err := app.Dao().FindRecordById("schools", s)

			if err != nil {
				log.Print("school not found")
				continue
			}

			schoolCollector = append(schoolCollector, r.GetString("name"))

		}

		schools := strings.Join(schoolCollector, ", ")

		content.Artists = append(content.Artists, dto.Artist{
			Name:       m.GetString("name"),
			Url:        url.GenerateArtistUrlFromRecord(m),
			Profession: m.GetString("profession"),
			BornDied:   normalizedBirthDeathActivity(m),
			Schools:    schools,
		})

		jsonLdCollector = append(jsonLdCollector, jsonld.ArtistJsonLd(&wgaModels.Artist{
			Id:           m.GetId(),
			Name:         m.GetString("name"),
			Slug:         m.GetString("slug"),
			Bio:          m.GetString("bio"),
			YearOfBirth:  m.GetInt("year_of_birth"),
			YearOfDeath:  m.GetInt("year_of_death"),
			PlaceOfBirth: m.GetString("place_of_birth"),
			PlaceOfDeath: m.GetString("place_of_death"),
			Published:    m.GetBool("published"),
			School:       schools,
			Profession:   m.GetString("profession"),
		}, c))

	}

	marshalledJsonLd, err := json.Marshal(jsonLdCollector)

	if err != nil {
		app.Logger().Error("Failed to marshal Artist JSON-LD", "error", err.Error())
		return apis.NewBadRequestError("Invalid page", err)
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalledJsonLd)

	pagination := utils.NewPagination(recordsCount, limit, page, "/artists?q="+searchExpression, "", "")

	content.Pagination = string(pagination.Render())

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Artists")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Check out the artists in the gallery.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, c.Scheme()+"://"+c.Request().Host+c.Request().URL.String())

	c.Response().Header().Set("HX-Push-Url", currentUrl)
	err = pages.ArtistsPageFull(content).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artists", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil

}

// registerArtists registers the "/artists" route in the provided PocketBase application.
// It adds a GET handler for the "/artists" route that calls the processArtists function.
func registerArtists(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/artists", func(c echo.Context) error {

			return processArtists(app, c)

		})

		return nil
	})
}
