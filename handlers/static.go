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

func registerStatic(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// fmt.Println(assets.PublicFiles.ReadFile("public/css/style.css"))
		e.Router.GET("/assets/*", staticEmbeddedHandler(assets.PublicFiles))
		return nil
	})
}

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
