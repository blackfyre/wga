package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
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

func registerHome(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		e.Router.GET("/", func(c echo.Context) error {

			result := Content{}
			artistCount := counter{}
			artworkCount := counter{}

			err := app.Dao().DB().NewQuery("SELECT name, content FROM strings WHERE name = {:field}").Bind(dbx.Params{
				"field": "welcome",
			}).
				One(&result)

			if err != nil {
				fmt.Println(err)
			}

			err = app.Dao().DB().NewQuery("SELECT COUNT(*) as c FROM artists WHERE published IS true").One(&artistCount)

			if err != nil {
				fmt.Println(err)
			}

			err = app.Dao().DB().NewQuery("SELECT COUNT(*) as c FROM artworks WHERE published IS true").One(&artworkCount)

			if err != nil {
				fmt.Println(err)
			}

			html, err := renderPage("home", map[string]any{
				"Content":      result.Content,
				"ArtistCount":  artistCount.C,
				"ArtworkCount": artworkCount.C,
			})

			if err != nil {
				// or redirect to a dedicated 404 HTML page
				return apis.NewNotFoundError("", err)
			}

			return c.HTML(http.StatusOK, html)
		})

		return nil
	})
}
