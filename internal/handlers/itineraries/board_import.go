package itineraries

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/studyboard"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func (ctx *securityContext) boardImportPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, err := ctx.owner(c)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return ctx.renderBoardImport(app, c, owner, itineraryworkflow.CSRFToken(token), c.Request.URL.Query().Get("board"), false, http.StatusOK)
}

func (ctx *securityContext) replaceDraftFromBoard(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	owner, token, clientID, err := ctx.guardMutation(app, c)
	if err != nil {
		return forbidden(c)
	}
	if !ctx.requireClientID(app, c, clientID) {
		return forbidden(c)
	}

	board, err := studyboard.Resolve(app, c.Request.FormValue("board"))
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	ids := board.IDs()
	allowed, charged := ctx.admitDraftIfNew(app, owner, clientID)
	if !allowed {
		return tooManyDrafts(c)
	}
	err = itineraryworkflow.ReplaceDraft(
		app,
		owner,
		ids,
		strings.TrimSpace(c.Request.FormValue("expectation")),
		c.Request.FormValue("confirm") == "replace",
	)
	if err != nil {
		if charged {
			ctx.limiter.Release(clientID, itineraryworkflow.AdmissionDraft)
		}
		switch {
		case errors.Is(err, itineraryworkflow.ErrReplacementStale):
			return ctx.renderBoardImport(app, c, owner, itineraryworkflow.CSRFToken(token), strings.Join(ids, ","), true, http.StatusConflict)
		case errors.Is(err, itineraryworkflow.ErrReplacementConfirmation):
			return ctx.renderBoardImport(app, c, owner, itineraryworkflow.CSRFToken(token), strings.Join(ids, ","), false, http.StatusBadRequest)
		case errors.Is(err, itineraryworkflow.ErrNoReplacementStops), errors.Is(err, itineraryworkflow.ErrArtworkUnavailable):
			return utils.BadRequestError(c)
		default:
			return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
		}
	}

	return c.Redirect(http.StatusSeeOther, "/itineraries/new")
}

func (ctx *securityContext) renderBoardImport(app core.App, c *core.RequestEvent, owner string, csrf string, rawBoard string, changed bool, status int) error {
	board, err := studyboard.Resolve(app, rawBoard)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	preview, err := itineraryworkflow.PreviewReplacement(app, owner, board.IDs())
	if err != nil {
		if errors.Is(err, itineraryworkflow.ErrNoReplacementStops) || errors.Is(err, itineraryworkflow.ErrArtworkUnavailable) {
			return utils.BadRequestError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	view := dto.ItineraryBoardImportView{
		Board: strings.Join(preview.ArtworkIDs, ","), CSRF: csrf, Expectation: preview.Expectation,
		IncomingCount: preview.IncomingCount, ExistingWorkCount: preview.ExistingWorkCount,
		NarrationCount: preview.NarrationCount, RequiresConfirm: preview.RequiresConfirm,
		ReplacementChanged: changed,
	}
	ctxb := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Study Board itinerary")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.DescriptionKey, "Review replacing the current itinerary draft with this Study Board.")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.CanonicalUrlKey, utils.AssetUrl("/study-board"))
	var buf bytes.Buffer
	if err := pages.ItineraryBoardImportPage(view).Render(ctxb, &buf); err != nil {
		logging.RequestLogger(app, c).Error("Study Board itinerary review render failed", "event", "itineraries.board_import.failed", "outcome", "render_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return c.HTML(status, buf.String())
}
