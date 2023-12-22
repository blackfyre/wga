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
	getData func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error)
}

var Pages = map[string]Page{
	"artists": {
		getData: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
			data, _, _, _, _, err := loadArtists(c, app, url)
			return data, err
		},
	},
	"artistS": {
		getData: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
			// i need to get from the url whatever is after /artists/
			slug := url[9:]
			data, err := getArtist(c, app, slug)
			return data, err
		},
	},
	"default_left": {
		getData: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	},
	"default_right": {
		getData: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	},
	"clear": {
		getData: func(c echo.Context, app *pocketbase.PocketBase, url string) (map[string]any, error) {
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
	params := c.QueryParams()
	leftParams := params["left"]
	rightParams := params["right"]

	leftPage, leftTarget := "default_left", "default_target"
	if len(leftParams) >= 1 {
		leftPage = leftParams[0]
		if len(leftParams) >= 2 {
			leftTarget = leftParams[1]
		}
	}

	rightPage, rightTarget := "default_right", "default_target"
	if len(rightParams) >= 1 {
		rightPage = rightParams[0]
		if len(rightParams) >= 2 {
			rightTarget = rightParams[1]
		}
	}

	leftHTML, err := renderPage(c, app, leftPage, leftTarget, leftParams)
	if err != nil {
		return "", err
	}

	rightHTML, err := renderPage(c, app, rightPage, rightTarget, rightParams)
	if err != nil {
		return "", err
	}

	if c.QueryParam(("right")) == "artists/aachen-hans-von" {
		rightPage = "artistS"
	}

	if _, exists := Pages[leftPage]; exists && confirmedHtmxRequest && leftPage != "default_left" {
		return leftHTML, nil
	}

	if _, exists := Pages[rightPage]; exists && confirmedHtmxRequest && rightPage != "default_right" {
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

func renderPage(c echo.Context, app *pocketbase.PocketBase, targetPage string, target string, params []string) (string, error) {
	html, err := renderSideBlock(targetPage, c, app, "/"+targetPage, targetPage+":content")
	if err != nil {
		return "", err
	}

	linkUpdatedHtml, err := linkUpdater(html, target, params)
	if err != nil {
		return "", err
	}

	// Update the hx-target attribute before returning the HTML
	updatedHtml, err := updateHxTarget(linkUpdatedHtml, target)
	if err != nil {
		return "", err
	}

	return updatedHtml, nil

}

func renderSideBlock(targetPage string, c echo.Context, app *pocketbase.PocketBase, path string, block string) (string, error) {
	var prerenderedHtml string
	var err error

	fmt.Println("targetPage: ", targetPage)
	if strings.HasPrefix(targetPage, "artists") && len(targetPage) > 8 {
		targetPage = "artistS"
		block = "artist:content"
	}
	if targetPage != "default_left" && targetPage != "default_right" {
		data, err := Pages[targetPage].getData(c, app, path)
		if err != nil {
			return "", err
		}

		prerenderedHtml, err = assets.RenderBlock(block, data)
		if err != nil {
			return "", err
		}
	} else {
		prerenderedHtml, err = assets.RenderWithLayout(assets.Renderable{
			IsHtmx: false,
			Block:  block,
		}, "noLayout")

		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%v", prerenderedHtml), nil
}

func linkUpdater(htmlStr string, target string, params []string) (string, error) {
	fmt.Println("params: ", params)
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i := range n.Attr {
				href := n.Attr[i].Val
				// remove the leading slash
				href = href[1:]
				if n.Attr[i].Key == "hx-get" && target != "" && href != "" {
					origin := "left"
					if target == "left" {
						origin = "right"
					}
					// add the href to the end of the hx-get attribute
					n.Attr[i].Val = "/dualMode?" + origin + "=" + params[0] + "&" + target + "=" + href
				}
				// remove href
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

func updateHxTarget(htmlStr string, target string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i := range n.Attr {
				if n.Attr[i].Key == "hx-target" {
					if target == "right" {
						n.Attr[i].Val = "#rc-area" // update the hx-target attribute
					}
					if target == "left" {
						n.Attr[i].Val = "#lc-area" // update the hx-target attribute
					}
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
