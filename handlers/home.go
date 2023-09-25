package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

type Content struct {
	FieldName string `db:"field_name" json:"field_name"`
	Content   string `db:"content" json:"content"`
}

func RegisterHome(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// this is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)
		registry := template.NewRegistry()

		e.Router.GET("/", func(c echo.Context) error {

			result := Content{}

			err := app.Dao().DB().NewQuery("SELECT field_name, content FROM strings WHERE field_name = {:field}").Bind(dbx.Params{
				"field": "welcome_message",
			}).
				One(&result)

			if err != nil {
				fmt.Println(err)
			}

			fmt.Println(result)

			html, err := registry.LoadFiles(
				"views/layout.html",
				"views/home.html",
			).Render(map[string]any{
				"Welcome": result.Content,
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
