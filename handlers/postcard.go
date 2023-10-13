package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerPostcardHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("postcard/send", func(c echo.Context) error {

			if !isHtmxRequest(c) {
				return apis.NewBadRequestError("Unexpected request", nil)
			}

			//get awid query param
			awid := c.QueryParam("awid")

			//check if awid is empty
			if awid == "" {
				return apis.NewBadRequestError("awid is empty", nil)
			}

			// find the artwork with the given awid
			// if not found, return 404
			// if found, render the send postcard page with the artwork data
			// if error, return 500

			_, err := app.Dao().FindRecordById("artworks", awid)

			if err != nil {
				return apis.NewNotFoundError("", err)
			}

			html, err := renderBlock("postcard:editor", map[string]any{})

			if err != nil {
				return apis.NewBadRequestError("", err)
			}

			return c.HTML(http.StatusOK, html)

		})

		e.Router.GET("postcards/:id", func(c echo.Context) error {
			return nil
		})

		e.Router.POST("postcards", func(c echo.Context) error {
			return nil
		})

		return nil
	})
}
