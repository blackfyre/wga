package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerArtist(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		e.Router.GET("artists/:name", func(c echo.Context) error {
			slug := c.PathParam("name")

			artist, err := app.Dao().FindRecordsByFilter("artists", "slug = '"+slug+"'", "+name", 1, 0)

			if err != nil {
				return apis.NewNotFoundError("", err)
			}

			works, err := app.Dao().FindRecordsByFilter("artworks", "author = '"+artist[0].GetString("id")+"'", "+title", 100, 0)

			if err != nil {
				return apis.NewNotFoundError("", err)
			}

			data := map[string]any{
				"Name":  artist[0].GetString("name"),
				"Bio":   artist[0].GetString("bio"),
				"Works": []map[string]string{},
			}

			for _, w := range works {
				data["Works"] = append(data["Works"].([]map[string]string), map[string]string{
					"Title":   w.GetString("title"),
					"Comment": w.GetString("comment"),
					"Image":   generateFileUrl(app, "artworks", w.GetString("id"), w.GetString("image")),
					"Thumb":   generateThumbUrl(app, "artworks", w.GetString("id"), w.GetString("image"), "320x240"),
				})
			}

			html := ""

			if isHtmxRequest(c) {
				html, err = renderBlock("artist:content", data)
			} else {
				html, err = renderPage("artist", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			c.Response().Header().Set("HX-Push-Url", "/artists/"+slug)

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}
