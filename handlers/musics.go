package handlers

import (
	"net/http"

	"blackfyre.ninja/wga/assets"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerMusicHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("musics", func(c echo.Context) error {
			html, err := "", error(nil)
			isHtmx := isHtmxRequest(c)
			cacheKey := "musics"

			setUrl(c, "")

			found := app.Cache().Has(cacheKey)

			// TODO: implement data getter
			musicList, years := getMock()

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

func getMock() ([]map[string]interface{}, []string) {
	musicList := []map[string]interface{}{
		{
			"century": "12",
			"composers": []map[string]interface{}{
				{
					"composer": "Anonymus",
					"date":     "1100-1199",
					"language": "French",
					"songs": []map[string]interface{}{
						{
							"name": "Gregorian Chants",
							"time": "3 minutes",
							"source":  "anonymous_conductus.mp3",
						},
						{
							"name": "Viderunt omnes, organum",
							"time": "3 minutes",
						},
					},
				},
				{
					"composer": "Hildegard von Bingen",
					"date":     "1100-1199",
					"language": "French",
					"songs": []map[string]interface{}{
						{
							"name": "O viridissima virga",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "13",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "14",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "16",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
						{
							"name": "Loca2",
							"time": "3 minutes",
						},
						{
							"name": "Loca3",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "17",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
						{
							"name": "Loca2",
							"time": "3 minutes",
						},
						{
							"name": "Loca3",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "18",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
						{
							"name": "Loca2",
							"time": "3 minutes",
						},
						{
							"name": "Loca3",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "19",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
						{
							"name": "Loca2",
							"time": "3 minutes",
						},
						{
							"name": "Loca3",
							"time": "3 minutes",
						},
					},
				},
			},
		},
		{
			"century": "20",
			"composers": []map[string]interface{}{
				{
					"composer": "Shakira",
					"date":     "1200-1299",
					"language": "German",
					"songs": []map[string]interface{}{
						{
							"name": "Loca",
							"time": "3 minutes",
						},
						{
							"name": "Loca2",
							"time": "3 minutes",
						},
						{
							"name": "Loca3",
							"time": "3 minutes",
						},
					},
				},
			},
		},
	}

	years := []string{}

	for _, century := range musicList {
		years = append(years, century["century"].(string))
	}
	return musicList, years
}
