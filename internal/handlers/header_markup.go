package handlers

import (
	"database/sql"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/pocketbase/pocketbase/core"
)

// trustedHeadMarkupName is the exact strings.name value for the optional
// operator-managed fragment rendered verbatim in the shared document head.
const trustedHeadMarkupName = "scripts:header"

// trustedHeadMarkupBoundaries are path roots that serve technical or
// non-document traffic. Requests under them must not receive the optional
// database-managed trusted head markup.
var trustedHeadMarkupBoundaries = []string{
	"/api",
	"/_",
	"/assets",
	"/health",
	"/sitemap",
	"/tmp/visual-overhaul",
}

// registerTrustedHeadMarkupMiddleware binds the header-markup middleware to the
// router. It must run before feature routes so the context is prepared for every
// eligible document request.
func registerTrustedHeadMarkupMiddleware(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return prepareTrustedHeadMarkup(app, e)
		})

		return se.Next()
	})
}

// prepareTrustedHeadMarkup reads the current scripts:header content once for an
// eligible request and carries it through the request context. Absence and empty
// content are normal; any other failure is logged without the record content and
// the request continues without the optional fragment.
func prepareTrustedHeadMarkup(app core.App, e *core.RequestEvent) error {
	if !eligibleForTrustedHeadMarkup(e.Request) {
		return e.Next()
	}

	record, err := app.FindFirstRecordByData(constants.CollectionStrings, "name", trustedHeadMarkupName)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logging.RequestLogger(app, e).Error("Trusted head markup lookup failed",
				"event", "header_markup.lookup.failed",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
		}

		return e.Next()
	}

	e.Request = e.Request.WithContext(
		tmplUtils.WithTrustedHeadMarkup(e.Request.Context(), record.GetString("content")),
	)

	return e.Next()
}

// eligibleForTrustedHeadMarkup reports whether a request is a full public HTML
// document request that should carry trusted head markup: a non-HTMX GET or HEAD
// request that negotiates HTML and does not target a technical boundary.
func eligibleForTrustedHeadMarkup(request *http.Request) bool {
	if request == nil {
		return false
	}

	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return false
	}

	if request.Header.Get("HX-Request") == "true" {
		return false
	}

	if !acceptsHTML(request.Header.Get("Accept")) {
		return false
	}

	return !isTrustedHeadMarkupBoundary(request.URL.Path)
}

// acceptsHTML reports whether the Accept header negotiates HTML documents.
func acceptsHTML(accept string) bool {
	for _, mediaRange := range strings.Split(accept, ",") {
		if htmlMediaRangeAccepted(mediaRange) {
			return true
		}
	}

	return false
}

// htmlMediaRangeAccepted reports whether a single Accept media range is exactly
// text/html (case-insensitive) with an acceptable quality value.
func htmlMediaRangeAccepted(mediaRange string) bool {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(mediaRange))
	if err != nil {
		return false
	}

	if mediaType != "text/html" {
		return false
	}

	return qualityAccepted(params["q"])
}

// qualityAccepted reports whether a quality value is valid and greater than
// zero. An absent q defaults to 1 (fully acceptable); malformed or out-of-range
// values are not acceptable.
func qualityAccepted(raw string) bool {
	if raw == "" {
		return true
	}

	quality, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return false
	}

	return quality > 0 && quality <= 1
}

// isTrustedHeadMarkupBoundary reports whether path names a technical boundary
// that must not receive database-managed trusted head markup.
func isTrustedHeadMarkupBoundary(path string) bool {
	for _, prefix := range trustedHeadMarkupBoundaries {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}

	return false
}
