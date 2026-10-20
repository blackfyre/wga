// Package itineraries adapts the visitor-itinerary workflow to the HTTP layer.
//
// Handlers parse input, resolve the anonymous session, invoke the owning
// workflow in internal/itineraries, and map the result onto Templ pages or
// HTMX fragments. All state-changing routes are validated POSTs guarded by an
// HMAC-bound synchroniser token, a canonical-origin Host/Origin/Referer check,
// and a required trusted-client identity for state creation and publication.
package itineraries

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// errForbidden rejects a state-changing request that fails CSRF, canonical
// origin, or trusted-identity checks.
var errForbidden = errors.New("forbidden")

// securityContext holds the validated policy resolved once at registration
// time and shared by every route handler.
type securityContext struct {
	policy    SecurityPolicy
	canonical *url.URL
	cookie    CookiePolicy
	limiter   *itineraryworkflow.AdmissionLimiter
}

// RegisterHandlers registers every itinerary route under /itineraries.
//
// It validates the supplied security policy and returns an error when it is
// invalid so the serial integration owner can fail closed at startup rather
// than serve a misconfigured anonymous-write surface.
func RegisterHandlers(app *pocketbase.PocketBase, policy SecurityPolicy) error {
	ctx, err := newSecurityContext(policy)
	if err != nil {
		return err
	}

	itineraryworkflow.RegisterHooks(app)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		group := se.Router.Group("/itineraries")

		// Every itinerary response — public index/slideshow, session-owned
		// builder/receipt, mutations, and errors — is non-cacheable so a cached
		// copy cannot outlive moderation or the one-year expiry.
		group.BindFunc(func(c *core.RequestEvent) error {
			noStore(c)
			ctx.projectSessionForPage(app, c)
			return c.Next()
		})

		group.GET("", func(c *core.RequestEvent) error {
			return ctx.indexPage(app, c)
		})

		group.GET("/new", func(c *core.RequestEvent) error {
			return ctx.builderPage(app, c)
		})

		group.GET("/draft", func(c *core.RequestEvent) error {
			return ctx.builderBlock(app, c)
		})
		group.GET("/draft/from-board", func(c *core.RequestEvent) error {
			return ctx.boardImportPage(app, c)
		})

		group.GET("/session", func(c *core.RequestEvent) error {
			return ctx.sessionTray(app, c)
		})

		group.GET("/published", func(c *core.RequestEvent) error {
			return ctx.publishedConfirmation(app, c)
		})

		group.POST("/draft/meta", func(c *core.RequestEvent) error {
			return ctx.setMeta(app, c)
		})

		group.POST("/draft/add", func(c *core.RequestEvent) error {
			return ctx.addStop(app, c)
		})

		group.POST("/draft/remove", func(c *core.RequestEvent) error {
			return ctx.removeStop(app, c)
		})

		group.POST("/draft/move", func(c *core.RequestEvent) error {
			return ctx.moveStop(app, c)
		})

		group.POST("/draft/narration", func(c *core.RequestEvent) error {
			return ctx.setNarration(app, c)
		})

		group.POST("/draft/clear", func(c *core.RequestEvent) error {
			return ctx.clearDraft(app, c)
		})
		group.POST("/draft/from-board", func(c *core.RequestEvent) error {
			return ctx.replaceDraftFromBoard(app, c)
		})

		group.POST("", func(c *core.RequestEvent) error {
			return ctx.publish(app, c)
		})

		group.GET("/{token}", func(c *core.RequestEvent) error {
			return ctx.viewPage(app, c)
		})

		return se.Next()
	})

	return nil
}

func newSecurityContext(policy SecurityPolicy) (*securityContext, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	canonical, err := policy.canonicalURL()
	if err != nil {
		return nil, err
	}

	cookie, err := policy.activeCookiePolicy()
	if err != nil {
		return nil, err
	}

	return &securityContext{
		policy:    policy,
		canonical: canonical,
		cookie:    cookie,
		limiter:   itineraryworkflow.NewAdmissionLimiter(policy.Admission),
	}, nil
}

