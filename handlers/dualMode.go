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
	// leftPageName := c.QueryParam("left")
	// rightPageName := c.QueryParam("right")

	// Get the Page structures for the left and right pages

	leftHTML, err := renderPageByName("artists", c, app)
	if err != nil {
		return "", err
	}

	rightHTML, err := renderPageByName("artists", c, app)
	if err != nil {
		return "", err
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

func renderPageByName(pageName string, c echo.Context, app *pocketbase.PocketBase) (string, error) {
	page := Pages[pageName]
	return renderSide(page, c, app, "/"+pageName, pageName+":content")
}

func renderSide(page Page, c echo.Context, app *pocketbase.PocketBase, path string, block string) (string, error) {
	data, err := page.Prerender(c, app, path)
	if err != nil {
		return "", err
	}

	prerenderedHtml, err := assets.RenderWithLayout(assets.Renderable{
		IsHtmx: false,
		Block:  block,
		Data:   data,
	}, "noLayout")

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", prerenderedHtml), nil
}
