package handlers

import (
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
	Prerender func(c echo.Context, app *pocketbase.PocketBase, url string) (string, error)
}

var Pages = map[string]Page{
	"artists": {
		Prerender: func(c echo.Context, app *pocketbase.PocketBase, url string) (string, error) {
			html, err := loadArtists(c, app, "/artists")
			return html, err
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

	leftPageName = "artists"
	rightPageName = "artists"

	// Get the Page structures for the left and right pages
	leftPage := Pages[leftPageName]
	rightPage := Pages[rightPageName]

	// Render the left and right pages
	leftHTML, err := leftPage.Prerender(c, app, "/artists")
	if err != nil {
		return "", err
	}
	rightHTML, err := rightPage.Prerender(c, app, "/artists")
	if err != nil {
		return "", err
	}

	// Pass both rendered pages to the dual mode page
	dualModeData := assets.NewRenderData(app)
	dualModeData["LeftContent"] = template.HTML(leftHTML)
	dualModeData["RightContent"] = template.HTML(rightHTML)

	dualModeHtml := ""

	if !confirmedHtmxRequest {
		dualModeHtml, err = assets.RenderBlock("dualMode:content", dualModeData)
	} else {
		dualModeHtml, err = assets.Render(assets.Renderable{
			IsHtmx: confirmedHtmxRequest,
			Block:  "dualMode:content",
			Data:   dualModeData,
		})
	}

	if err != nil {
		return "", err
	}

	return dualModeHtml, err
}
