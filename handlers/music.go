package handlers

import (
	"net/http"

	"blackfyre.ninja/wga/assets"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerMusic(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/music", func(c echo.Context) error {
			html, err := "", error(nil)
			isHtmx := isHtmxRequest(c)
			data := map[string]any{}

			setUrl(c, "")

			// html, err = assets.RenderPage("music", data)

			if isHtmx {
				html, err = assets.RenderBlock("music:content", data)

			} else {
				html, err = assets.RenderPage("music", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}

