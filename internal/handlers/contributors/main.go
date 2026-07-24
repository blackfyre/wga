package contributors

import (
	"bytes"
	"context"
	"net/http"

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
		se.Router.GET("/contributors", func(c *core.RequestEvent) error {
			fullUrl := tmplUtils.AssetUrl("/contributors")
			pushUrl := utils.GenerateCurrentRelativePageUrl(c)
			repo := repositories.NewContributorsRepository(app)

			contributors, source, err := repo.GetContributorsWithSource(c.Request.Context())
			if err != nil {
				return contributorServerError(app, c, "fetch_error", err)
			}

			if source == repositories.ContributorsSourceFileFallback {
				logging.RequestLogger(app, c).Warn("Contributors fallback served",
					"event", "contributors.request.completed",
					"outcome", "fallback",
					"source", source,
				)
			}

			content := pages.ContributorsPageDTO{
				Contributors: contributors,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Contributors")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "The people who have contributed to the Web Gallery of Art.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

			c.Response.Header().Set("HX-Push-Url", pushUrl)
			c.Response.Header().Set("X-WGA-Contributors-Source", string(source))

			// Create a bytes buffer to write the response to
			var buf bytes.Buffer

			err = pages.ContributorsPage(content).Render(ctx, &buf)

			if err != nil {
				return contributorServerError(app, c, "render_error", err)
			}

			return c.HTML(http.StatusOK, buf.String())

		})

		return se.Next()
	})
}

func contributorServerError(app core.App, c *core.RequestEvent, outcome string, err error) error {
	logging.RequestLogger(app, c).Error("Contributors request failed",
		"event", "contributors.request.failed",
		"outcome", outcome,
		"error_type", logging.ErrorType(err),
		"error", logging.Redact(err),
	)

	return c.InternalServerError("Unable to load contributors.", nil)
}
