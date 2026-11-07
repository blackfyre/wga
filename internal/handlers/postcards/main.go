package postcards

import (
	"time"

	"github.com/blackfyre/wga/internal/antiabuse"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPostcardHandlers(app *pocketbase.PocketBase, captcha config.Captcha, keyring config.PostcardTokenKeyring, verifier antiabuse.Verifier, resolver requesttrust.Resolver) {
	limiter := newSubmissionLimiter(3, 10*time.Minute)
	search, err := newReplacementSearchGuard(replacementSearchesPerVisitor, replacementSearchWindow, replacementSearchConcurrency)
	if err != nil {
		// The bounds are compile-time constants, so this is unreachable;
		// a nil guard fails closed by refusing every search.
		app.Logger().Error("Postcard replacement search guard unavailable", "error", err)
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		ag := se.Router.Group("/postcard")

		ag.GET("/send", func(c *core.RequestEvent) error {
			return sendPostcard(app, c, captcha)
		})

		ag.GET("", func(c *core.RequestEvent) error {

			return viewPostcard(app, c)
		})

		ag.POST("", func(c *core.RequestEvent) error {
			return savePostcard(app, c, captcha, keyring, verifier, limiter, search, resolver)
		})
		return se.Next()
	})
}
