package handlers

import (
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

func getGuestbookTextContent(app *pocketbase.PocketBase, content string) (string, error) {
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

func getGuestbookContent(app *pocketbase.PocketBase, filter string, searchExpression string) ([]map[string]string, error) {
	records, err := app.Dao().FindRecordsByFilter(
		"guestbook",
		filter,
		"-updated",
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
			"Message":    m.GetString("message"),
			"Name":       m.GetString("name"),
			"Location":	  m.GetString("location"),
			"Updated":    formattedDateString,
		}

		preRendered = append(preRendered, row)
	}

	return preRendered, err
}

func registerGuestbook(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/guestbook", func(c echo.Context) error {
			searchExpression := ""
			searchExpressionPresent := false
			confirmedHtmxRequest := isHtmxRequest(c)
			currentUrl := c.Request().URL.String()
			c.Response().Header().Set("HX-Push-Url", currentUrl)

			if c.QueryParam("q") != "" {
				searchExpression = c.QueryParam("q")
			}

			if c.QueryParams().Has("q") {
				searchExpressionPresent = true
			}

			cacheKey := "guestbook:" + ":" + searchExpression

			if confirmedHtmxRequest {
				cacheKey = cacheKey + ":htmx"
			}

			if searchExpressionPresent {
				cacheKey = cacheKey + ":search"
			}

			if app.Cache().Has(cacheKey) {
				html := app.Cache().Get(cacheKey).(string)
				return c.HTML(http.StatusOK, html)
			} else {

				filter := "id != null"

				if searchExpression != "" {
					filter = filter + " && created ~ {:searchExpression}"
				}

				guestBookTextFirst, err := getGuestbookTextContent(app, "thankYou")

				if err != nil {
					fmt.Println(err)
				}

				guestBookTextSecond, err := getGuestbookTextContent(app, "guestbookPlayMusicLink")

				if err != nil {
					fmt.Println(err)
				}

				guestBookContent, err := getGuestbookContent(app, filter, searchExpression)

				if err != nil {
					fmt.Println(err)
				}

				isHtmx := isHtmxRequest(c)

				html := ""

				years := getYears()

				data := map[string]any{
					"FirstContent":  guestBookTextFirst,
					"SecondContent":  guestBookTextSecond,
					"MainContent": guestBookContent,
					"CalendarYears": years,
				}

				if isHtmx {
					html, err = renderBlock("guestbook:search-results", data)

				} else {
					html, err = renderPage("guestbook", data)
				}

				if err != nil {
					// or redirect to a dedicated 404 HTML page
					return apis.NewNotFoundError("", err)
				}

				c.Response().Header().Set("HX-Push-Url", "/guestbook")

				return c.HTML(http.StatusOK, html)
			}
		})

		return nil

	})
}

func getYears() [][]string {
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
