package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"golang.org/x/net/html"

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
	"clear": {
		Prerender: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	},
	// Add other pages here
}

func registerDualMode(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/dualMode", func(c echo.Context) error {
			currentUrl := c.Request().URL.String()
			c.Response().Header().Set("HX-Push-Url", currentUrl)

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
	leftParam := c.QueryParam("left")
	rightParam := c.QueryParam("right")

	leftHTML, err := "", error(nil)
	rightHTML, err := "", error(nil)

	leftHTML, err = renderPage(c, app, leftParam, "default_left")
	if err != nil {
		return "", err
	}

	rightHTML, err = renderPage(c, app, rightParam, "default_right")
	if err != nil {
		return "", err
	}

	if _, exists := Pages[leftParam]; exists && confirmedHtmxRequest {
		return leftHTML, nil
	}

	if _, exists := Pages[rightParam]; exists && confirmedHtmxRequest {
		return rightHTML, nil
	}

	// Pass both rendered pages to the dual mode page
	testHtml, err := linkUpdater(leftHTML)
	if err != nil {
		return "", err
	}
	dualModeData := assets.NewRenderData(app)
	dualModeData["LeftContent"] = template.HTML(testHtml)
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

func renderPage(c echo.Context, app *pocketbase.PocketBase, param string, defaultPage string) (string, error) {
	if param == "" {
		param = defaultPage
	}

	if page, exists := Pages[param]; exists {
		return renderSideBlock(page, c, app, "/"+param, param+":content")
	}

	return renderDefaultSide(c, app, defaultPage)
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

func linkUpdater(htmlStr string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i := range n.Attr {
				if n.Attr[i].Key == "hx-get" {
					n.Attr[i].Val += "a" // add the extra character
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var b strings.Builder
	html.Render(&b, doc)
	return b.String(), nil
}
