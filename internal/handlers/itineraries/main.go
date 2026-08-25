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
		limiter:   itineraryworkflow.NewAdmissionLimiter(),
	}, nil
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
// creation budget. It reports whether the request may proceed and whether a
// budget slot was actually charged; callers release the slot when the guarded
// workflow subsequently fails, since a rolled-back transaction leaves no draft.
func (ctx *securityContext) admitDraftIfNew(app core.App, owner string, clientID string) (allowed bool, charged bool) {
	_, err := itineraryworkflow.FindDraft(app, owner)
	if err == nil {
		return true, false
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// A read failure is surfaced by the workflow rather than here.
		return true, false
	}

	if !ctx.limiter.Admit(clientID, itineraryworkflow.AdmissionDraft) {
		return false, false
	}

	return true, true
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
