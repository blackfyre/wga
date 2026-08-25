package postcards

import (
	"time"

	"github.com/blackfyre/wga/internal/antiabuse"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPostcardHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy, captcha config.Captcha, keyring config.PostcardTokenKeyring, verifier antiabuse.Verifier, resolver requesttrust.Resolver) {
	limiter := newSubmissionLimiter(3, 10*time.Minute)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		ag := se.Router.Group("/postcard")

		ag.GET("/send", func(c *core.RequestEvent) error {
			return sendPostcard(app, c, captcha)
		})

		ag.GET("", func(c *core.RequestEvent) error {

			return viewPostcard(app, c)
		})

		ag.POST("", func(c *core.RequestEvent) error {
			return savePostcard(app, c, p, captcha, keyring, verifier, limiter, resolver)
		})
		return se.Next()
	})
}
