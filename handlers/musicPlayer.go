package handlers

import (
	"fmt"
	"net/http"

	"blackfyre.ninja/wga/assets"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerMusicPlayer(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/musicPlayer", func(c echo.Context) error {

			fs, err := app.NewFilesystem()

			if err != nil {
				return err
			}

			defer fs.Close()

			welcomeText, err := getWelcomContent(app)

			if err != nil {
				fmt.Println(err)
			}

			artistCount, err := getArtistCount(app)

			if err != nil {
				fmt.Println(err)
			}

			artworkCount, err := getArtworkCount(app)

			if err != nil {
				fmt.Println(err)
			}

			isHtmx := isHtmxRequest(c)

			html := ""

			data := map[string]any{
				"Content":      welcomeText,
				"ArtistCount":  artistCount,
				"ArtworkCount": artworkCount,
			}

			if isHtmx {
				html, err = assets.RenderBlock("home:content", data)

			} else {
				html, err = assets.RenderPage("home", data)
			}

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			c.Response().Header().Set("HX-Push-Url", "/")

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}

