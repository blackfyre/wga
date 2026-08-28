package landing

import (
	"bytes"
	"net/http"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// This is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		se.Router.GET("/", func(c *core.RequestEvent) error {
			content, err := buildHomePage(repositories.NewLandingRepository(app), time.Now().UTC())
			if err != nil {
				logging.RequestLogger(app, c).Error("Build home page", "error", err)
				return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
			}

			ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Web Gallery of Art | Explore artists and artworks")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Explore artists, artworks, and side-by-side comparisons in the Web Gallery of Art.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, tmplUtils.AssetUrl("/"))

			c.Response.Header().Set("HX-Push-Url", "/")

			var buff bytes.Buffer

			err = pages.HomePageWrapped(content).Render(ctx, &buff)

			if err != nil {
				logging.RequestLogger(app, c).Error("Render home page", "error", err)
				return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
			}

			return c.HTML(http.StatusOK, buff.String())
		})

		return se.Next()
	})
}
