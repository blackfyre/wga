package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"blackfyre.ninja/wga/assets"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Song struct {
	Title  string
	URL    string
	Source []string
}

type Composer struct {
	Name     string
	Date     string
	Language string
	Songs    []Song
}

type Century struct {
	Century   string
	Composers []Composer
}

func registerMusicHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("musics", func(c echo.Context) error {
			html, err := "", error(nil)
			isHtmx := isHtmxRequest(c)
			cacheKey := "musics"

			setUrl(c, "")

			found := app.Cache().Has(cacheKey)

			// TODO: implement data getter
			musicList := getMusics()

			years := []string{}
			for _, century := range musicList {
				years = append(years, century.Century)
			}
			if found {
				html = app.Cache().Get(cacheKey).(string)
			} else {
				data := map[string]any{
					"Centuries": years,
					"MusicList": musicList,
				}

				if isHtmx {
					html, err = assets.RenderBlock("musics:content", data)
				} else {
					html, err = assets.RenderPageWithLayout("musics", "noLayout", data)
				}

				if err != nil {
					// or redirect to a dedicated 404 HTML page
					return apis.NewNotFoundError("", err)
				}
			}

			return c.HTML(http.StatusOK, html)
		})

		e.Router.GET("musics/:name", func(c echo.Context) error {
			slug := c.PathParam("name")
			cacheKey := "music:" + slug

			if isHtmxRequest(c) {
				cacheKey = cacheKey + "-htmx"
			}

			html := ""
			err := error(nil)

			data := map[string]any{
				"Title":    "Gregorian Chants",
				"Composer": "Anonymus",
				"Date":     "1123",
				"Source":   "anonymous_conductus.mp3",
			}

			if isHtmxRequest(c) {
				html, err = assets.RenderBlock("music:content", data)
			} else {
				html, err = assets.RenderPageWithLayout("musics/music", "noLayout", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			app.Cache().Set(cacheKey, html)

			c.Response().Header().Set("HX-Push-Url", "/musics/"+slug)

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}

func getMusics() []Century {
	var data []Century

	fileData, err := os.ReadFile("./assets/reference/musics.json")

	if err != nil {
		fmt.Println("Error reading file:", err)
	}

	err = json.Unmarshal(fileData, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data:", err)
	}

	musicList := data
	return musicList
}
