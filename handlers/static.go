package handlers

import (
	"embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
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

			slug := c.PathParam("slug")

			page, err := models.FindStaticPageBySlug(app.Dao(), slug)

			if err != nil {
				app.Logger().Error("Error retrieving static page", "page", slug, err)
				return err
			}

			d := assets.NewRenderData(app)

			d["Title"] = page.Title
			d["Slug"] = page.Slug
			d["Content"] = page.Content

			html, err := assets.Render(assets.Renderable{
				IsHtmx: utils.IsHtmxRequest(c),
				Block:  "staticpage:content",
				Data:   d,
			})

			if err != nil {
				app.Logger().Error("Error rendering static page", "page", slug, err)
				return err
			}

			c.Response().Header().Set("HX-Push-Url", "/pages/"+slug)

			return c.HTML(http.StatusOK, html)

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
			return c.FileFS("public/404.html", embedded)
		}

		return fileErr
	}
}
