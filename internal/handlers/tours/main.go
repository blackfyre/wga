package tours

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	tourworkflow "github.com/blackfyre/wga/internal/tours"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		service := tourworkflow.NewService(app)
		se.Router.GET("/tours", func(event *core.RequestEvent) error {
			view, err := service.Index(event.Request.URL.Query().Get("kind"))
			if err != nil {
				logging.RequestLogger(app, event).Error("List guided tours", "error", err)
				return utils.ServerFaultError(event, utils.ServerFailure{Category: "server_fault", Cause: err})
			}
			canonical := "/tours"
			if view.Filter != "" {
				canonical += "?kind=" + url.QueryEscape(view.Filter)
			}
			event.Response.Header().Set("HX-Push-Url", canonical)
			ctx := templutils.DecorateContext(templutils.ContextFromRequest(event.Request), templutils.TitleKey, "Guided Tours")
			ctx = templutils.DecorateContext(ctx, templutils.DescriptionKey, "Permanent, revision-aware editorial tours of the collection.")
			ctx = templutils.DecorateContext(ctx, templutils.CanonicalUrlKey, templutils.AssetUrl("/tours"))
			var output bytes.Buffer
			if utils.IsHtmxRequest(event) && !utils.RequestsMainContentArea(event) {
				err = pages.ToursBlock(view).Render(ctx, &output)
			} else {
				err = pages.ToursPage(view).Render(ctx, &output)
			}
			if err != nil {
				return utils.ServerFaultError(event, utils.ServerFailure{Category: "server_fault", Cause: err})
			}
			return event.HTML(http.StatusOK, output.String())
		})
		se.Router.GET("/tours/{slug}", func(event *core.RequestEvent) error {
			return tourAddress(app, service, event, event.Request.PathValue("slug"), 1)
		})
		se.Router.GET("/tours/{slug}/{page}", func(event *core.RequestEvent) error {
			address, err := strconv.Atoi(event.Request.PathValue("page"))
			if err != nil {
				return tourLegacy(app, service, event)
			}
			return tourAddress(app, service, event, event.Request.PathValue("slug"), address)
		})
		se.Router.GET("/tour/{path...}", func(event *core.RequestEvent) error {
			return tourLegacy(app, service, event)
		})
		return se.Next()
	})
}

func tourAddress(app *pocketbase.PocketBase, service *tourworkflow.Service, event *core.RequestEvent, slug string, address int) error {
	view, err := service.Page(slug, address)
	if err == nil {
		return renderTourPage(app, event, view)
	}
	if !errors.Is(err, tourworkflow.ErrNotFound) {
		logging.RequestLogger(app, event).Error("Read guided tour", "error", err)
		return utils.ServerFaultError(event, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return tourLegacy(app, service, event)
}

func tourLegacy(app *pocketbase.PocketBase, service *tourworkflow.Service, event *core.RequestEvent) error {
	destination, err := service.LegacyRoute(event.Request.URL.EscapedPath())
	if err == nil {
		return event.Redirect(http.StatusMovedPermanently, destination)
	}
	if !errors.Is(err, tourworkflow.ErrNotFound) {
		logging.RequestLogger(app, event).Error("Resolve guided tour legacy route", "error", err)
		return utils.ServerFaultError(event, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return utils.NotFoundError(event)
}

func renderTourPage(app *pocketbase.PocketBase, event *core.RequestEvent, view dto.TourPage) error {
	canonical := "/tours/" + view.Slug
	if view.Address > 1 {
		canonical += "/" + strconv.Itoa(view.Address)
	}
	event.Response.Header().Set("HX-Push-Url", canonical)
	ctx := templutils.DecorateContext(templutils.ContextFromRequest(event.Request), templutils.TitleKey, view.PageTitle+" — "+view.TourTitle)
	ctx = templutils.DecorateContext(ctx, templutils.DescriptionKey, view.Blurb)
	ctx = templutils.DecorateContext(ctx, templutils.CanonicalUrlKey, templutils.AssetUrl(canonical))
	var output bytes.Buffer
	var err error
	if utils.IsHtmxRequest(event) && !utils.RequestsMainContentArea(event) {
		err = pages.TourBlock(view).Render(ctx, &output)
	} else {
		err = pages.TourPage(view).Render(ctx, &output)
	}
	if err != nil {
		return utils.ServerFaultError(event, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return event.HTML(http.StatusOK, output.String())
}
