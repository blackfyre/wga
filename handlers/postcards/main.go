package postcards

import (
	"github.com/blackfyre/wga/utils"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("postcard/send", func(c *core.RequestEvent) error {
			return sendPostcard(app, c)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		se.Router.GET("postcards", func(c *core.RequestEvent) error {

			return viewPostcard(app, c)
		})

		se.Router.POST("postcards", func(c *core.RequestEvent) error {
			return savePostcard(app, c, p)
		}).BindFunc(utils.IsHtmxRequestMiddleware)
		return se.Next()
	})
}
