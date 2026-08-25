package timeline

import (
	"bytes"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// processTimeline is a thin request/render adapter. It delegates parsing,
// querying, and read-model assembly to buildTimelineView and only maps the
// result onto the HTTP response (ADR 0007).
func processTimeline(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	view, canonicalURL, err := buildTimelineView(newRepository(app), c.Request.URL.Query())
	if err != nil {
		logging.RequestLogger(app, c).Error("Build timeline", "error", err)
		return utils.ServerFaultError(c)
	}

	c.Response.Header().Set("HX-Push-Url", canonicalURL)

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Timeline")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Explore the collection by date: published works and art periods across a window of years.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, tmplUtils.AssetUrl(canonicalURL))

	var buffer bytes.Buffer
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err = pages.TimelineBlock(view).Render(ctx, &buffer)
	} else {
		err = pages.TimelinePage(view).Render(ctx, &buffer)
	}
	if err != nil {
		logging.RequestLogger(app, c).Error("Render timeline", "error", err)
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buffer.String())
}

// RegisterHandlers registers the timeline route. Central registration in
// internal/handlers/main.go is owned by the serial integrator.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/timeline", func(c *core.RequestEvent) error {
			return processTimeline(app, c)
		})
		return se.Next()
	})
}
