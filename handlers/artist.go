package handlers

import (
	"net/http"

	"github.com/jellydator/ttlcache/v3"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerArtist(app *pocketbase.PocketBase, cache *ttlcache.Cache[string, string]) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/:name", func(c echo.Context) error {
			// name := c.PathParam("name")

			html, err := renderPage("artist", map[string]any{})

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}
