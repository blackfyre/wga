package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"

	// wgamodels "blackfyre.ninja/wga/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type Page struct {
	Prerender func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error)
}

var Pages = map[string]Page{
	"artists": {
		Prerender: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
			data, _, _, _, _, err := loadArtists(c, app, "/artists")
			return data, err
		},
	},
	// Add other pages here
}

func registerDualMode(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/dualMode", func(c echo.Context) error {
			confirmedHtmxRequest := utils.IsHtmxRequest(c)

			html, err := dualModeHandler(c, app, confirmedHtmxRequest)

			if err != nil {
				return err
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}

func dualModeHandler(c echo.Context, app *pocketbase.PocketBase, confirmedHtmxRequest bool) (string, error) {
	// Get the names of the pages to display from the request parameters
	leftPageName := c.QueryParam("left")
	rightPageName := c.QueryParam("right")

	_, leftExists := Pages[leftPageName]
	_, rightExists := Pages[rightPageName]

	leftHTML, err := renderPage(c, app, leftPageName, leftExists, "default_left")
	if err != nil {
		return "", err
	}

	rightHTML, err := renderPage(c, app, leftPageName, rightExists, "default_right")
	if err != nil {
		return "", err
	}

	leftChanged := leftExists && confirmedHtmxRequest
	rightChanged := rightExists && confirmedHtmxRequest
	if leftChanged {
		return leftHTML, nil
	}

	if rightChanged {
		return rightHTML, nil
	}

	// Pass both rendered pages to the dual mode page
	dualModeData := assets.NewRenderData(app)
	dualModeData["LeftContent"] = template.HTML(leftHTML)
	dualModeData["RightContent"] = template.HTML(rightHTML)

	dualModeHtml := ""

	if !confirmedHtmxRequest {
		dualModeHtml, err = assets.RenderWithLayout(assets.Renderable{
			IsHtmx: false,
			Page:   "dualMode",
			Data:   dualModeData,
		}, "noLayout")
	} else {
		dualModeHtml, err = assets.RenderBlock("dualMode:content", dualModeData)
	}

	if err != nil {
		return "", err
	}

	return dualModeHtml, err
}

func renderPage(c echo.Context, app *pocketbase.PocketBase, pageName string, exists bool, defaultPage string) (string, error) {
	if pageName == "" {
		pageName = defaultPage
	}

	html, err := defaultPage, error(nil)
	if exists {
		html, err = renderSideBlock(Pages[pageName], c, app, "/"+pageName, pageName+":content")
	} else {
		html, err = renderDefaultSide(c, app, defaultPage)
	}

	if err != nil {
		return "", err
	}

	return html, nil
}

func renderSideBlock(page Page, c echo.Context, app *pocketbase.PocketBase, path string, block string) (string, error) {
	data, err := page.Prerender(c, app, path)
	if err != nil {
		return "", err
	}

	prerenderedHtml, err := assets.RenderBlock(block, data)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", prerenderedHtml), nil
}

func renderDefaultSide(c echo.Context, app *pocketbase.PocketBase, block string) (string, error) {
	prerenderedHtml, err := assets.RenderWithLayout(assets.Renderable{
		IsHtmx: false,
		Block:  block,
	}, "noLayout")

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", prerenderedHtml), nil
}
