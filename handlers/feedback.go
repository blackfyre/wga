package handlers

import (
	"context"
	"fmt"

	"github.com/blackfyre/wga/assets/templ/components"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

type feedbackForm struct {
	Email         string `json:"email" form:"fp_email" query:"email"`
	Message       string `json:"message" form:"message" query:"message"`
	Name          string `json:"name" form:"fp_name" query:"name"`
	HoneyPotName  string `json:"honey_pot_name" form:"name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" form:"email" query:"honey_pot_email"`
	ReferTo       string `json:"refer_to"`
}

func validateFeedbackForm(form feedbackForm) error {

	if form.HoneyPotEmail != "" || form.HoneyPotName != "" {
		return fmt.Errorf("failed to parse form")
	}

	if form.Email == "" {
		return fmt.Errorf("email is required")
	}

	if form.Message == "" {
		return fmt.Errorf("message is required")
	}

	return nil
}

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
func registerFeedbackHandlers(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/feedback", func(c echo.Context) error {
			e.Router.Use(utils.IsHtmxRequestMiddleware)
			return presentFeedbackForm(c, app)
		})

		e.Router.POST("/feedback", func(c echo.Context) error {

			e.Router.Use(utils.IsHtmxRequestMiddleware)

			postData := feedbackForm{
				ReferTo: c.Request().Header.Get("Referer"),
			}

			if err := c.Bind(&postData); err != nil {
				app.Logger().Error("Failed to parse form data", "error", err.Error())
				utils.SendToastMessage("Failed to parse form", "error", true, c, "")
				return utils.ServerFaultError(c)
			}

			if err := validateFeedbackForm(postData); err != nil {
				app.Logger().Error("Failed to validate form data", "error", err.Error())
				utils.SendToastMessage(err.Error(), "error", true, c, "")

				if err == fmt.Errorf("failed to parse form") {
					app.Logger().Error("Bot caught in honeypot", "error", err.Error())
				}

				return utils.ServerFaultError(c)
			}

			collection, err := app.Dao().FindCollectionByNameOrId("feedbacks")
			if err != nil {
				app.Logger().Error("Database table not found", "error", err.Error())
				utils.SendToastMessage("Database table not found", "error", true, c, "")
				return utils.ServerFaultError(c)
			}

			record := models.NewRecord(collection)

			form := forms.NewRecordUpsert(app, record)

			err = form.LoadData(map[string]any{
				"email":    postData.Email,
				"name":     postData.Name,
				"message":  postData.Message,
				"refer_to": postData.ReferTo,
			})
			if err != nil {
				app.Logger().Error("Failed to process the feedback", "error", err.Error())
				return err
			}

			if err := form.Submit(); err != nil {

				app.Logger().Error("Failed to store the feedback", "error", err.Error())

				err := components.FeedbackForm().Render(context.Background(), c.Response().Writer)

				if err != nil {
					app.Logger().Error("Failed to render the feedback form after form submission error", "error", err.Error())
					return utils.ServerFaultError(c)
				}

				utils.SendToastMessage("Failed to store the feedback", "error", false, c, "")

				return utils.ServerFaultError(c)
			}

			utils.SendToastMessage("Thank you! Your feedback is valuable to us!", "success", true, c, "")

			return nil
		})

		return nil
	})
}

func presentFeedbackForm(c echo.Context, app *pocketbase.PocketBase) error {
	err := components.FeedbackForm().Render(context.Background(), c.Response().Writer)

	if err != nil {
		app.Logger().Error("Failed to render the feedback form", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return err
}
