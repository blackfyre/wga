package handlers

import (
	"context"
	"fmt"

	"github.com/blackfyre/wga/assets/templ/components"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

// registerFeedbackHandlers registers the feedback handlers for the application.
// It adds the GET and POST routes for the feedback form, handles form submission,
// and stores the feedback in the database.
//
// Parameters:
// - app: The PocketBase application instance.
// - p: The bluemonday policy for sanitizing HTML input.
//
// Returns:
// - An error if there was a problem registering the handlers, or nil otherwise.
func registerFeedbackHandlers(app *pocketbase.PocketBase, p *bluemonday.Policy) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("feedback", func(c echo.Context) error {
			if !utils.IsHtmxRequest(c) {
				app.Logger().Error("Unexpected request to feedback form")
				return utils.ServerFaultError(c)
			}

			err := components.FeedbackForm().Render(context.Background(), c.Response().Writer)

			if err != nil {
				app.Logger().Error("Failed to render the feedback form", err)
				return utils.ServerFaultError(c)
			}

			return err
		})

		e.Router.POST("feedback", func(c echo.Context) error {

			if !utils.IsHtmxRequest(c) {
				return utils.ServerFaultError(c)
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
				app.Logger().Error("Failed to parse form data", err)
				sendToastMessage("Failed to parse form", "is-danger", true, c)
				return utils.ServerFaultError(c)
			}

			if postData.HoneyPotEmail != "" || postData.HoneyPotName != "" {
				// this is probably a bot
				app.Logger().Warn("Honey pot triggered", "data", fmt.Sprintf("+%v", postData))
				utils.SendToastMessage("Failed to parse form", "is-danger", true, c, "")
				return utils.ServerFaultError(c)
			}

			collection, err := app.Dao().FindCollectionByNameOrId("feedbacks")
			if err != nil {
				app.Logger().Error("Database table not found", err)
				utils.SendToastMessage("Database table not found", "is-danger", true, c, "")
				return utils.ServerFaultError(c)
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

				app.Logger().Error("Failed to store the feedback", err)

				err := components.FeedbackForm().Render(context.Background(), c.Response().Writer)

				if err != nil {
					app.Logger().Error("Failed to render the feedback form after form submission error", err)
					return utils.ServerFaultError(c)
				}

				utils.SendToastMessage("Failed to store the feedback", "is-danger", false, c, "")

				return utils.ServerFaultError(c)
			}

			utils.SendToastMessage("Thank you! Your feedback is valuable to us!", "is-success", true, c, "")

			return nil
		})

		return nil
	})
}
