package artists

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/utils/url"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/jsonld"
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
func processArtists(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	limit := 30
	page := 1
	searchExpression := ""
	searchExpressionPresent := false
	currentUrl := c.Request.URL.String()
	c.Response.Header().Set("HX-Push-Url", currentUrl)

	queryParams := c.Request.URL.Query()

	if c.Request.URL.Query().Get("page") != "" {
		err := error(nil)
		page, err = strconv.Atoi(queryParams.Get("page"))

		if err != nil {
			app.Logger().Error("Invalid page: ", queryParams.Get("page"), err)
			return apis.NewBadRequestError("Invalid page", err)
		}
	}

	if queryParams.Has("q") {
		searchExpressionPresent = true
	}

	if queryParams.Get("q") != "" {
		searchExpression = queryParams.Get("q")
	}

	offset := (page - 1) * limit

	filter, filterParams, activeLetter := buildArtistsFilter(searchExpression, queryParams.Get("letter"))

	records, err := app.FindRecordsByFilter(
		"artists",
		filter,
		"+name",
		limit,
		offset,
		filterParams,
	)

	if err != nil {
		app.Logger().Error("Failed to get artist records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	totalRecords, err := app.FindRecordsByFilter(
		"artists",
		filter,
		"+name",
		0,
		0,
		filterParams,
	)

	if err != nil {
		app.Logger().Error("Failed to get total records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	recordsCount := len(totalRecords)

	content := dto.ArtistsView{
		Count:        strconv.Itoa(recordsCount),
		ActiveLetter: activeLetter,
	}

	availableRecords, err := app.FindRecordsByFilter(
		"artists",
		"published = true",
		"+name",
		0,
		0,
	)
	if err != nil {
		app.Logger().Error("Failed to get available artist initials", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	content.AvailableLetters = map[string]bool{}
	for _, record := range availableRecords {
		name := strings.ToUpper(strings.TrimSpace(record.GetString("name")))
		if len(name) == 0 || name[0] < 'A' || name[0] > 'Z' {
			continue
		}

		content.AvailableLetters[string(name[0])] = true
	}

	if len(searchExpression) > 0 && searchExpressionPresent {
		content.QueryStr = searchExpression
	}

	var jsonLdCollector []jsonld.Person

	for _, m := range records {

		// TODO: handle a.k.a. names

		schools := utils.RenderSchoolNames(app, m.GetStringSlice("school"))

		content.Artists = append(content.Artists, dto.Artist{
			Name:       m.GetString("name"),
			Url:        url.GenerateArtistUrlFromRecord(m),
			Profession: m.GetString("profession"),
			BornDied:   utils.NormalizedBirthDeathActivity(m),
			Schools:    schools,
		})

		jsonLdCollector = append(jsonLdCollector, jsonld.ArtistJsonLd(m))

	}

	marshalledJsonLd, err := json.Marshal(jsonLdCollector)

	if err != nil {
		app.Logger().Error("Failed to marshal Artist JSON-LD", "error", err.Error())
		return utils.BadRequestError(c)
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalledJsonLd)

	paginationQuery := neturl.Values{}
	paginationQuery.Set("q", searchExpression)
	if activeLetter != "" {
		paginationQuery.Set("letter", activeLetter)
	}
	pagination := utils.NewPagination(recordsCount, limit, page, "/artists?"+paginationQuery.Encode(), "", "")

	content.Pagination = string(pagination.Render())

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Artists")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Check out the artists in the gallery.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, utils.AssetUrl(c.Request.URL.String()))

	var buff bytes.Buffer

	c.Response.Header().Set("HX-Push-Url", currentUrl)
	err = pages.ArtistsPageFull(content).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artists", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())

}

func buildArtistsFilter(searchExpression string, rawLetter string) (string, dbx.Params, string) {
	activeLetter := strings.ToUpper(strings.TrimSpace(rawLetter))
	if len(activeLetter) != 1 || activeLetter[0] < 'A' || activeLetter[0] > 'Z' {
		activeLetter = ""
	}

	filter := "published = true"
	params := dbx.Params{
		"searchExpression": searchExpression,
	}

	if searchExpression != "" {
		filter = filter + " && name ~ {:searchExpression}"
	}
	if activeLetter != "" {
		filter = filter + " && name ~ {:activeLetter}"
		params["activeLetter"] = activeLetter + "%"
	}

	return filter, params, activeLetter
}

func RegisterHandlers(app *pocketbase.PocketBase) {

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		ag := se.Router.Group("/artists")

		ag.GET("", func(c *core.RequestEvent) error {

			return processArtists(app, c)

		})

		ag.GET("/{name}", func(e *core.RequestEvent) error {
			return processArtist(e, app)
		})

		ag.GET("/{name}/{awid}", func(e *core.RequestEvent) error {
			return processArtwork(e, app)
		})
		return se.Next()
	})
}
