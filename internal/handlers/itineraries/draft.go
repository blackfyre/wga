package itineraries

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// builderStateFromQuery reads the volatile builder UI state from GET query
// parameters (pq for the picker search, picker=1 for the disclosure, stop=N for
// the selected stop). It is used by the full page and block renders.
func builderStateFromQuery(c *core.RequestEvent) builderState {
	return builderState{
		Query:      strings.TrimSpace(c.Request.URL.Query().Get("pq")),
		PickerOpen: c.Request.URL.Query().Get("picker") == "1",
		Selected:   parseSelected(c.Request.URL.Query().Get("stop")),
	}
}

// builderStateFromForm reads the same volatile state from a mutation's hidden
// form fields, so an add/reorder/narration response keeps the picker query,
// disclosure, and selected stop the visitor was looking at.
func builderStateFromForm(c *core.RequestEvent) builderState {
	return builderState{
		Query:      strings.TrimSpace(c.Request.FormValue("pq")),
		PickerOpen: c.Request.FormValue("picker") == "1",
		Selected:   parseSelected(c.Request.FormValue("stop")),
	}
}

// parseSelected parses a stop index, defaulting to zero for an absent or
// malformed value so the caller's clamp keeps the selection valid.
func parseSelected(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}

	return value
}

// builderURL encodes the volatile builder state back onto the ordinary (no-JS)
// builder URL so redirects after a mutation preserve the visitor's place.
func builderURL(state builderState) string {
	query := url.Values{}
	if state.PickerOpen {
		query.Set("picker", "1")
	}
	if state.Query != "" {
		query.Set("pq", state.Query)
	}
	if state.Selected > 0 {
		query.Set("stop", strconv.Itoa(state.Selected))
	}
	if len(query) == 0 {
		return "/itineraries/new"
	}

	return "/itineraries/new?" + query.Encode()
}

