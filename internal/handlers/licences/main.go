package licences

import (
	"bytes"
	"net/http"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/open-source-licences", func(c *core.RequestEvent) error {
			ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Open-source licences")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Third-party software included in the Web Gallery of Art.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, tmplUtils.AssetUrl("/open-source-licences"))
			c.Response.Header().Set("HX-Push-Url", utils.GenerateCurrentRelativePageUrl(c))

			var buffer bytes.Buffer
			if err := pages.LicencesPage(assets.OpenSourceLicencesHTML).Render(ctx, &buffer); err != nil {
				return utils.ServerFaultError(c)
			}

			return c.HTML(http.StatusOK, buffer.String())
		})

		return se.Next()
	})
}
