package static

import (
	"bytes"
	"context"
	"io/fs"
	"os"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/assets/templ/error_pages"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func getFilePublicSystem() fs.FS {
	fsys, err := fs.Sub(assets.PublicFiles, "public")

	if err != nil {
		panic(err)
	}

	return fsys
}

// RegisterHandlers registers the static routes for the application.
// It adds a middleware to serve static assets and a handler to serve static pages.
// The static pages are retrieved from the database based on the slug parameter in the URL.
// If the request is an Htmx request, only the content block is rendered, otherwise the entire page is rendered.
// The function returns an error if there was a problem registering the routes.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Assets
		if app.IsDev() {
			se.Router.GET("/assets/{path...}", apis.Static(os.DirFS("./assets/public"), false))
		} else {
			se.Router.GET("/assets/{path...}", apis.Static(getFilePublicSystem(), false))
		}

		// Sitemap
		se.Router.GET("/sitemap/*", apis.Static(os.DirFS("./wga_sitemap"), false))

		// "Static" pages
		se.Router.GET("/pages/:slug", func(c *core.RequestEvent) error {

			slug := c.Request.PathValue("slug")
			fullUrl := tmplUtils.AssetUrl("/pages/" + slug)

			page, err := app.FindFirstRecordByData("static_pages", "slug", slug)

			if err != nil {
				app.Logger().Error("Error retrieving static page", "page", slug, "error", err)

				return utils.NotFoundError(c)
			}

			content := pages.StaticPageDTO{
				Title:   page.GetString("title"),
				Content: page.GetString("content"),
				Url:     "/pages/" + page.GetString("slug"),
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, page.GetString("title"))
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, page.GetString("content"))
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

			c.Response.Header().Set("HX-Push-Url", fullUrl)

			var buf bytes.Buffer

			pages.StaticPage(content).Render(ctx, &buf)

			return c.HTML(200, buf.String())

		})

		se.Router.GET("/error_404", func(c *core.RequestEvent) error {
			c.Response.Header().Set("HX-Push-Url", "/error_404")

			var buffer bytes.Buffer

			err := error_pages.NotFoundPage().Render(context.Background(), &buffer)

			if err != nil {
				app.Logger().Error("Error rendering error page", "error", err)
				return c.HTML(500, "Internal server error")
			}

			return c.HTML(404, buffer.String())
		})

		return se.Next()
	})
}