// builderPage renders the full server-rendered builder page. A visitor without
// a draft sees an empty builder projection; no draft record is created here.
func (ctx *securityContext) builderPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, err := ctx.owner(c)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	view, err := loadBuilderView(app, owner, itineraryworkflow.CSRFToken(token), builderStateFromQuery(c))
	if err != nil {
		logging.RequestLogger(app, c).Error("Itinerary builder load failed", "event", "itineraries.builder.failed", "outcome", "load_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	ctxb := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Itinerary")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.DescriptionKey, "Arrange and narrate your personal itinerary through the collection.")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.CanonicalUrlKey, utils.AssetUrl("/itineraries/new"))

	var buf bytes.Buffer
	if err := pages.ItineraryBuilderPage(view).Render(ctxb, &buf); err != nil {
		logging.RequestLogger(app, c).Error("Itinerary builder render failed", "event", "itineraries.builder.failed", "outcome", "render_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// builderBlock renders only the builder block, used for HTMX refreshes.
func (ctx *securityContext) builderBlock(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, err := ctx.owner(c)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	view, err := loadBuilderView(app, owner, itineraryworkflow.CSRFToken(token), builderStateFromQuery(c))
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	var buf bytes.Buffer
	if err := pages.ItineraryBuilder(view, false).Render(tmplUtils.ContextFromRequest(c.Request), &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// renderBuilder renders the builder block and an out-of-band tray update for an
// HTMX mutation response.
func renderBuilder(app core.App, c *core.RequestEvent, owner string, token string, state builderState) error {
	view, err := loadBuilderView(app, owner, itineraryworkflow.CSRFToken(token), state)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	tray, err := loadTrayView(app, owner)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	var buf bytes.Buffer
	if err := pages.ItineraryBuilder(view, false).Render(tmplUtils.ContextFromRequest(c.Request), &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	if err := components.ItineraryTray(tray, true).Render(tmplUtils.ContextFromRequest(c.Request), &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// respondBuilder renders the builder block for HTMX or redirects to the full
// page without JavaScript, preserving the volatile builder state in both paths.
func respondBuilder(app core.App, c *core.RequestEvent, owner string, token string, state builderState) error {
	if utils.IsHtmxRequest(c) {
		return renderBuilder(app, c, owner, token, state)
	}

	return c.Redirect(http.StatusSeeOther, builderURL(state))
}

// renderTrayWithBuilder renders the tray as the primary HTMX target plus an
// out-of-band builder refresh, used by the add route so both surfaces update.
// The tray's CLEAR action reads its synchroniser token from the read-only
// itinerary projection carried in the render context, exactly as the central
// middleware supplies it for full-page renders. A POST add does not pass through
// that middleware, so the context is decorated here to keep the swapped-in
// tray's clear action working under the same session CSRF contract. The OOB
// builder embeds its own token through the builder view.
func renderTrayWithBuilder(app core.App, c *core.RequestEvent, owner string, token string, state builderState) error {
	tray, err := loadTrayView(app, owner)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	view, err := loadBuilderView(app, owner, itineraryworkflow.CSRFToken(token), state)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	ctxb := tmplUtils.WithItineraryProjection(
		tmplUtils.ContextFromRequest(c.Request),
		itineraryworkflow.CSRFToken(token),
		tray,
		nil,
	)

	var buf bytes.Buffer
	if err := components.ItineraryTray(tray, false).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	if err := pages.ItineraryBuilder(view, true).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

func (ctx *securityContext) setMeta(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, clientID, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}
	if !ctx.requireClientID(app, c, clientID) {
		return forbidden(c)
	}

	state := builderStateFromForm(c)
	meta := itineraryworkflow.Meta{
		Title:   c.Request.FormValue("title"),
		Intro:   c.Request.FormValue("intro"),
		Creator: c.Request.FormValue("creator"),
	}

	allowed, charged := ctx.admitDraftIfNew(app, owner, clientID)
	if !allowed {
		return tooManyDrafts(c)
	}
	if err := itineraryworkflow.SetMeta(app, owner, meta); err != nil {
		if charged {
			ctx.limiter.Release(clientID, itineraryworkflow.AdmissionDraft)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return respondBuilder(app, c, owner, token, state)
}

func (ctx *securityContext) addStop(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, clientID, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}
	if !ctx.requireClientID(app, c, clientID) {
		return forbidden(c)
	}

	state := builderStateFromForm(c)
	artworkID := strings.TrimSpace(c.Request.FormValue("artwork_id"))
	if artworkID == "" {
		return utils.BadRequestError(c)
	}

	allowed, charged := ctx.admitDraftIfNew(app, owner, clientID)
	if !allowed {
		return tooManyDrafts(c)
	}
	if _, err := itineraryworkflow.AddStop(app, owner, artworkID); err != nil {
		if charged {
			ctx.limiter.Release(clientID, itineraryworkflow.AdmissionDraft)
		}
		if errors.Is(err, itineraryworkflow.ErrStopLimit) {
			utils.SendToastMessage("Itineraries are limited to 15 stops.", "error", true, c, "")
			return utils.BadRequestError(c)
		}
		if errors.Is(err, itineraryworkflow.ErrArtworkUnavailable) {
			utils.SendToastMessage("That artwork is not available.", "error", true, c, "")
			return utils.NotFoundError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	utils.SendToastMessage("Added to your itinerary.", "success", true, c, "")

	// HTMX adds refresh the tray (primary target) and the builder block (OOB).
	if utils.IsHtmxRequest(c) {
		return renderTrayWithBuilder(app, c, owner, token, state)
	}

	return c.Redirect(http.StatusSeeOther, builderURL(state))
}

func (ctx *securityContext) removeStop(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, _, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}

	state := builderStateFromForm(c)
	stopID := strings.TrimSpace(c.Request.FormValue("stop_id"))
	if err := itineraryworkflow.RemoveStop(app, owner, stopID); err != nil {
		if errors.Is(err, itineraryworkflow.ErrStopNotFound) || errors.Is(err, sql.ErrNoRows) {
			return utils.BadRequestError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return respondBuilder(app, c, owner, token, state)
}

func (ctx *securityContext) moveStop(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, _, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}

	state := builderStateFromForm(c)
	stopID := strings.TrimSpace(c.Request.FormValue("stop_id"))
	dir := strings.TrimSpace(c.Request.FormValue("dir"))
	if err := itineraryworkflow.MoveStop(app, owner, stopID, dir); err != nil {
		if errors.Is(err, itineraryworkflow.ErrInvalidMove) || errors.Is(err, itineraryworkflow.ErrStopNotFound) || errors.Is(err, sql.ErrNoRows) {
			return utils.BadRequestError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return respondBuilder(app, c, owner, token, state)
}

func (ctx *securityContext) setNarration(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, _, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}

	state := builderStateFromForm(c)
	stopID := strings.TrimSpace(c.Request.FormValue("stop_id"))
	narration := c.Request.FormValue("narration")
	if err := itineraryworkflow.SetNarration(app, owner, stopID, narration); err != nil {
		if errors.Is(err, itineraryworkflow.ErrStopNotFound) || errors.Is(err, sql.ErrNoRows) {
			return utils.BadRequestError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return respondBuilder(app, c, owner, token, state)
}

func (ctx *securityContext) clearDraft(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, _, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}

	state := builderStateFromForm(c)
	if err := itineraryworkflow.ClearDraft(app, owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.BadRequestError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	// The tray's CLEAR action targets #itinerary-tray, which mounts on every
	// page, so it needs the tray-primary response shape. The builder's CLEAR
	// ALL still targets #itinerary-builder and keeps the builder-primary shape.
	if utils.IsHtmxRequest(c) && c.Request.Header.Get("HX-Target") == "itinerary-tray" {
		return renderTrayWithBuilder(app, c, owner, token, state)
	}

	return respondBuilder(app, c, owner, token, state)
}
