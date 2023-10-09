package handlers

import (
	"embed"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"blackfyre.ninja/wga/assets"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// registerStatic registers the static assets handler to the PocketBase app.
// It adds a BeforeServe event to the app that serves the static assets from the embedded files.
func registerStatic(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// fmt.Println(assets.PublicFiles.ReadFile("public/css/style.css"))
		e.Router.GET("/assets/*", staticEmbeddedHandler(assets.PublicFiles))
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
