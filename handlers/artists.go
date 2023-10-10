package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerArtists(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/artists", func(c echo.Context) error {
			// name := c.PathParam("name")

			data := map[string]any{
				"Content": "",
			}

			html := ""
			err := error(nil)

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
