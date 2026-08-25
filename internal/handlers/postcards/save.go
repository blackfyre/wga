package postcards

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/antiabuse"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/validation"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type postcardSubmission struct {
	SenderName     string   `json:"sender_name" form:"sender_name"`
	SenderEmail    string   `json:"sender_email" form:"sender_email"`
	Recipient      string   `json:"recipient" form:"recipient"`
	Recipients     []string `json:"recipients" form:"recipients[]"`
	Message        string   `json:"message" form:"message"`
	ImageID        string   `json:"image_id" form:"image_id"`
	IncludeMusic   bool     `json:"include_music" form:"include_music"`
	RecaptchaToken string   `json:"recaptcha_token" form:"g-recaptcha-response"`
	HoneyPotName   string   `json:"honey_pot_name" form:"name"`
	HoneyPotEmail  string   `json:"honey_pot_email" form:"email"`
}

func savePostcard(app core.App, c *core.RequestEvent, policy *bluemonday.Policy, captcha config.Captcha, keyring config.PostcardTokenKeyring, verifier antiabuse.Verifier, limiter *submissionLimiter, resolver requesttrust.Resolver) error {
	logger := logging.RequestLogger(app, c)

	clientID, ok := "", false
	if resolver != nil {
		clientID, ok = resolver(c.Request)
	}
	if !ok || clientID == "" {
		logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "missing_trusted_identity")
		return utils.BadRequestError(c)
	}

	var input postcardSubmission
	if err := c.BindBody(&input); err != nil {
		logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "invalid_payload")
		return utils.BadRequestError(c)
	}
	values := pages.PostcardComposeView{SenderName: input.SenderName, SenderEmail: input.SenderEmail, Recipient: input.Recipient, Message: input.Message, IncludeMusic: input.IncludeMusic}
	if input.Recipient == "" && len(input.Recipients) > 0 {
		input.Recipient = input.Recipients[0]
		values.Recipient = input.Recipient
	}

	if err := validation.ValidateHoneypot(input.HoneyPotName, input.HoneyPotEmail); err != nil {
		if errors.Is(err, errs.ErrHoneypotTriggered) {
			logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "honeypot")
		}
		return utils.BadRequestError(c)
	}
	plainMessage := strings.TrimSpace(bluemonday.StrictPolicy().Sanitize(input.Message))
	if strings.TrimSpace(input.SenderName) == "" || strings.TrimSpace(input.SenderEmail) == "" || strings.TrimSpace(input.Recipient) == "" || plainMessage == "" || utf8.RuneCountInString(plainMessage) > pages.PostcardMessageLimit {
		logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "validation")
		return renderForm(input.ImageID, values, "Check the required fields and keep the message within 300 characters.", http.StatusUnprocessableEntity, app, c, captcha)
	}
	if limiter == nil || !limiter.allow(clientID, time.Now()) {
		logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "rate_limited")
		return c.String(http.StatusTooManyRequests, "Too many postcard submissions. Please try again later.")
	}
	if err := validation.ValidateRecaptchaToken(input.RecaptchaToken); err != nil {
		logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "invalid_captcha_token")
		return utils.BadRequestError(c)
	}
	if captcha.Verify() {
		verified, err := verifier.Verify(tmplUtils.ContextFromRequest(c.Request), input.RecaptchaToken, clientID)
		if err != nil {
			logger.Error("Postcard captcha verification failed", "event", "postcard.captcha.failed", "outcome", "provider_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
			return utils.ServerFaultError(c)
		}
		if !verified {
			logger.Warn("Postcard submission rejected", "event", "postcard.submission.rejected", "outcome", "captcha_rejected")
			return utils.BadRequestError(c)
		}
	}

	recipients := input.Recipients
	if input.Recipient != "" {
		recipients = []string{input.Recipient}
	}
	result, err := postcardworkflow.QueueWithAccess(app, keyring, postcardworkflow.QueueInput{
		SenderName: strings.TrimSpace(input.SenderName), SenderEmail: strings.TrimSpace(input.SenderEmail), Recipients: recipients,
		Message: policy.Sanitize(input.Message), ImageID: input.ImageID, IncludeMusic: input.IncludeMusic,
		CorrelationID: logging.RequestID(c),
	}, types.NowDateTime())
	if err != nil {
		outcome := "persistence_error"
		status := http.StatusInternalServerError
		message := "The postcard could not be queued. Please try again."
		if errors.Is(err, postcardworkflow.ErrInvalidPostcard) || errors.Is(err, postcardworkflow.ErrArtworkUnavailable) || errors.Is(err, postcardworkflow.ErrNoRecipients) {
			outcome = "validation"
			status = http.StatusUnprocessableEntity
			message = "Check the postcard details and selected artwork."
		}
		logger.Warn("Postcard submission failed", "event", "postcard.submission.failed", "outcome", outcome, "error_type", logging.ErrorType(err), "error", logging.Redact(err))
		return renderForm(input.ImageID, values, message, status, app, c, captcha)
	}
	access := result.Access[0]
	confirmation := pages.PostcardConfirmationView{
		MaskedRecipient: maskEmail(access.Recipient),
		ViewURL:         "/postcard?token=" + access.Token,
		Expires:         access.ExpiresAt.Time().UTC().Format("2 January 2006"),
	}
	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Postcard queued")
	var buf bytes.Buffer
	if c.Request.Header.Get("HX-Request") == "true" {
		err = pages.PostcardConfirmationDialog(confirmation).Render(ctx, &buf)
	} else {
		err = pages.PostcardConfirmationPage(confirmation).Render(ctx, &buf)
	}
	if err != nil {
		return utils.ServerFaultError(c)
	}
	logger.Info("Postcard submission queued", "event", "postcard.submission.queued", "outcome", "queued", "correlation_id", result.Postcard.GetString("correlation_id"))
	return c.HTML(http.StatusAccepted, buf.String())
}

func maskEmail(address string) string {
	at := strings.LastIndex(address, "@")
	if at <= 0 {
		return "hidden recipient"
	}
	return string([]rune(address[:at])[0]) + "••••@" + address[at+1:]
}
