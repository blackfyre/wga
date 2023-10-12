package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
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

func getGuestbookContent(app *pocketbase.PocketBase) ([]map[string]string, error) {
	records, err := app.Dao().FindRecordsByFilter(
		"guestbook",
		"id != null",
		"-updated",
		0,
		0,
	)

	if err != nil {
		return nil, apis.NewBadRequestError("Invalid something", err)
	}

	preRendered := []map[string]string{}

	for _, m := range records {
		
		updateDateString := m.GetString("updated")
		updateTime, err := time.Parse("2006-01-02 15:04:05.000Z", updateDateString)
		if err != nil {
			fmt.Println("Error parsing the date:", err)
			return nil, err
		}
		formattedDateString := updateTime.Format("January 2, 2006")

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
			guestBookTextFirst, err := getGuestbookTextContent(app, "thankYou")

			if err != nil {
				fmt.Println(err)
			}

			guestBookTextSecond, err := getGuestbookTextContent(app, "guestbookPlayMusicLink")

			if err != nil {
				fmt.Println(err)
			}

			guestBookContent, err := getGuestbookContent(app)

			if err != nil {
				fmt.Println(err)
			}

			isHtmx := isHtmxRequest(c)

			html := ""

			data := map[string]any{
				"FirstContent":  guestBookTextFirst,
				"SecondContent":  guestBookTextSecond,
				"MainContent": guestBookContent,
			}

			if isHtmx {
				html, err = renderBlock("guestbook:content", data)

			} else {
				html, err = renderPage("guestbook", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			c.Response().Header().Set("HX-Push-Url", "/guestbook")

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}
