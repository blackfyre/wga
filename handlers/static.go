package handlers

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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

// registerStatic registers the static routes for the application.
// It adds a middleware to serve static assets and a handler to serve static pages.
// The static pages are retrieved from the database based on the slug parameter in the URL.
// If the request is an Htmx request, only the content block is rendered, otherwise the entire page is rendered.
// The function returns an error if there was a problem registering the routes.
func registerStatic(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Assets
		e.Router.GET("/assets/*", staticEmbeddedHandler(assets.PublicFiles))

		// Sitemap
		e.Router.GET("/sitemap/*", apis.StaticDirectoryHandler(os.DirFS("./wga_sitemap"), false))

		// "Static" pages
		e.Router.GET("/pages/:slug", func(c echo.Context) error {

			isHtmx := utils.IsHtmxRequest(c)
			slug := c.PathParam("slug")

			page, err := models.FindStaticPageBySlug(app.Dao(), slug)

			if err != nil {
				app.Logger().Error("Error retrieving static page", "page", slug, err)

				return utils.NotFoundError(c)
			}

			content := pages.StaticPageDTO{
				Title:   page.Title,
				Content: page.Content,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, page.Title)

			if isHtmx {
				c.Response().Header().Set("HX-Push-Url", "/pages/"+slug)
				return pages.StaticPageBlock(content).Render(ctx, c.Response().Writer)

			} else {
				return pages.StaticPage(content).Render(ctx, c.Response().Writer)

			}

		})

		e.Router.GET("/error_404", func(c echo.Context) error {
			c.Response().Header().Set("HX-Push-Url", "/error_404")
			return error_pages.NotFoundPage().Render(context.Background(), c.Response().Writer)
		})

		return nil
	})
}

// staticEmbeddedHandler returns an echo.HandlerFunc that serves static files embedded in the given embed.FS.
// The function takes a context object and returns an error. It first unescapes the URL path and then constructs
// the file path by cleaning and trimming the path parameter. If the file exists, it is served using the echo.Context's
// FileFS method. If the file does not exist, the function serves the 404.html file from the public directory.
func staticEmbeddedHandler(embedded embed.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		p := c.PathParam("*")

		// escape url path
		tmpPath, err := url.PathUnescape(p)
		if err != nil {
			return fmt.Errorf("failed to unescape path variable: %w", err)
		}
		p = tmpPath

		name := "public/" + filepath.ToSlash(filepath.Clean(strings.TrimPrefix(p, "/")))

		fileErr := c.FileFS(name, embedded)

		if fileErr != nil && errors.Is(fileErr, echo.ErrNotFound) {
			return c.Redirect(404, "/error_404")
		}

		return fileErr
	}
}
