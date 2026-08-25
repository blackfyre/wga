package handlers

import (
	"net/http"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/handlers/itineraries"
	"github.com/pocketbase/pocketbase/core"
)

// registerItinerarySessionMiddleware binds the read-only itinerary session
// projection to the router. It must run before feature routes so the shared
// layout tray and supported add controls can read the projection from the
// request context.
func registerItinerarySessionMiddleware(app core.App, cookie itineraries.CookiePolicy) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return projectItinerarySession(app, e, cookie)
		})

		return se.Next()
	})
}

// projectItinerarySession resolves or issues the visitor's itinerary session
// for an eligible public document request and carries it through the request
// context. A missing, duplicated, or malformed cookie is rotated to a fresh
// token before downstream rendering, without creating records. Responses that
// carry the session-personalised projection are marked private, no-store.
func projectItinerarySession(app core.App, e *core.RequestEvent, cookie itineraries.CookiePolicy) error {
	if !eligibleForItineraryProjection(e.Request) {
		return e.Next()
	}

	session, err := itineraries.ProjectSession(app, e, cookie)
	if err != nil {
		// Startup validation already fails closed on an invalid policy; a
		// per-request token-generation failure must not block rendering.
		return e.Next()
	}

	e.Request = e.Request.WithContext(
		tmplUtils.WithItineraryProjection(e.Request.Context(), session.CSRF, session.Tray, session.AddedIDs()),
	)
	e.Response.Header().Set("Cache-Control", "private, no-store")

	return e.Next()
}

// eligibleForItineraryProjection reports whether a request is a supported
// public HTML rendering request that should carry the itinerary session
// projection. It includes full-page GET/HEAD requests that negotiate HTML and
// HTMX GET fragments (which always render HTML), and excludes technical
// boundaries, POST/PUT/DELETE, and non-HTML technical requests. This keeps
// cookies and read-only projection out of API, admin, asset, sitemap, and
// private preview traffic.
func eligibleForItineraryProjection(request *http.Request) bool {
	if request == nil {
		return false
	}

	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return false
	}

	if isTrustedHeadMarkupBoundary(request.URL.Path) {
		return false
	}

	if request.Header.Get("HX-Request") == "true" {
		return true
	}

	return acceptsHTML(request.Header.Get("Accept"))
}
