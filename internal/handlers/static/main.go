package static

import (
	"bytes"
	"database/sql"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func getFilePublicSystem() fs.FS {
	fsys, err := fs.Sub(assets.PublicFiles, "public")

	if err != nil {
		panic(err)
	}

	return fsys
}

func shouldRegisterVisualOverhaul(environment config.Environment) bool {
	return environment != config.EnvironmentProduction
}

func assetCacheControl(path string) string {
	if path == "js/app.js" {
		return "no-cache"
	}
	if strings.HasPrefix(path, "js/") && strings.HasSuffix(path, ".js") {
		return "public, max-age=31536000, immutable"
	}
	return ""
}

// RegisterHandlers registers the static routes for the application.
// It adds a middleware to serve static assets and a handler to serve static pages.
// The static pages are retrieved from the database based on the slug parameter in the URL.
// If the request is an Htmx request, only the content block is rendered, otherwise the entire page is rendered.
// The function returns an error if there was a problem registering the routes.
func RegisterHandlers(app core.App, environment config.Environment) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if shouldRegisterVisualOverhaul(environment) {
			se.Router.GET("/tmp/visual-overhaul", func(c *core.RequestEvent) error {
				content, err := assets.ReferenceFiles.ReadFile("reference/visual-overhaul.html")
				if err != nil {
					app.Logger().Error("Failed to read visual overhaul reference", "error", err)
					return utils.ServerFaultError(c, utils.ServerFailure{
						Category: "visual_overhaul_reference_read",
						Cause:    err,
					})
				}

				c.Response.Header().Set("Cache-Control", "no-store")
				return c.Blob(http.StatusOK, "text/html; charset=utf-8", content)
			})
			se.Router.GET("/tmp/visual-overhaul/footer", func(c *core.RequestEvent) error {
				return visualOverhaulFooterFixture(app, c)
			})
		}

		// Assets
		assetHandler := apis.Static(getFilePublicSystem(), false)
		if app.IsDev() {
			assetHandler = apis.Static(os.DirFS("../internal/assets/public"), false)
		}
		se.Router.GET("/assets/{path...}", func(c *core.RequestEvent) error {
			if cacheControl := assetCacheControl(c.Request.PathValue("path")); cacheControl != "" {
				c.Response.Header().Set("Cache-Control", cacheControl)
			}
			return assetHandler(c)
		})

		// Sitemap
		se.Router.GET("/sitemap/*", apis.Static(os.DirFS("./wga_sitemap"), false))

		// "Static" pages
		se.Router.GET("/pages/{slug}", func(c *core.RequestEvent) error {
			return staticPageHandler(app, c)
		})

		se.Router.GET("/error_404", func(c *core.RequestEvent) error {
			c.Response.Header().Set("HX-Push-Url", "/error_404")
			return utils.NotFoundError(c)
		})

		// The landing home route is registered as "GET /", which in Go's
		// ServeMux doubles as a catch-all for every unmatched GET/HEAD path.
		// This middleware recognises that fall-through after matching and
		// turns it into the correct public or technical 404, leaving the root
		// home request, every known route and all non-GET methods untouched.
		se.Router.BindFunc(publicRouteFallback)

		return se.Next()
	})
}

// visualOverhaulFooterFixture supplies a server-rendered footer fixture for
// development-only browser coverage. An HTMX request receives only the footer
// fragment; direct navigation receives a minimal document whose initial button
// performs an outer footer swap against the same endpoint.
func visualOverhaulFooterFixture(app core.App, c *core.RequestEvent) error {
	ctx := tmplUtils.ContextFromRequest(c.Request)
	var footer bytes.Buffer
	if err := components.Footer().Render(ctx, &footer); err != nil {
		return staticPageServerError(app, c, "visual_overhaul_footer_render_error", err)
	}

	c.Response.Header().Set("Cache-Control", "no-store")
	c.Response.Header().Set("Vary", "HX-Request")
	if c.Request.Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, footer.String())
	}

	page := `<!doctype html><html><head><meta charset="utf-8"><script type="module" src="/assets/js/app.js"></script></head><body><main><button type="button" hx-get="/tmp/visual-overhaul/footer" hx-target="footer" hx-swap="outerHTML">Replace footer</button></main>` + footer.String() + `</body></html>`
	return c.HTML(http.StatusOK, page)
}

func staticPageHandler(app core.App, c *core.RequestEvent) error {
	slug := c.Request.PathValue("slug")
	fullUrl := tmplUtils.AssetUrl("/pages/" + slug)
	pushUrl := utils.GenerateCurrentRelativePageUrl(c)

	page, err := app.FindFirstRecordByData(constants.CollectionStaticPages, "slug", slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.NotFoundError(c)
		}

		return staticPageServerError(app, c, "fetch_error", err)
	}

	contentHTML, toc, err := withTableOfContents(page.GetString("content"))
	if err != nil {
		return staticPageServerError(app, c, "contents_error", err)
	}

	content := pages.StaticPageDTO{
		Title:   page.GetString("title"),
		Content: contentHTML,
		Url:     "/pages/" + page.GetString("slug"),
		TOC:     toc,
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, page.GetString("title"))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, page.GetString("content"))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

	c.Response.Header().Set("HX-Push-Url", pushUrl)

	var buf bytes.Buffer

	if err := pages.StaticPage(content).Render(ctx, &buf); err != nil {
		return staticPageServerError(app, c, "render_error", err)
	}

	return c.HTML(http.StatusOK, buf.String())
}

// landingHomePattern is the ServeMux pattern registered by the landing home
// route. Because it is a bare "GET /" subtree, it also matches every
// unmatched GET/HEAD path, which is how the middleware below distinguishes a
// genuine home request from a public fall-through.
const landingHomePattern = "GET /"

// publicRouteFallback is a router middleware that runs after ServeMux has
// matched the request. When the request matched the landing home pattern but
// is not actually for "/", it was a fall-through. Only an actual GET miss on a
// public path receives the shared HTML 404; every other fall-through (HEAD or
// any non-GET method, plus technical boundaries) keeps PocketBase's native
// 404/405 shape. Root home and all known routes proceed unchanged.
func publicRouteFallback(e *core.RequestEvent) error {
	if e.Request.Pattern == landingHomePattern && e.Request.URL.Path != "/" {
		if e.Request.Method == http.MethodGet && !isReservedBoundary(e.Request.URL.Path) {
			return utils.NotFoundError(e)
		}

		return e.NotFoundError("", nil)
	}

	return e.Next()
}

// staticPageServerError logs a request-scoped failure with the internal detail
// confined to redacted/error-type fields and returns the shared server-fault page.
func staticPageServerError(app core.App, c *core.RequestEvent, outcome string, err error) error {
	logging.RequestLogger(app, c).Error("Static page request failed",
		"event", "static_page.request.failed",
		"outcome", outcome,
		"page", c.Request.PathValue("slug"),
		"error_type", logging.ErrorType(err),
		"error", logging.Redact(err),
	)

	return utils.ServerFaultError(c, utils.ServerFailure{Category: outcome, Cause: err})
}

// reservedBoundaryPrefixes are path roots owned by PocketBase or the
// application's technical surfaces. Requests under them must retain their
// native API/static semantics instead of the public HTML 404 page.
var reservedBoundaryPrefixes = []string{
	"/api",
	"/_",
	"/assets",
	"/sitemap",
	"/tmp/visual-overhaul",
}

// isReservedBoundary reports whether path names a technical boundary that must
// not be answered with the public HTML 404 page.
func isReservedBoundary(path string) bool {
	for _, prefix := range reservedBoundaryPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}

	return false
}
