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
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

// renderFeedbackEditor renders the feedback editor block using the assets package.
// It returns the rendered block as a string and an error if there was any.
func renderFeedbackEditor() (string, error) {
	return assets.RenderBlock("feedback:editor", nil)

}

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

			html, err := renderFeedbackEditor()

			if err != nil {
				apis.NewApiError(500, "Failed to render the form", nil)
			}

			return c.HTML(http.StatusOK, html)
		})

		e.Router.POST("feedback", func(c echo.Context) error {

			if !isHtmxRequest(c) {
				return apis.NewBadRequestError("Unexpected request", nil)
			}

			postData := struct {
				Email         string `json:"email" form:"fp_email" query:"email"`
				Message       string `json:"message" form:"message" query:"message"`
				Name          string `json:"name" form:"fp_name" query:"name"`
				HoneyPotName  string `json:"honey_pot_name" form:"name" query:"honey_pot_name"`
				HoneyPotEmail string `json:"honey_pot_email" form:"email" query:"honey_pot_email"`
				ReferTo       string `json:"refer_to"`
			}{
				ReferTo: c.Request().Header.Get("Referer"),
			}

			if err := c.Bind(&postData); err != nil {
				sendToastMessage("Failed to parse form", "is-danger", true, c)
				return apis.NewBadRequestError("Failed to parse form data", err)
			}

			if postData.HoneyPotEmail != "" || postData.HoneyPotName != "" {
				// this is probably a bot
				//TODO: use the new generic logger in pb to log this event
				sendToastMessage("Failed to parse form", "is-danger", true, c)
				return nil
			}

			collection, err := app.Dao().FindCollectionByNameOrId("feedbacks")
			if err != nil {
				sendToastMessage("Database table not found", "is-danger", true, c)
				return apis.NewNotFoundError("Database table not found", err)
			}

			record := models.NewRecord(collection)

			form := forms.NewRecordUpsert(app, record)

			form.LoadData(map[string]any{
				"email":    postData.Email,
				"name":     postData.Name,
				"message":  postData.Message,
				"refer_to": postData.ReferTo,
			})

			if err := form.Submit(); err != nil {

				html, err := renderFeedbackEditor()

				if err != nil {
					return err
				}

				sendToastMessage("Failed to store the feedback", "is-danger", false, c)

				return c.HTML(http.StatusOK, html)

			}

			sendToastMessage("Thank you! Your feedback is valuable to us!", "is-success", true, c)

			return nil
		})

		return nil
	})
}
