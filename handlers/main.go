package handlers

import (
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
)

// RegisterHandlers registers all the handlers for the application.
// It takes a pointer to a PocketBase instance and initializes the cache.
// The cache is used to store frequently accessed data for faster access.
// The cache is automatically cleaned up every 30 minutes.
func RegisterHandlers(app *pocketbase.PocketBase) {

	p := bluemonday.NewPolicy()

	registerFeedbackHandlers(app, p)
	registerGuestbookHandlers(app)
	registerArtist(app)
	registerArtists(app)
	registerPostcardHandlers(app, p)
	registerContributors(app)
	registerStatic(app)
	registerSearchHandlers(app)
	registerHome(app)
}

// isHtmxRequest checks if the request is an htmx request by checking the value of the "HX-Request" header.
func isHtmxRequest(c echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
