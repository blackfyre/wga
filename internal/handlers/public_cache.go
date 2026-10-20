package handlers

import (
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

// publicCatalogueCacheControl lets a shared cache keep anonymous catalogue HTML
// for five minutes while browsers always revalidate.
const publicCatalogueCacheControl = "public, max-age=0, s-maxage=300"

// publicCataloguePaths are the route families whose anonymous documents carry
// no visitor state. A family matches its exact path and every path below it.
var publicCataloguePaths = []string{
	"/artists",
	"/artworks",
	"/dual-mode",
	"/glossary",
	"/pages",
	"/search",
	"/statistics",
	"/timeline",
	"/tours",
}

// registerPublicCatalogueCache marks successful anonymous catalogue documents
// as shared-cacheable. HTMX responses stay private because one URL serves both
// a document and a fragment and the edge cannot vary on request headers. A
// handler that renders visitor state opts out with utils.KeepResponsePrivate,
// and any response that sets a cookie or is not a 200 is left unmarked.
func registerPublicCatalogueCache(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			if !isCatalogueRead(e.Request) {
				return e.Next()
			}
			if utils.IsHtmxRequest(e) {
				utils.KeepResponsePrivate(e)
				return e.Next()
			}
			if !negotiatesHTML(e.Request) {
				return e.Next()
			}
			e.Response = &publicCacheWriter{ResponseWriter: e.Response}
			return e.Next()
		})
		return se.Next()
	})
}

// isCatalogueRead reports whether a request reads an opted-in catalogue route.
func isCatalogueRead(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := r.URL.Path
	if path == "/" {
		return true
	}
	for _, family := range publicCataloguePaths {
		if path == family || strings.HasPrefix(path, family+"/") {
			return true
		}
	}
	return false
}

// negotiatesHTML reports whether a request takes the browser variant. Other
// Accept values render without operator head markup or the agent description
// link, and the edge cannot vary its cache on Accept, so only this variant may
// be shared.
func negotiatesHTML(r *http.Request) bool {
	accept := strings.Join(r.Header.Values("Accept"), ",")
	return acceptsHTML(accept) && !prefersMarkdown(accept)
}

// publicCacheWriter decides the cache policy when the status is written, after
// the handler has had every chance to set cookies or an explicit policy.
type publicCacheWriter struct {
	http.ResponseWriter
	decided bool
}

func (w *publicCacheWriter) WriteHeader(status int) {
	w.decide(status)
	w.ResponseWriter.WriteHeader(status)
}

func (w *publicCacheWriter) Write(body []byte) (int, error) {
	w.decide(http.StatusOK)
	return w.ResponseWriter.Write(body)
}

func (w *publicCacheWriter) decide(status int) {
	if w.decided {
		return
	}
	w.decided = true

	header := w.Header()
	if status != http.StatusOK || header.Get("Cache-Control") != "" || len(header.Values("Set-Cookie")) > 0 {
		return
	}
	header.Set("Cache-Control", publicCatalogueCacheControl)
}

// Unwrap exposes the wrapped writer to PocketBase's status tracking and to
// http.ResponseController.
func (w *publicCacheWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
