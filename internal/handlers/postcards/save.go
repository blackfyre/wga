package postcards

import (
	"errors"
	"net/http"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/validation"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
)

func savePostcard(app core.App, c *core.RequestEvent, p *bluemonday.Policy, captcha config.Captcha) error {
	logger := logging.RequestLogger(app, c)
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
		logger.Error("Postcard submission parsing failed",
			"event", "postcard.submission.failed",
			"outcome", "invalid_payload",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		utils.SendToastMessage("Failed to parse form", "error", true, c, "")
		return utils.ServerFaultError(c)
	}

	if err := validation.ValidateHoneypot(postData.HoneyPotName, postData.HoneyPotEmail); err != nil {
		if errors.Is(err, errs.ErrHoneypotTriggered) {
			logger.Warn("Postcard submission rejected",
				"event", "postcard.submission.rejected",
				"outcome", "honeypot",
			)
			return utils.ServerFaultError(c)
		}

		return utils.ServerFaultError(c)
	}

	if err := validation.ValidateRecaptchaToken(postData.RecaptchaToken); err != nil {
		logger.Warn("Postcard submission rejected",
			"event", "postcard.submission.rejected",
			"outcome", "invalid_captcha_token",
		)
		utils.SendToastMessage("Captcha verification failed", "error", true, c, "")
		return utils.BadRequestError(c)
	}

	if captcha.Verify() {
		verified, err := verifyRecaptchaToken(c.Request.Context(), http.DefaultClient, captcha.Secret(), postData.RecaptchaToken, c.RealIP())
		if err != nil {
			logger.Error("Postcard captcha verification failed",
				"event", "postcard.captcha.failed",
				"outcome", "provider_error",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
			utils.SendToastMessage("Failed to verify captcha", "error", true, c, "")
			return utils.ServerFaultError(c)
		}

		if !verified {
			logger.Warn("Postcard submission rejected",
				"event", "postcard.submission.rejected",
				"outcome", "captcha_rejected",
			)
			utils.SendToastMessage("Captcha verification failed", "error", true, c, "")
			return utils.BadRequestError(c)
		}
	} else {
		logger.Warn("Postcard captcha verification skipped",
			"event", "postcard.captcha.skipped",
			"outcome", "disabled",
		)
	}

	_, err := postcardworkflow.Queue(app, postcardworkflow.QueueInput{
		SenderName:   postData.SenderName,
		SenderEmail:  postData.SenderEmail,
		Recipients:   postData.Recipients,
		Message:      p.Sanitize(postData.Message),
		ImageID:      postData.ImageId,
		NotifySender: postData.NotificationRequired,
	})
	if err != nil {
		logger.Error("Postcard submission persistence failed",
			"event", "postcard.submission.failed",
			"outcome", "persistence_error",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return renderForm(postData.ImageId, app, c, captcha)
	}

	logger.Info("Postcard submission queued",
		"event", "postcard.submission.queued",
		"outcome", "queued",
	)

	utils.SendToastMessage("Thank you! Your postcard has been queued for sending!", "success", true, c, "")

	return nil
}
