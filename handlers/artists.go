package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerArtists(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/artists", func(c echo.Context) error {

			limit := 30
			page := 1

			if c.QueryParam("page") != "" {
				err := error(nil)
				page, err = strconv.Atoi(c.QueryParam("page"))

				if err != nil {
					return apis.NewBadRequestError("Invalid page", err)
				}
			}

			offset := (page - 1) * limit

			data := map[string]any{
				"Content": "",
			}

			records, err := app.Dao().FindRecordsByFilter(
				"artists",
				"published = true",
				"+name",
				limit,
				offset,
			)

			if err != nil {
				return apis.NewBadRequestError("Invalid page", err)
			}

			preRendered := []map[string]string{}

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

				row := map[string]string{
					"Name":       m.GetString("name"),
					"Url":        artistUrl(m.GetString("slug")),
					"Profession": m.GetString("profession"),
					"BornDied":   normalizedBirthDeathActivity(m),
					"Schools":    strings.Join(schoolCollector, ", "),
				}

				preRendered = append(preRendered, row)
			}

			data["Content"] = preRendered

			html := ""

			if isHtmxRequest(c) {
				html, err = renderBlock("artists:content", data)
			} else {
				html, err = renderPage("artists", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			c.Response().Header().Set("HX-Push-Url", "/artists")

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}
