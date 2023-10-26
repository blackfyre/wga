package search

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func search(app *pocketbase.PocketBase, e *core.ServeEvent, c echo.Context) error {
	// htmx := isHtmxRequest(c)
	// filters := buildFilters(c)

	return nil
}

func RegisterSearchHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/search", func(c echo.Context) error {
			return search(app, e, c)
		})
		return nil
	})
}