// projectSessionForPage carries an existing visitor session into itinerary
// page rendering, so these private pages render the tray, its CLEAR action,
// and add-control state on the server, including without JavaScript. It never
// issues a cookie; mutation handlers resolve their own session.
func (ctx *securityContext) projectSessionForPage(app core.App, c *core.RequestEvent) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return
	}
	// The session fragment builds its own projection.
	if c.Request.URL.Path == sessionTrayPath {
		return
	}
	token := itineraryworkflow.TokenFromRequest(c.Request, ctx.cookie.Name)
	if token == "" {
		return
	}
	session := projectFromToken(app, token)
	c.Request = c.Request.WithContext(tmplUtils.WithItineraryProjection(c.Request.Context(), session.CSRF, session.Tray, session.AddedIDs()))
}

// noStore marks a response non-cacheable. It is applied as group middleware to
// every route under /itineraries.
func noStore(c *core.RequestEvent) {
	c.Response.Header().Set("Cache-Control", "private, no-store")
}

// resolveToken returns the session bearer token, issuing a new one when the
// request carries no single valid token. It is the only point that creates the
// anonymous owner identity.
func (ctx *securityContext) resolveToken(c *core.RequestEvent) (string, error) {
	if token := itineraryworkflow.TokenFromRequest(c.Request, ctx.cookie.Name); token != "" {
		return token, nil
	}

	token, err := itineraryworkflow.NewToken()
	if err != nil {
		return "", err
	}

	c.SetCookie(itineraryworkflow.SessionCookie(token, ctx.cookie.Name, ctx.cookie.Secure))
	return token, nil
}

// owner resolves the request's owner digest, issuing a cookie first. It never
// creates durable state: draft allocation happens only in guarded POSTs.
func (ctx *securityContext) owner(c *core.RequestEvent) (string, string, error) {
	token, err := ctx.resolveToken(c)
	if err != nil {
		return "", "", err
	}

	return itineraryworkflow.OwnerDigest(token), token, nil
}

// guardMutation validates a state-changing request: a present, HMAC-bound
// synchroniser token and a canonical-origin Host/Origin/Referer check. It also
// resolves the trusted client identity, returning an empty string when the
// resolver reports it absent or invalid. State-creating and publishing handlers
// fail closed on an empty identity.
func (ctx *securityContext) guardMutation(app core.App, c *core.RequestEvent) (string, string, string, error) {
	token, err := ctx.resolveToken(c)
	if err != nil {
		return "", "", "", err
	}

	if !itineraryworkflow.ValidCSRF(token, c.Request.FormValue("_csrf")) {
		logging.RequestLogger(app, c).Warn("Itinerary mutation rejected",
			"event", "itineraries.mutation.rejected",
			"outcome", "invalid_csrf",
		)
		return "", "", "", errForbidden
	}

	if !itineraryworkflow.SameOrigin(c.Request, ctx.canonical) {
		logging.RequestLogger(app, c).Warn("Itinerary mutation rejected",
			"event", "itineraries.mutation.rejected",
			"outcome", "cross_origin",
		)
		return "", "", "", errForbidden
	}

	clientID, ok := ctx.policy.TrustedClientID(c.Request)
	if !ok {
		clientID = ""
	}

	return itineraryworkflow.OwnerDigest(token), token, clientID, nil
}

// errConfirmAdd marks a session-holding add without a synchroniser token, which
// a no-JavaScript visitor sends from a shared-cached page. It is answered with a
// token-bearing confirmation form rather than a rejection.
var errConfirmAdd = errors.New("itinerary add needs confirmation")

// errRefreshAdd marks the HTMX form of the same case, typically an add made
// before the session tray loaded. It is answered with the session tray, which
// carries the token, so the next attempt succeeds.
var errRefreshAdd = errors.New("itinerary add needs the session tray")

// addGuard is the validated identity of an add request. newSession reports a
// first addition whose session cookie is issued only after the addition
// succeeds, so rejected attempts do not create sessions.
type addGuard struct {
	owner      string
	token      string
	clientID   string
	newSession bool
}

