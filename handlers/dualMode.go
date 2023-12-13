package handlers

import (
	"net/http"

	"blackfyre.ninja/wga/assets"
	// wgamodels "blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerDualMode(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/dualMode", func(c echo.Context) error {

			isHtmx := utils.IsHtmxRequest(c)

			html := ""
			err := error(nil)

			if err != nil {
				app.Logger().Error("Error getting welcome content", err)
			}

			data := assets.NewRenderData(app)

			data["Content"] = "Welcome to the dual mode page!"

			// html, err = assets.Render(assets.Renderable{
			// 	IsHtmx: isHtmx,
			// 	Block:  "home:content",
			// 	Data:   data,
			// })

			html, err = assets.Render(assets.Renderable{
				IsHtmx: isHtmx,
				Block:  "home:content",
				Data:   data,
			})

			// if err != nil {
			// 	// or redirect to a dedicated 404 HTML page
			// 	app.Logger().Error("Error rendering dual mode page", err)
			// 	return apis.NewNotFoundError("", err)
			// }

			c.Response().Header().Set("HX-Push-Url", "/")

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}
