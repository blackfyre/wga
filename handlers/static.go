package handlers

import (
	"context"
	"io/fs"
	"net/http"
	"os"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/assets/templ/error_pages"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// getFilePublicSystem returns an http.FileSystem that represents the public files system.
// It retrieves the public files from the assets.PublicFiles and creates a sub file system.
// It then returns the http.FS representation of the file system.
func getFilePublicSystem() http.FileSystem {
	fsys, err := fs.Sub(assets.PublicFiles, "public")

	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

// registerStatic registers the static routes for the application.
// It adds a middleware to serve static assets and a handler to serve static pages.
// The static pages are retrieved from the database based on the slug parameter in the URL.
// If the request is an Htmx request, only the content block is rendered, otherwise the entire page is rendered.
// The function returns an error if there was a problem registering the routes.
func registerStatic(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Assets
		assetHandler := http.FileServer(getFilePublicSystem())
		e.Router.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets/", assetHandler)))

		// Sitemap
		e.Router.GET("/sitemap/*", apis.StaticDirectoryHandler(os.DirFS("./wga_sitemap"), false))

		// "Static" pages
		e.Router.GET("/pages/:slug", func(c echo.Context) error {

			slug := c.PathParam("slug")
			fullUrl := c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()

			page, err := models.FindStaticPageBySlug(app.Dao(), slug)

			if err != nil {
				app.Logger().Error("Error retrieving static page", "page", slug, "error", err)

				return utils.NotFoundError(c)
			}

			content := pages.StaticPageDTO{
				Title:   page.Title,
				Content: page.Content,
				Url:     "/pages/" + page.Slug,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, page.Title)
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, page.Content)
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

			c.Response().Header().Set("HX-Push-Url", fullUrl)
			return pages.StaticPage(content).Render(ctx, c.Response().Writer)

		})

		e.Router.GET("/error_404", func(c echo.Context) error {
			c.Response().Header().Set("HX-Push-Url", "/error_404")
			return error_pages.NotFoundPage().Render(context.Background(), c.Response().Writer)
		})

		return nil
	})
}
