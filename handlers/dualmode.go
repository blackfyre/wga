package handlers

import (
	"context"

	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerDualMode(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/dualmode", func(c echo.Context) error {
			content := pages.DualModePage{}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Dual Mode")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "This is a dual mode page. It can render two different pages on the same URL.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, c.Scheme()+"://"+c.Request().Host+c.Request().URL.String())

			c.Response().Header().Set("HX-Push-Url", "/dualmode")
			err := pages.DualModePageWrapped(content).Render(ctx, c.Response().Writer)

			if err != nil {
				app.Logger().Error("Error rendering home page", err)
				return utils.ServerFaultError(c)
			}

			return nil
		})

		return nil
	})
}
