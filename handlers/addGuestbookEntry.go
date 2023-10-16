package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Data struct {
	Title         string
	FirstContent  string
	SecondContent string
	MainContent   []map[string]string
	LatestEntries string
}

type PreparedData struct {
	Title                   string
	FirstContent            string
	SecondContent           string
	MainContent             []map[string]string
	LatestEntries           string
	CalendarYears           [][]string
	SearchExpressionPresent bool
	SearchExpression        string
}

func registerAddGuestbookEntry(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/addGuestbookEntry", func(c echo.Context) error {
			confirmedHtmxRequest := isHtmxRequest(c)
			currentUrl := c.Request().URL.String()
			c.Response().Header().Set("HX-Push-Url", currentUrl)

			cacheKey := gbAddEntrySetCacheSettings(confirmedHtmxRequest)
			shouldReturn, returnValue := gbAddEntryIsCached(app, cacheKey, c)
			if shouldReturn {
				return returnValue
			}

			data, err := gbAddEntryGetData(app)

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			html, err := gbAddEntryRender(confirmedHtmxRequest, data)

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			c.Response().Header().Set("HX-Push-Url", "/addGuestbookEntry")

			return c.HTML(http.StatusOK, html)
		})

		return nil

	})
}

func gbAddEntrySetCacheSettings(confirmedHtmxRequest bool) string {
	cacheKey := "addGuestbookEntry"

	return cacheKey
}

func gbAddEntryIsCached(app *pocketbase.PocketBase, cacheKey string, c echo.Context) (bool, error) {
	if app.Cache().Has(cacheKey) {
		html := app.Cache().Get(cacheKey).(string)
		return true, c.HTML(http.StatusOK, html)
	}
	return false, nil
}

func gbAddEntryGetData(app *pocketbase.PocketBase) (PreparedData, error) {
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

	latestEntries, err := gbGetGuestbookTextContent(app, "latestEntries")

	if err != nil {
		fmt.Println(err)
	}

	rawData := Data{
		Title:         guestBookTitle,
		FirstContent:  guestBookTextFirst,
		SecondContent: guestBookTextSecond,
		LatestEntries: latestEntries,
	}

	data := gbAddEntryPrepareDataForRender(rawData)

	return data, err
}

func gbAddEntryPrepareDataForRender(data Data) PreparedData {
	years := gbGetYears()

	preparedData := PreparedData{
		Title:                   data.Title,
		FirstContent:            data.FirstContent,
		SecondContent:           data.SecondContent,
		MainContent:             data.MainContent,
		CalendarYears:           years,
		LatestEntries:           data.LatestEntries,
	}
	return preparedData
}

func gbAddEntryRender(confirmedHtmxRequest bool, data PreparedData) (string, error) {
	dataMap, shouldReturn, html, err := gbAddEntryGeneralizer(data)
	if shouldReturn {
		return html, err
	}

	if confirmedHtmxRequest {
		blockToRender := "addGuestbookEntry:content"

		html, err := renderBlock(blockToRender, dataMap)
		return html, err
	} else {
		html, err := renderPage("addGuestbookEntry", dataMap)
		return html, err
	}
}

func gbAddEntryGeneralizer(data PreparedData) (map[string]interface{}, bool, string, error) {
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


