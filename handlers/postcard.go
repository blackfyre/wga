package handlers

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerPostcardHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("postcards", func(c echo.Context) error {
			return nil
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
