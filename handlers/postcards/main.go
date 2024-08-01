package postcards

import (
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("postcard/send", func(c echo.Context) error {

			e.Router.Use(utils.IsHtmxRequestMiddleware)

			return sendPostcard(app, c)
		})

		e.Router.GET("postcards", func(c echo.Context) error {

			return viewPostcard(app, c)
		})

		e.Router.POST("postcards", func(c echo.Context) error {

			e.Router.Use(utils.IsHtmxRequestMiddleware)

			return savePostcard(app, c, p)
		})
		return nil
	})
}
