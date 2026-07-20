package postcards

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/validation"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func savePostcard(app *pocketbase.PocketBase, c *core.RequestEvent, p *bluemonday.Policy, captcha config.Captcha) error {
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

	if err := c.BindBody(&postData); err != nil {
		app.Logger().Error("Failed to parse form", "error", err.Error())
		utils.SendToastMessage("Failed to parse form", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	if err := validation.ValidateHoneypot(postData.HoneyPotName, postData.HoneyPotEmail); err != nil {
		if errors.Is(err, errs.ErrHoneypotTriggered) {
			app.Logger().Warn("Honey pot triggered", "data", fmt.Sprintf("%+v", postData), "ip", c.RealIP())
			return utils.ServerFaultError(c)
		}

		return utils.ServerFaultError(c)
	}

	if err := validation.ValidateRecaptchaToken(postData.RecaptchaToken); err != nil {
		utils.SendToastMessage("Captcha verification failed", "error", true, c, "")
		return utils.BadRequestError(c)
	}

	if captcha.Verify() {
		verified, err := verifyRecaptchaToken(c.Request.Context(), http.DefaultClient, captcha.Secret(), postData.RecaptchaToken, c.RealIP())
		if err != nil {
			app.Logger().Error("Failed to verify recaptcha token", "error", err.Error())
			utils.SendToastMessage("Failed to verify captcha", "error", true, c, "")
			return utils.ServerFaultError(c)
		}

		if !verified {
			app.Logger().Warn("Recaptcha validation failed", "ip", c.RealIP())
			utils.SendToastMessage("Captcha verification failed", "error", true, c, "")
			return utils.BadRequestError(c)
		}
	} else {
		app.Logger().Warn("Captcha verification is disabled for the local or test environment")
	}

	collection, err := app.FindCollectionByNameOrId(constants.CollectionPostcards)
	if err != nil {
		app.Logger().Error("Failed to find postcard collection", "error", err.Error())
		return utils.NotFoundError(c)
	}

	record := core.NewRecord(collection)

	record.Set("status", "queued")
	record.Set("sender_name", postData.SenderName)
	record.Set("sender_email", postData.SenderEmail)
	record.Set("recipients", strings.Join(postData.Recipients, ","))
	record.Set("message", p.Sanitize(postData.Message))
	record.Set("image_id", postData.ImageId)
	record.Set("notify_sender", postData.NotificationRequired)

	if err := app.Save(record); err != nil {

		return renderForm(postData.ImageId, app, c)
	}

	utils.SendToastMessage("Thank you! Your postcard has been queued for sending!", "success", true, c, "")

	return nil
}
