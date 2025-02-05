package feedback

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/blackfyre/wga/assets/templ/components"
	"github.com/blackfyre/wga/errs"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type feedbackForm struct {
	Email         string `json:"email" form:"fp_email" query:"email"`
	Message       string `json:"message" form:"message" query:"message"`
	Name          string `json:"name" form:"fp_name" query:"name"`
	HoneyPotName  string `json:"honey_pot_name" form:"name" query:"honey_pot_name"`
	HoneyPotEmail string `json:"honey_pot_email" form:"email" query:"honey_pot_email"`
	ReferTo       string `json:"refer_to"`
}

// validateFeedbackForm validates the feedback form.
// It checks if the honey pot fields are empty and returns an error if they are not.
// It also checks if the email and message fields are empty and returns an error if they are.
// If all validations pass, it returns nil.
func validateFeedbackForm(form feedbackForm) error {
	if form.HoneyPotName != "" || form.HoneyPotEmail != "" {
		return errs.ErrHoneypotTriggered
	}

	if form.Message == "" {
		return errs.ErrMessageRequired
	}

	return nil
}

// presentFeedbackForm is a function that presents a feedback form to the user.
// It takes an echo.Context and a *pocketbase.PocketBase as parameters.
// It renders the feedback form using the components.FeedbackForm() function.
// If there is an error during rendering, it logs the error and returns a server fault error.
// Otherwise, it returns nil.
func presentFeedbackForm(c *core.RequestEvent, app *pocketbase.PocketBase) error {

	var buff bytes.Buffer

	err := components.FeedbackForm().Render(context.Background(), &buff)

	if err != nil {
		app.Logger().Error("Failed to render the feedback form", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

// processFeedbackForm processes the feedback form submitted by the user.
// It takes an echo.Context and a *pocketbase.PocketBase as parameters.
// The function binds the form data to the feedbackForm struct and validates it.
// If the form data fails to parse or validate, an error is logged and a server fault error is returned.
// If the form data is valid, it is saved using the saveFeedback function.
// If there is an error while saving the feedback, the feedback form is rendered again and a server fault error is returned.
// If the feedback is successfully saved, a success toast message is sent to the user.
// The function returns nil if there are no errors.
func processFeedbackForm(c *core.RequestEvent, app *pocketbase.PocketBase) error {
	postData := feedbackForm{
		ReferTo: c.Request.Header.Get("Referer"),
	}

	if err := c.BindBody(&postData); err != nil {
		app.Logger().Error("Failed to parse form data", "error", err.Error())
		utils.SendToastMessage("Failed to parse form", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	if err := validateFeedbackForm(postData); err != nil {
		app.Logger().Error("Failed to validate form data", "error", err.Error())
		utils.SendToastMessage(err.Error(), "error", true, c, "")

		if errors.Is(err, errs.ErrHoneypotTriggered) {
			app.Logger().Error("Bot caught in honeypot", "error", err.Error())
		}

		return utils.ServerFaultError(c)
	}

	if err := saveFeedback(app, c, postData); err != nil {

		app.Logger().Error("Failed to store the feedback", "error", err.Error())

		var buff bytes.Buffer

		err := components.FeedbackForm().Render(context.Background(), &buff)

		if err != nil {
			app.Logger().Error("Failed to render the feedback form after form submission error", "error", err.Error())
			return utils.ServerFaultError(c)
		}

		utils.SendToastMessage("Failed to store the feedback", "error", false, c, "")

		return c.HTML(http.StatusOK, buff.String())
	}

	utils.SendToastMessage("Thank you! Your feedback is valuable to us!", "success", true, c, "")

	return nil
}

// saveFeedback saves the feedback provided by the user.
// It takes the PocketBase app instance, the echo.Context c, and the postData feedbackForm as parameters.
// It returns an error if there is any issue with saving the feedback.
//
// The function first attempts to find the "feedbacks" collection in the database using the app.FindCollectionByNameOrId method.
// If the collection is not found, it logs an error and sends a toast message to the user indicating the issue.
// It then returns a server fault error using the utils.ServerFaultError function.
//
// Next, it creates a new record using the models.NewRecord method and the retrieved collection.
//
// It creates a new form using the forms.NewRecordUpsert method and the app instance and the created record.
//
// The function loads the data from the postData into the form using the form.LoadData method.
// If there is an error during the data loading process, it logs an error and returns the error.
//
// Finally, it submits the form using the form.Submit method and returns the result.
func saveFeedback(app *pocketbase.PocketBase, c *core.RequestEvent, postData feedbackForm) error {
	collection, err := app.FindCollectionByNameOrId("feedbacks")
	if err != nil {
		app.Logger().Error("Database table not found", "error", err.Error())
		utils.SendToastMessage("Database table not found", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	r := core.NewRecord(collection)

	r.Set("email", postData.Email)
	r.Set("name", postData.Name)
	r.Set("message", postData.Message)
	r.Set("refer_to", postData.ReferTo)

	err = app.Save(r)
	if err != nil {
		app.Logger().Error("Failed to process the feedback", "error", err.Error())
		return err
	}

	return nil
}

// RegisterHandlers registers the feedback handlers to the provided PocketBase application.
// It adds GET and POST routes for "/feedback" endpoint, which are responsible for presenting and processing feedback forms.
// The handlers use the given echo.Context and PocketBase app to handle the requests.
// The handlers also utilize the IsHtmxRequestMiddleware from the utils package.
// This function should be called before serving the application.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/feedback", func(c *core.RequestEvent) error {
			return presentFeedbackForm(c, app)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		e.Router.POST("/feedback", func(c *core.RequestEvent) error {
			return processFeedbackForm(c, app)
		}).BindFunc(utils.IsHtmxRequestMiddleware)

		return nil
	})
}
