package itineraries

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// indexPage renders the public listed-itinerary index.
func (ctx *securityContext) indexPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	view, err := loadIndexView(app)
	if err != nil {
		logging.RequestLogger(app, c).Error("Itinerary index load failed", "event", "itineraries.index.failed", "outcome", "load_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	ctxb := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Itineraries")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.DescriptionKey, "Visitor-made itineraries through the collection.")
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.CanonicalUrlKey, utils.AssetUrl("/itineraries"))

	var buf bytes.Buffer
	if err := pages.ItinerariesPage(view).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// viewPage renders one stop of a public itinerary slideshow. Only approved,
// unexpired itineraries are served; rejected or legacy-pending itineraries look
// indistinguishable from missing ones, while expired itineraries return 410.
func (ctx *securityContext) viewPage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	token := strings.TrimSpace(c.Request.PathValue("token"))
	if token == "" {
		return utils.NotFoundError(c)
	}

	record, err := itineraryworkflow.FindPublishedByToken(app, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.NotFoundError(c)
		}
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	if !itineraryworkflow.IsPublicStatus(record.GetString("status")) {
		return utils.NotFoundError(c)
	}

	if itineraryworkflow.IsExpired(record) {
		expiredCtx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Itinerary expired")
		var buf bytes.Buffer
		if err := pages.ItineraryExpiredPage().Render(expiredCtx, &buf); err != nil {
			return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
		}
		return c.HTML(http.StatusGone, buf.String())
	}

	stops, err := itineraryworkflow.AvailableStops(app, record.Id)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	if len(stops) == 0 {
		return utils.NotFoundError(c)
	}

	index := stopIndexFromQuery(c)
	if index < 0 || index >= len(stops) {
		index = 0
	}

	view, err := loadViewer(app, record, stops, index)
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	ctxb := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, record.GetString("title"))
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.DescriptionKey, record.GetString("intro"))
	ctxb = tmplUtils.DecorateContext(ctxb, tmplUtils.CanonicalUrlKey, utils.AssetUrl("/itineraries/"+token))

	var buf bytes.Buffer
	if err := pages.ItineraryViewPage(view).Render(ctxb, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buf.String())
}

func stopIndexFromQuery(c *core.RequestEvent) int {
	raw := strings.TrimSpace(c.Request.URL.Query().Get("stop"))
	if raw == "" {
		return 0
	}

	index, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return index
}

// loadViewer assembles the one-stop-at-a-time slideshow projection.
func loadViewer(app core.App, record *core.Record, stops []*core.Record, index int) (dto.ItineraryView, error) {
	view := dto.ItineraryView{
		Title:   record.GetString("title"),
		Creator: displayCreator(record.GetString("creator")),
		Total:   len(stops),
		Index:   index,
		HasPrev: index > 0,
		HasNext: index < len(stops)-1,
		ExitURL: "/itineraries",
	}

	token := record.GetString("token")
	view.PrevURL = viewerURL(token, index-1)
	view.NextURL = viewerURL(token, index+1)

	view.StopURLs = make([]string, len(stops))
	for i := range stops {
		view.StopURLs[i] = viewerURL(token, i)
	}

	stop := stops[index]
	view.Narration = stop.GetString("narration")
	view.StopTitle = stop.GetString("title")

	artwork, err := app.FindRecordById(constants.CollectionArtworks, stop.GetString("artwork"))
	if err == nil {
		view.StopTitle = artwork.GetString("title")
		view.Plate = viewerPlate(app, artwork)
		view.StopSchool = stopSchool(app, artwork)
		if year := artwork.GetInt("date_start"); year > 0 {
			view.StopDate = strconv.Itoa(year)
		}
		if errs := app.ExpandRecord(artwork, []string{"author"}, nil); len(errs) == 0 {
			if author := artwork.ExpandedOne("author"); author != nil {
				if hasCompleteItineraryArtistIdentity(author) {
					view.StopArtist = author.GetString("filing_name")
				}
			}
		}
	}

	return view, nil
}

// viewerPlate builds the deliberate artwork plate with the viewer zoom URL and
// the no-upscale fallback supplied by the URL utility.
func viewerPlate(app core.App, artwork *core.Record) dto.Plate {
	alt := artwork.GetString("title")
	label := artwork.GetString("title")

	if errs := app.ExpandRecord(artwork, []string{"author"}, nil); len(errs) == 0 {
		if author := artwork.ExpandedOne("author"); author != nil {
			if hasCompleteItineraryArtistIdentity(author) {
				shortName := author.GetString("short_name")
				alt = artwork.GetString("title") + " by " + shortName
				label = alt
			}
		}
	}

	if artwork.GetString("image") == "" {
		return dto.Plate{
			DisplayURL:  utils.AssetUrl("/assets/images/no-image.png"),
			Alt:         alt,
			Label:       label,
			Placeholder: artwork.GetString("title"),
			Aspect:      "aspect-[4/5]",
			Contain:     true,
		}
	}

	return dto.Plate{
		DisplayURL: url.GenerateArtworkImageURL(artwork, url.DeliveryProfileArtworkRecordTourPage, ""),
		ZoomURL:    url.GenerateArtworkImageURL(artwork, url.DeliveryProfileViewer, ""),
		Alt:        alt,
		Label:      label,
		Aspect:     "aspect-[4/5]",
		Contain:    true,
	}
}

func viewerURL(token string, index int) string {
	return "/itineraries/" + token + "?stop=" + strconv.Itoa(index)
}