// guardAdd validates an add request. A visitor without a session has no session
// secret that a forged request could exploit, so a first addition is accepted on
// an asserted same-origin request alone. A visitor with a session keeps the
// full guardMutation contract.
//
// Two concurrent first additions from one browser each create a session; the
// later cookie wins and the earlier stop stays in an unreachable draft that the
// purge job removes. The race needs two adds before the first response returns.
func (ctx *securityContext) guardAdd(app core.App, c *core.RequestEvent) (addGuard, error) {
	if existing := itineraryworkflow.TokenFromRequest(c.Request, ctx.cookie.Name); existing != "" {
		owner, token, clientID, err := ctx.guardMutation(app, c)
		if errors.Is(err, errForbidden) &&
			c.Request.FormValue("_csrf") == "" &&
			itineraryworkflow.SameOrigin(c.Request, ctx.canonical) {
			if utils.IsHtmxRequest(c) {
				return addGuard{token: existing}, errRefreshAdd
			}
			return addGuard{token: existing}, errConfirmAdd
		}
		return addGuard{owner: owner, token: token, clientID: clientID}, err
	}

	if !itineraryworkflow.AssertedSameOrigin(c.Request, ctx.canonical) {
		logging.RequestLogger(app, c).Warn("Itinerary mutation rejected",
			"event", "itineraries.mutation.rejected",
			"outcome", "cross_origin",
		)
		return addGuard{}, errForbidden
	}

	token, err := itineraryworkflow.NewToken()
	if err != nil {
		return addGuard{}, err
	}

	clientID, ok := ctx.policy.TrustedClientID(c.Request)
	if !ok {
		clientID = ""
	}

	return addGuard{owner: itineraryworkflow.OwnerDigest(token), token: token, clientID: clientID, newSession: true}, nil
}

// requireClientID fails closed when the trusted client identity is absent or
// invalid, logging only a generic outcome and never the raw identity.
func (ctx *securityContext) requireClientID(app core.App, c *core.RequestEvent, clientID string) bool {
	if clientID != "" {
		return true
	}

	logging.RequestLogger(app, c).Warn("Itinerary mutation rejected",
		"event", "itineraries.mutation.rejected",
		"outcome", "missing_trusted_identity",
	)
	return false
}

// admitDraftIfNew charges the per-identity new-draft budget only when the owner
// has no draft yet, so mutations of an existing draft never consume the
// creation budget. It reports whether the request may proceed and the charged
// reservation, which is the zero value when nothing was charged; callers
// release it when the guarded workflow subsequently fails, since a rolled-back
// transaction leaves no draft.
func (ctx *securityContext) admitDraftIfNew(app core.App, owner string, clientID string) (allowed bool, reservation itineraryworkflow.AdmissionReservation) {
	_, err := itineraryworkflow.FindDraft(app, owner)
	if err == nil {
		return true, itineraryworkflow.AdmissionReservation{}
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// A read failure is surfaced by the workflow rather than here.
		return true, itineraryworkflow.AdmissionReservation{}
	}

	reservation, ok := ctx.limiter.Reserve(clientID, itineraryworkflow.AdmissionDraft)
	if !ok {
		return false, itineraryworkflow.AdmissionReservation{}
	}

	return true, reservation
}

// tooManyDrafts responds with the bounded-draft rate-limit page.
func tooManyDrafts(c *core.RequestEvent) error {
	utils.SendToastMessage("You have created too many itineraries recently. Try again later.", "error", true, c, "")
	return c.HTML(http.StatusTooManyRequests, "")
}

// tooManyPublishes responds with the bounded-publication rate-limit page.
func tooManyPublishes(c *core.RequestEvent) error {
	utils.SendToastMessage("You have published too many itineraries recently. Try again later.", "error", true, c, "")
	return c.HTML(http.StatusTooManyRequests, "")
}

// forbidden responds with the shared bad-request page for rejected mutations.
func forbidden(c *core.RequestEvent) error {
	return utils.BadRequestError(c)
}
