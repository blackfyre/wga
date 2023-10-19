package handlers

import (
	"log"
	"net/http"

	"blackfyre.ninja/wga/assets"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func registerFeedbackHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("feedback", func(c echo.Context) error {
			if !isHtmxRequest(c) {
				return apis.NewBadRequestError("Unexpected request", nil)
			}

			html, err := assets.RenderBlock("feedback:editor", nil)

			if err != nil {
				return err
			}

			return c.HTML(http.StatusOK, html)
		})
		return nil
	})
}
