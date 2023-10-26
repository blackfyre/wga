package search

import (
	"net/http"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func search(app *pocketbase.PocketBase, e *core.ServeEvent, c echo.Context) error {
	htmx := utils.IsHtmxRequest(c)
	// filters := buildFilters(c)

	td := map[string]any{}

	td["ArtFormOptions"], _ = getArtFormOptions(app)
	td["ArtTypeOptions"], _ = getArtTypesOptions(app)

	html := ""
	err := error(nil)

	if htmx {
		html, err = assets.RenderBlock("search:content", td)
	} else {
		html, err = assets.RenderPage("search", td)
	}

	if err != nil {
		return apis.NewNotFoundError("", err)
	}

	c.Response().Header().Set("HX-Push-Url", "/search")

	return c.HTML(http.StatusOK, html)

}

// RegisterSearchHandlers registers search handlers to the given PocketBase app.
func RegisterSearchHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/search", func(c echo.Context) error {
			return search(app, e, c)
		})
		return nil
	})
}
