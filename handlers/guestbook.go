package handlers

import (
	"github.com/blackfyre/wga/handlers/guestbook"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type SearchSettings struct {
	searchExpression        string
	searchExpressionPresent bool
	filter                  string
}

type GbData struct {
	Title         string
	FirstContent  string
	SecondContent string
	MainContent   []GuestBookMessagePrepared
	LatestEntries string
}

type GbPreparedData struct {
	Title                   string
	FirstContent            string
	SecondContent           string
	MainContent             []GuestBookMessagePrepared
	LatestEntries           string
	CalendarYears           [][]string
	SearchExpressionPresent bool
	SearchExpression        string
}

type GuestBookMessage struct {
	Name          string `json:"name" form:"name" query:"name" validate:"required"`
	Email         string `json:"email" form:"email" query:"email" validate:"required"`
	Location      string `json:"location" form:"location" query:"location" validate:"required"`
	Message       string `json:"message" form:"message" query:"message" validate:"required"`
	HoneyPotName  string `json:"honey_pot_name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" query:"honey_pot_email"`
}

type GuestBookMessagePrepared struct {
	Message  string
	Name     string
	Location string
	Created  string
}

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
			return guestbook.StoreEntryViewHandler(app, c)
		})

		e.Router.POST("/guestbook/add", func(c echo.Context) error {
			return guestbook.StoreEntryHandler(app, c)
		})

		return nil

	})
}
