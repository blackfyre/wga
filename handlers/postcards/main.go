package postcards

import (
	"github.com/blackfyre/wga/utils"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("postcard/send", func(c *core.RequestEvent) error {
			return sendPostcard(app, c)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		e.Router.GET("postcards", func(c *core.RequestEvent) error {

			return viewPostcard(app, c)
		})

		e.Router.POST("postcards", func(c *core.RequestEvent) error {
			return savePostcard(app, c, p)
		}).BindFunc(utils.IsHtmxRequestMiddleware)
		return nil
	})
}
