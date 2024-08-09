package postcards

import (
	"fmt"
	"strings"

	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

func savePostcard(app *pocketbase.PocketBase, c echo.Context, p *bluemonday.Policy) error {
	postData := struct {
		SenderName           string   `json:"sender_name" form:"sender_name" query:"sender_name" validate:"required"`
		SenderEmail          string   `json:"sender_email" form:"sender_email" query:"sender_email" validate:"required,email"`
		Recipients           []string `json:"recipients" form:"recipients[]" query:"recipients" validate:"required"`
		Message              string   `json:"message" form:"message" query:"message" validate:"required"`
		ImageId              string   `json:"image_id" form:"image_id" query:"image_id" validate:"required"`
		NotificationRequired bool     `json:"notification_required" form:"notify_sender" query:"notification_required"`
		RecaptchaToken       string   `json:"recaptcha_token" form:"g-recaptcha-response" query:"recaptcha_token" validate:"required"`
		HoneyPotName         string   `json:"honey_pot_name" form:"name" query:"honey_pot_name"`
		HoneyPotEmail        string   `json:"honey_pot_email" form:"email" query:"honey_pot_email"`
	}{}

	if err := c.Bind(&postData); err != nil {
		app.Logger().Error("Failed to parse form", "error", err.Error())
		utils.SendToastMessage("Failed to parse form", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	if postData.HoneyPotEmail != "" || postData.HoneyPotName != "" {
		// this is probably a bot
		app.Logger().Warn("Honey pot triggered", "data", fmt.Sprintf("%+v", postData), "ip", c.RealIP())
		return utils.ServerFaultError(c)
	}

	collection, err := app.Dao().FindCollectionByNameOrId("postcards")
	if err != nil {
		app.Logger().Error("Failed to find postcard collection", "error", err.Error())
		return utils.NotFoundError(c)
	}

	record := models.NewRecord(collection)

	form := forms.NewRecordUpsert(app, record)

	err = form.LoadData(map[string]any{
		"status":        "queued",
		"sender_name":   postData.SenderName,
		"sender_email":  postData.SenderEmail,
		"recipients":    strings.Join(postData.Recipients, ","),
		"message":       p.Sanitize(postData.Message),
		"image_id":      postData.ImageId,
		"notify_sender": postData.NotificationRequired,
	})

	if err != nil {
		app.Logger().Error("Failed to process postcard form", "error", err.Error())
		utils.SendToastMessage("Failed to process postcard form", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	if err := form.Submit(); err != nil {

		return renderForm(postData.ImageId, app, c)
	}

	utils.SendToastMessage("Thank you! Your postcard has been queued for sending!", "success", true, c, "")

	return nil
}
