package artists

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"sort"
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

	filters, err := buildArtistFilters(app, queryParams)
	if err != nil {
		app.Logger().Error("Build artist filters", "error", err)
		return utils.ServerFaultError(c)
	}

	offset := (page - 1) * limit

	filter, params := filters.buildFilter()

	records, err := app.FindRecordsByFilter(
		"artists",
		filter,
		"+name",
		limit,
		offset,
		params,
	)

	if err != nil {
		app.Logger().Error("Failed to get artist records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	recordsCount, err := utils.CountRecordsByFilter(app, "artists", filter, params)

	if err != nil {
		app.Logger().Error("Failed to get total records", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	content := dto.ArtistsView{
		Count:              strconv.Itoa(recordsCount),
		QueryStr:           filters.Query,
		SelectedLetter:     filters.Letter,
		SelectedSchool:     filters.School,
		SelectedPeriod:     filters.Period,
		SelectedProfession: filters.Profession,
	}
	if err := populateArtistFilterOptions(app, &content); err != nil {
		app.Logger().Error("Load artist filter options", "error", err)
		return utils.ServerFaultError(c)
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
			Portrait:   url.GenerateArtistPortraitThumbnailURL(m, url.ThumbnailPortraitCard),
		})

		jsonLdCollector = append(jsonLdCollector, jsonld.ArtistJsonLd(m))

	}

	marshalledJsonLd, err := json.Marshal(jsonLdCollector)

	if err != nil {
		app.Logger().Error("Failed to marshal Artist JSON-LD", "error", err.Error())
		return utils.BadRequestError(c)
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalledJsonLd)

	pagination := utils.NewPagination(recordsCount, limit, page, filters.path(), "", "")

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

type artistFilters struct {
	Query      string
	Letter     string
	School     string
	Period     string
	Profession string
	PeriodFrom int
	PeriodTo   int
}

func buildArtistFilters(app *pocketbase.PocketBase, values neturl.Values) (artistFilters, error) {
	filters := artistFilters{
		Query:      strings.TrimSpace(values.Get("q")),
		School:     strings.TrimSpace(values.Get("school")),
		Period:     strings.TrimSpace(values.Get("period")),
		Profession: strings.TrimSpace(values.Get("profession")),
	}
	letter := strings.ToUpper(strings.TrimSpace(values.Get("letter")))
	if len(letter) == 1 && letter[0] >= 'A' && letter[0] <= 'Z' {
		filters.Letter = letter
	}
	if filters.Period == "" {
		return filters, nil
	}

	period, err := app.FindRecordById("art_periods", filters.Period)
	if err != nil {
		filters.Period = ""
		return filters, nil
	}
	filters.PeriodFrom = period.GetInt("start")
	filters.PeriodTo = period.GetInt("end")
	return filters, nil
}

func (f artistFilters) buildFilter() (string, dbx.Params) {
	conditions := []string{"published = true"}
	params := dbx.Params{}
	if f.Query != "" {
		conditions = append(conditions, "(name ~ {:query} || profession ~ {:query} || school.name ~ {:query})")
		params["query"] = f.Query
	}
	if f.Letter != "" {
		conditions = append(conditions, "name ~ {:letter}")
		params["letter"] = f.Letter + "%"
	}
	if f.School != "" {
		conditions = append(conditions, "school.slug = {:school}")
		params["school"] = f.School
	}
	if f.Profession != "" {
		conditions = append(conditions, "profession ~ {:profession}")
		params["profession"] = f.Profession
	}
	if f.PeriodFrom > 0 {
		conditions = append(conditions, "year_of_birth >= {:period_from}")
		params["period_from"] = f.PeriodFrom
	}
	if f.PeriodTo > 0 {
		conditions = append(conditions, "year_of_birth <= {:period_to}")
		params["period_to"] = f.PeriodTo
	}
	return strings.Join(conditions, " && "), params
}

func (f artistFilters) path() string {
	values := neturl.Values{}
	if f.Query != "" {
		values.Set("q", f.Query)
	}
	if f.Letter != "" {
		values.Set("letter", f.Letter)
	}
	if f.School != "" {
		values.Set("school", f.School)
	}
	if f.Period != "" {
		values.Set("period", f.Period)
	}
	if f.Profession != "" {
		values.Set("profession", f.Profession)
	}
	if len(values) == 0 {
		return "/artists"
	}
	return "/artists?" + values.Encode()
}

func populateArtistFilterOptions(app *pocketbase.PocketBase, view *dto.ArtistsView) error {
	schools, err := app.FindRecordsByFilter("schools", "", "+name", 0, 0)
	if err != nil {
		return err
	}
	for _, school := range schools {
		view.SchoolOptions = append(view.SchoolOptions, dto.ArtistFilterOption{Value: school.GetString("slug"), Label: school.GetString("name")})
	}
	periods, err := app.FindRecordsByFilter("art_periods", "", "+start,+name", 0, 0)
	if err != nil {
		return err
	}
	for _, period := range periods {
		view.PeriodOptions = append(view.PeriodOptions, dto.ArtistFilterOption{Value: period.Id, Label: period.GetString("name")})
	}
	artists, err := app.FindRecordsByFilter("artists", "published = true", "+profession", 0, 0)
	if err != nil {
		return err
	}
	professions := map[string]struct{}{}
	for _, artist := range artists {
		profession := strings.TrimSpace(artist.GetString("profession"))
		if profession != "" {
			professions[profession] = struct{}{}
		}
	}
	for profession := range professions {
		view.ProfessionOptions = append(view.ProfessionOptions, dto.ArtistFilterOption{Value: profession, Label: profession})
	}
	sort.Slice(view.ProfessionOptions, func(i, j int) bool { return view.ProfessionOptions[i].Label < view.ProfessionOptions[j].Label })
	return nil
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
