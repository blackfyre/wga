package artists

import (
	"bytes"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// processArtists is a thin request/render adapter for the artist index. It
// delegates parsing, querying, and read-model assembly to buildArtistIndexView
// and only maps the result onto the HTTP response.
func processArtists(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	view, canonicalURL, err := buildArtistIndexView(app, c.Request.URL.Query())
	if err != nil {
		app.Logger().Error("Build artist index", "error", err)
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	c.Response.Header().Set("HX-Push-Url", canonicalURL)

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Artists")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Check out the artists in the gallery.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, utils.AssetUrl(canonicalURL))

	var buffer bytes.Buffer
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err = pages.ArtistsBlock(view).Render(ctx, &buffer)
	} else {
		err = pages.ArtistsPage(view).Render(ctx, &buffer)
	}
	if err != nil {
		app.Logger().Error("Error rendering artists", "error", err.Error())
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buffer.String())
}

func RegisterHandlers(app *pocketbase.PocketBase, environment config.Environment) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		ag := se.Router.Group("/artists")

		ag.GET("", func(c *core.RequestEvent) error {
			return processArtists(app, c)
		})

		ag.GET("/{name}", func(e *core.RequestEvent) error {
			return processArtist(e, app)
		})

		ag.GET("/{name}/selections/{selectionID}", func(e *core.RequestEvent) error {
			return processSelection(e, app)
		})

		ag.GET("/{name}/{awid}", func(e *core.RequestEvent) error {
			return processArtwork(e, app, environment)
		})
		return se.Next()
	})
}
