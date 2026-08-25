package inspire

import (
	"bytes"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func inspirationHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	content, err := inspirationWorks(app)
	if err != nil {
		logging.RequestLogger(app, c).Error("Build inspiration page", "error", err)
		return utils.ServerFaultError(c)
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Inspiration")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Explore a shuffled selection of works from the Web Gallery of Art collection.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, tmplUtils.AssetUrl("/inspire"))

	var buff bytes.Buffer

	c.Response.Header().Set("HX-Push-Url", "/inspire")
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err = pages.InspirationContent(content).Render(ctx, &buff)
	} else {
		err = pages.InspirePage(content).Render(ctx, &buff)
	}

	if err != nil {
		logging.RequestLogger(app, c).Error("Render inspiration page", "error", err)
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

// RegisterHandlers registers the HTTP handlers for the PocketBase application.
// It binds a function to the OnServe event, which sets up a GET route for "/inspire".
// When the "/inspire" route is accessed, the inspirationHandler function is called.
//
// Parameters:
//   - app: A pointer to the PocketBase application instance.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/inspire", func(c *core.RequestEvent) error {
			return inspirationHandler(app, c)
		})
		return se.Next()
	})
}
