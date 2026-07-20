package postcards

import (
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy, captcha config.Captcha) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		ag := se.Router.Group("/postcard")

		ag.GET("/send", func(c *core.RequestEvent) error {
			return sendPostcard(app, c)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		ag.GET("", func(c *core.RequestEvent) error {

			return viewPostcard(app, c)
		})

		ag.POST("", func(c *core.RequestEvent) error {
			return savePostcard(app, c, p, captcha)
		}).BindFunc(utils.IsHtmxRequestMiddleware)
		return se.Next()
	})
}
