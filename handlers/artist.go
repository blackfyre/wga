package handlers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

func RegisterArtist(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)
		registry := template.NewRegistry()

		e.Router.GET("/:name", func(c echo.Context) error {
			name := c.PathParam("name")

			html, err := registry.LoadFiles(
				"views/layout.html",
				"views/artist.html",
			).Render(map[string]any{
				"name": name,
			})

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
