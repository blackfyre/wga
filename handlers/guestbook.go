package handlers

import (
	"github.com/blackfyre/wga/handlers/guestbook"
	"github.com/blackfyre/wga/utils"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// registerGuestbookHandlers registers the handlers for the guestbook routes.
// It takes an instance of pocketbase.PocketBase as input and adds the necessary
// route handlers to the app's router. The handlers include GET and POST methods
// for displaying and adding messages to the guestbook.
func registerGuestbookHandlers(app *pocketbase.PocketBase) {

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		se.Router.GET("/guestbook", func(c *core.RequestEvent) error {
			return guestbook.EntriesHandler(app, c)
		})

		se.Router.GET("/guestbook/add", func(c *core.RequestEvent) error {
			return guestbook.StoreEntryViewHandler(app, c)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		se.Router.POST("/guestbook/add", func(c *core.RequestEvent) error {
			return guestbook.StoreEntryHandler(app, c)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		return se.Next()

	})
}
