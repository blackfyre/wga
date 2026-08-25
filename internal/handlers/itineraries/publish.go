package itineraries

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/validation"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// publish validates the draft, transitions it straight to approved with an
// immutable token, and redirects to the session-owned confirmation page. The
// token is readable immediately; the listed choice governs index discovery. A
// publication admission slot is reserved before the workflow runs and released
// on any failure, so validation and persistence failures never permanently
// spend a per-identity publication budget slot.
func (ctx *securityContext) publish(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, _, clientID, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}
	if !ctx.requireClientID(app, c, clientID) {
		return forbidden(c)
	}

	// Honeypot abuse control, matching the established public-form pattern.
	if err := validation.ValidateHoneypot(c.Request.FormValue("name"), c.Request.FormValue("email")); err != nil {
		logging.RequestLogger(app, c).Warn("Itinerary publish rejected",
			"event", "itineraries.publish.rejected",
			"outcome", "honeypot",
		)
		return c.NoContent(http.StatusNoContent)
	}

	if !ctx.limiter.Admit(clientID, itineraryworkflow.AdmissionPublish) {
		return tooManyPublishes(c)
	}

	// The publish form carries the listing choice as a radio ("1" listed, "0"
	// link only). It is recorded before publication; an absent value (tests and
	// non-radio clients) leaves the draft's existing choice untouched.
	if raw := c.Request.FormValue("listed"); raw == "1" || raw == "0" {
		if err := itineraryworkflow.SetListed(app, owner, raw == "1"); err != nil && !errors.Is(err, sql.ErrNoRows) {
			ctx.limiter.Release(clientID, itineraryworkflow.AdmissionPublish)
			return utils.ServerFaultError(c)
		}
	}

	if _, err := itineraryworkflow.Publish(app, owner); err != nil {
		ctx.limiter.Release(clientID, itineraryworkflow.AdmissionPublish)
		switch {
		case errors.Is(err, itineraryworkflow.ErrPublishRateLimit):
			return tooManyPublishes(c)
		case errors.Is(err, itineraryworkflow.ErrTitleRequired),
			errors.Is(err, itineraryworkflow.ErrNoStops),
			errors.Is(err, itineraryworkflow.ErrNotDraft),
			errors.Is(err, sql.ErrNoRows):
			utils.SendToastMessage("Your itinerary needs a title and at least one stop.", "error", true, c, "")
			return utils.BadRequestError(c)
		default:
			logging.RequestLogger(app, c).Error("Itinerary publish failed",
				"event", "itineraries.publish.failed",
				"outcome", "persistence_error",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
			return utils.ServerFaultError(c)
		}
	}

	logging.RequestLogger(app, c).Info("Itinerary published",
		"event", "itineraries.publish.published",
		"outcome", "published",
	)

	// HTMX publishes must become a real page navigation: the confirmation is a
	// session-owned full page whose URL is the only record of the share link,
	// so swapping it into the current body would leave the browser on
	// /itineraries/new with no way back. The HX-Redirect header makes htmx
	// perform the navigation; the ordinary 303 preserves the no-JavaScript flow.
	if utils.IsHtmxRequest(c) {
		c.Response.Header().Set("HX-Redirect", "/itineraries/published")
		return c.NoContent(http.StatusNoContent)
	}

	return c.Redirect(http.StatusSeeOther, "/itineraries/published")
}

// publishedConfirmation renders the session-owned publication receipt.
func (ctx *securityContext) publishedConfirmation(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, _, err := ctx.owner(c)
	if err != nil {
		return utils.ServerFaultError(c)
	}

	record, err := itineraryworkflow.LatestPublished(app, owner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Redirect(http.StatusSeeOther, "/itineraries/new")
		}
		return utils.ServerFaultError(c)
	}

	view := loadConfirmationView(app, record)

	ctxb := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Itinerary published")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.DescriptionKey, "Your itinerary is published and shareable for one year.")

	var buf bytes.Buffer
	if err := pages.ItineraryPublishedPage(view).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buf.String())
}
