package health

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterHandlers registers the liveness endpoint.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/health", func(c *core.RequestEvent) error {
			return c.String(http.StatusOK, "ok")
		})

		return se.Next()
	})
}
