package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type SearchSettings struct {
	searchExpression        string
	searchExpressionPresent bool
	filter                  string
}

type GbData struct {
	Title         string
	FirstContent  string
	SecondContent string
	MainContent   []map[string]string
	LatestEntries string
}

type GbPreparedData struct {
	Title                   string
	FirstContent            string
	SecondContent           string
	MainContent             []map[string]string
	LatestEntries           string
	CalendarYears           [][]string
	SearchExpressionPresent bool
	SearchExpression        string
}

func registerGuestbook(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/guestbook", func(c echo.Context) error {
			confirmedHtmxRequest := isHtmxRequest(c)
			currentUrl := c.Request().URL.String()
			c.Response().Header().Set("HX-Push-Url", currentUrl)

			searchSettings := gbProcessRequest(c)

			cacheKey := gbSetCacheSettings(confirmedHtmxRequest, searchSettings)
			shouldReturn, returnValue := gbIsCached(app, cacheKey, c)
			if shouldReturn {
				return returnValue
			}

			data, err := gbGetData(app, searchSettings)

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			html, err := gbRender(confirmedHtmxRequest, data)

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil

	})
}

func gbProcessRequest(c echo.Context) SearchSettings {
	searchExpression := ""
	searchExpressionPresent := false
	filter := "id != null"

	if c.QueryParam("q") != "" {
		searchExpression = c.QueryParam("q")
	}

	if c.QueryParams().Has("q") {
		searchExpressionPresent = true
	}

	if searchExpression != "" {
		filter = filter + " && created ~ {:searchExpression}"
	}

	searchSettings := SearchSettings{
		searchExpression:        searchExpression,
		searchExpressionPresent: searchExpressionPresent,
		filter:                  filter,
	}

	return searchSettings
}

func gbSetCacheSettings(confirmedHtmxRequest bool, searchSettings SearchSettings) string {
	cacheKey := "guestbook:" + ":" + searchSettings.searchExpression

	if confirmedHtmxRequest {
		cacheKey = cacheKey + ":htmx"
	}

	if searchSettings.searchExpressionPresent {
		cacheKey = cacheKey + ":search"
	}
	return cacheKey
}

func gbIsCached(app *pocketbase.PocketBase, cacheKey string, c echo.Context) (bool, error) {
	if app.Cache().Has(cacheKey) {
		html := app.Cache().Get(cacheKey).(string)
		return true, c.HTML(http.StatusOK, html)
	}
	return false, nil
}

func gbGetData(app *pocketbase.PocketBase, searchSettings SearchSettings) (GbPreparedData, error) {
	guestBookTitle, err := gbGetGuestbookTextContent(app, "guestbook")

	if err != nil {
		fmt.Println(err)
	}

	guestBookTextFirst, err := gbGetGuestbookTextContent(app, "thankYou")

	if err != nil {
		fmt.Println(err)
	}

	guestBookTextSecond, err := gbGetGuestbookTextContent(app, "guestbookPlayMusicLink")

	if err != nil {
		fmt.Println(err)
	}

	guestBookContent, err := gbGetGuestbookContent(app, searchSettings.filter, searchSettings.searchExpression)

	if err != nil {
		fmt.Println(err)
	}

	latestEntries, err := gbGetGuestbookTextContent(app, "latestEntries")

	if err != nil {
		fmt.Println(err)
	}

	rawData := GbData{
		Title:         guestBookTitle,
		FirstContent:  guestBookTextFirst,
		SecondContent: guestBookTextSecond,
		MainContent:   guestBookContent,
		LatestEntries: latestEntries,
	}

	data := gbPrepareDataForRender(rawData, searchSettings)

	return data, err
}

func gbPrepareDataForRender(data GbData, searchSettings SearchSettings) GbPreparedData {
	years := gbGetYears()

	preparedData := GbPreparedData{
		Title:                   data.Title,
		FirstContent:            data.FirstContent,
		SecondContent:           data.SecondContent,
		MainContent:             data.MainContent,
		CalendarYears:           years,
		SearchExpressionPresent: searchSettings.searchExpressionPresent,
		SearchExpression:        searchSettings.searchExpression,
		LatestEntries:           data.LatestEntries,
	}

	return preparedData
}

func gbRender(confirmedHtmxRequest bool, data GbPreparedData) (string, error) {
	dataMap, shouldReturn, html, err := gbGeneralizer(data)
	if shouldReturn {
		return html, err
	}

	if confirmedHtmxRequest {
		blockToRender := "guestbook:content"

		html, err := renderBlock(blockToRender, dataMap)
		return html, err
	} else {
		html, err := renderPage("guestbook", dataMap)
		return html, err
	}
}

func gbGetGuestbookTextContent(app *pocketbase.PocketBase, content string) (string, error) {
	strContent := fmt.Sprintf("strings:%s", content)

	found := app.Cache().Has(strContent)

	if found {
		return app.Cache().Get(strContent).(string), nil
	}

	record, err := app.Dao().FindFirstRecordByData("strings", "name", content)

	if err != nil {
		log.Println(err)
		return "", err
	}

	result := record.Get("content")

	app.Cache().Set(strContent, result.(string))

	return result.(string), nil
}

func gbGetGuestbookContent(app *pocketbase.PocketBase, filter string, searchExpression string) ([]map[string]string, error) {
	records, err := app.Dao().FindRecordsByFilter(
		"guestbook",
		filter,
		"-created",
		0,
		0,
		dbx.Params{
			"searchExpression": searchExpression,
		},
	)

	if err != nil {
		return nil, apis.NewBadRequestError("Invalid something", err)
	}

	preRendered := []map[string]string{}

	for _, m := range records {

		createDateString := m.GetString("created")
		createTime, err := time.Parse("2006-01-02 15:04:05.000Z", createDateString)
		if err != nil {
			fmt.Println("Error parsing the date:", err)
			return nil, err
		}
		formattedDateString := createTime.Format("January 2, 2006")

		row := map[string]string{
			"Message":  m.GetString("message"),
			"Name":     m.GetString("name"),
			"Location": m.GetString("location"),
			"Updated":  formattedDateString,
		}

		preRendered = append(preRendered, row)
	}

	return preRendered, err
}

func gbGetYears() [][]string {
	years := []string{}
	thisYear := time.Now().Year()

	for i := 1997; i <= thisYear; i++ {
		years = append(years, fmt.Sprintf("%d", i))
	}

	// group by 5
	groupedYears := [][]string{}
	for i := 0; i < len(years); i += 5 {
		end := i + 5
		if end > len(years) {
			end = len(years)
		}
		groupedYears = append(groupedYears, years[i:end])
	}

	// reverse order
	for i, j := 0, len(groupedYears)-1; i < j; i, j = i+1, j-1 {
		groupedYears[i], groupedYears[j] = groupedYears[j], groupedYears[i]
	}
	return groupedYears
}

func gbGeneralizer(data GbPreparedData) (map[string]interface{}, bool, string, error) {
	dataMap := make(map[string]interface{})
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, true, "", err
	}
	err = json.Unmarshal(jsonData, &dataMap)
	if err != nil {
		return nil, true, "", err
	}
	return dataMap, false, "", nil
}
