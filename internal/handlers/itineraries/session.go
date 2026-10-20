package itineraries

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

// sessionTray renders the visitor's tray, synchroniser token, and added state
// for documents that carry no session state of their own. It never issues a
// cookie: a visitor without a session receives an empty tray, so page views
// and automated traffic do not mint sessions.
func (ctx *securityContext) sessionTray(app core.App, c *core.RequestEvent) error {
	session := PublicSession{Tray: dto.ItineraryTrayView{BuilderURL: "/itineraries/new"}}
	if token := itineraryworkflow.TokenFromRequest(c.Request, ctx.cookie.Name); token != "" {
		session = projectFromToken(app, token)
	}

	ctxb := tmplUtils.WithItineraryProjection(tmplUtils.ContextFromRequest(c.Request), session.CSRF, session.Tray, session.AddedIDs())

	var buf bytes.Buffer
	if err := components.ItineraryTray(session.Tray, false).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// confirmAdd answers a no-JavaScript add from a shared-cached page, which
// carries the session cookie but no synchroniser token, with a token-bearing
// form that repeats the addition.
func confirmAdd(c *core.RequestEvent, token string) error {
	artworkID := strings.TrimSpace(c.Request.FormValue("artwork_id"))
	if artworkID == "" {
		return utils.BadRequestError(c)
	}

	ctxb := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Add to itinerary")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.DescriptionKey, "Confirm adding this work to your itinerary.")

	var buf bytes.Buffer
	if err := pages.ItineraryAddConfirmPage(artworkID, itineraryworkflow.CSRFToken(token)).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}
