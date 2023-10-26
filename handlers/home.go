package handlers

import (
	"fmt"
	"log"
	"net/http"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Content struct {
	FieldName string `db:"name" json:"name"`
	Content   string `db:"content" json:"content"`
}

type counter struct {
	C int `db:"c" json:"c"`
}

func getWelcomContent(app *pocketbase.PocketBase) (string, error) {

	found := app.Cache().Has("strings:welcome")

	if found {
		return app.Cache().Get("strings:welcome").(string), nil
	}

	record, err := app.Dao().FindFirstRecordByData("strings", "name", "welcome")

	if err != nil {
		log.Println(err)
		return "", err
	}

	result := record.Get("content")

	app.Cache().Set("strings:welcome", result.(string))

	return result.(string), nil

}

func getArtistCount(app *pocketbase.PocketBase) (string, error) {

	key := "count:artists"

	found := app.Cache().Has(key)

	if found {
		return app.Cache().Get(key).(string), nil
	}

	c := counter{}

	err := app.Dao().DB().NewQuery("SELECT COUNT(*) as c FROM artists WHERE published IS true").One(&c)

	if err != nil {
		log.Println(err)
		return "0", err
	}

	result := fmt.Sprintf("%d", c.C)

	app.Cache().Set(key, result)

	return result, nil

}

func getArtworkCount(app *pocketbase.PocketBase) (string, error) {

	key := "count:artworks"

	found := app.Cache().Has(key)

	if found {
		return app.Cache().Get(key).(string), nil
	}

	c := counter{}

	err := app.Dao().DB().NewQuery("SELECT COUNT(*) as c FROM artworks WHERE published IS true").One(&c)

	if err != nil {
		log.Println(err)
		return "0", err
	}

	result := fmt.Sprintf("%d", c.C)

	app.Cache().Set(key, result)

	return result, nil

}

func registerHome(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/", func(c echo.Context) error {

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

			isHtmx := utils.IsHtmxRequest(c)

			html := ""

			data := newTemplateData(c)

			data["Content"] = welcomeText
			data["ArtistCount"] = artistCount
			data["ArtworkCount"] = artworkCount

			html, err = assets.Render(assets.Renderable{
				IsHtmx: isHtmx,
				Block:  "home:content",
				Data:   data,
			})

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
