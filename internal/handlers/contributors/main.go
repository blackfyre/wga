package contributors

import (
	"bytes"
	"context"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/contributors", func(c *core.RequestEvent) error {
			fullUrl := tmplUtils.AssetUrl("/contributors")
			repo := repositories.NewContributorsRepository(app)

			contributors, source, err := repo.GetContributorsWithSource()
			if err != nil {
				app.Logger().Error("Error getting contributors", "error", err)
				return apis.NewApiError(500, err.Error(), err)
			}

			if source == repositories.ContributorsSourceFileFallback {
				app.Logger().Warn("Contributors endpoint served fallback data", "source", source)
			}

			content := pages.ContributorsPageDTO{
				Contributors: contributors,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Contributors")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "The people who have contributed to the Web Gallery of Art.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

			c.Response.Header().Set("HX-Push-Url", fullUrl)
			c.Response.Header().Set("X-WGA-Contributors-Source", string(source))

			// Create a bytes buffer to write the response to
			var buf bytes.Buffer

			err = pages.ContributorsPage(content).Render(ctx, &buf)

			if err != nil {
				app.Logger().Error("Error rendering artwork page", "error", err.Error())
				return c.Error(http.StatusInternalServerError, "failed to render response template", err)
			}

			return c.HTML(http.StatusOK, buf.String())

		})

		return se.Next()
	})
}
