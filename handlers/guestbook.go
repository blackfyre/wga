package handlers

import (
	"github.com/blackfyre/wga/handlers/guestbook"
    "github.com/blackfyre/wga/utils"

    "github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// registerGuestbookHandlers registers the handlers for the guestbook routes.
// It takes an instance of pocketbase.PocketBase as input and adds the necessary
// route handlers to the app's router. The handlers include GET and POST methods
// for displaying and adding messages to the guestbook.
func registerGuestbookHandlers(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("/guestbook", func(c echo.Context) error {
			return guestbook.EntriesHandler(app, c)
		})

		e.Router.GET("/guestbook/add", func(c echo.Context) error {
			e.Router.Use(utils.IsHtmxRequestMiddleware)
			return guestbook.StoreEntryViewHandler(app, c)
		})

		e.Router.POST("/guestbook/add", func(c echo.Context) error {
			e.Router.Use(utils.IsHtmxRequestMiddleware)
			return guestbook.StoreEntryHandler(app, c)
		})

		return nil

	})
}
