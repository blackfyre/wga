package itineraries

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

// builderState is the volatile, URL/form-carried builder UI state that must
// survive a mutation response: the picker query, its disclosure, and the
// selected stop index. It is never persisted.
type builderState struct {
	Query      string
	PickerOpen bool
	Selected   int
}

func hasCompleteItineraryArtistIdentity(author *core.Record) bool {
	if author == nil {
		return false
	}
	return strings.TrimSpace(author.GetString("filing_name")) != "" && strings.TrimSpace(author.GetString("short_name")) != ""
}

// loadTrayView maps the workflow tray projection to its templ DTO, resolving
// tray thumbnails at the 120px profile.
func loadTrayView(app core.App, owner string) (dto.ItineraryTrayView, error) {
	view := dto.ItineraryTrayView{BuilderURL: "/itineraries/new"}

	tray, err := itineraryworkflow.LoadTrayView(app, owner)
	if err != nil {
		return view, err
	}
	view.Count = tray.Count

	for _, artworkID := range tray.ArtworkIDs {
		artwork, err := app.FindRecordById(constants.CollectionArtworks, artworkID)
		if err != nil || artwork.GetString("image") == "" {
			continue
		}
		view.Thumbs = append(view.Thumbs, url.GenerateArtworkImageURL(artwork, url.DeliveryProfileItineraryTray, ""))
	}

	return view, nil
}

// stopDTO maps a persisted stop to its display projection, reading the artwork
// through its relation and marking it unavailable when the artwork is gone.
func stopDTO(app core.App, stop *core.Record) dto.ItineraryStop {
	out := dto.ItineraryStop{
		ID:        stop.Id,
		ArtworkID: stop.GetString("artwork"),
		Title:     stop.GetString("title"),
		Narration: stop.GetString("narration"),
		Position:  stop.GetInt("position"),
	}

	artwork, err := app.FindRecordById(constants.CollectionArtworks, out.ArtworkID)
	if err != nil {
		out.Unavailable = true
		return out
	}

	if !artwork.GetBool("published") {
		out.Unavailable = true
		out.Title = stop.GetString("title")
		return out
	}

	out.Title = artwork.GetString("title")
	out.ImageURL = resolveArtworkImage(app, artwork, url.DeliveryProfileCardAndArtistIndex)
	if year := artwork.GetInt("date_start"); year > 0 {
		out.Date = strconv.Itoa(year)
	}
	out.School = stopSchool(app, artwork)

	if errs := app.ExpandRecord(artwork, []string{"author"}, nil); len(errs) == 0 {
		author := artwork.ExpandedOne("author")
		if hasCompleteItineraryArtistIdentity(author) {
			out.Artist = author.GetString("filing_name")
			out.URL = url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   author.GetString("name"),
				ArtistId:     author.Id,
				ArtworkTitle: artwork.GetString("title"),
				ArtworkId:    artwork.Id,
			})
		} else {
			out.Artist = ""
			out.URL = ""
			out.Unavailable = true
		}
	} else {
		out.Unavailable = true
	}

	return out
}

// stopsDTO maps an ordered stop list to its display projections.
func stopsDTO(app core.App, stops []*core.Record) []dto.ItineraryStop {
	out := make([]dto.ItineraryStop, 0, len(stops))
	for _, stop := range stops {
		out = append(out, stopDTO(app, stop))
	}

	return out
}

// stopSchool resolves an artwork's school relation ids to their display names,
// joining them in order and skipping any id that no longer resolves. It returns
// an empty string when the artwork carries no schools, so the builder renders
// no fabricated metadata.
func stopSchool(app core.App, artwork *core.Record) string {
	ids := artwork.GetStringSlice("school")
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		school, err := app.FindRecordById(constants.CollectionSchools, id)
		if err != nil {
			continue
		}
		if name := school.GetString("name"); name != "" {
			names = append(names, name)
		}
	}

	return strings.Join(names, ", ")
}

// loadBuilderView assembles the builder page/block projection. It never
// creates durable state: when no draft exists it renders an empty builder
// projection (no id, no stops) so cookie-less GETs allocate no records. The
// picker results are loaded only while the disclosure is open.
func loadBuilderView(app core.App, owner string, csrf string, state builderState) (dto.ItineraryBuilderView, error) {
	view := dto.ItineraryBuilderView{
		Max:        itineraryworkflow.MaxStops,
		CSRF:       csrf,
		Query:      state.Query,
		PickerOpen: state.PickerOpen,
		Selected:   state.Selected,
	}

	draft, err := itineraryworkflow.LoadDraft(app, owner)
	switch {
	case err == nil:
		view.ID = draft.Record.Id
		view.Title = draft.Record.GetString("title")
		view.Intro = draft.Record.GetString("intro")
		view.Creator = draft.Record.GetString("creator")
		view.Listed = draft.Record.GetBool("listed")
		view.Stops = stopsDTO(app, draft.Stops)
		view.Count = len(view.Stops)
		view.AtLimit = view.Count >= itineraryworkflow.MaxStops
		view.Duration = itineraryworkflow.EstimateDuration(draft.Stops)
	case errors.Is(err, sql.ErrNoRows):
		// No draft yet: leave the empty projection for the guarded POST to
		// allocate one.
	default:
		return view, err
	}

	view.Selected = clampSelected(view.Selected, view.Count)

	if view.PickerOpen {
		picker, err := pickerWorks(app, owner, state.Query)
		if err != nil {
			return view, err
		}
		view.Picker = picker
	}

	return view, nil
}

// clampSelected keeps the selected stop index inside the current stop range,
// collapsing to zero when the draft is empty.
func clampSelected(selected int, count int) int {
	if selected < 0 {
		return 0
	}
	if count > 0 && selected >= count {
		return count - 1
	}

	return selected
}

// pickerWorks returns a bounded set of published artworks for the builder's
// add picker, filtered by an optional title-or-artist query.
func pickerWorks(app core.App, owner string, query string) ([]dto.ItineraryPickerWork, error) {
	filter := "published = true"
	params := map[string]any{}
	if query != "" {
		filter = "published = true && (title ~ {:query} || author.filing_name ~ {:query})"
		params["query"] = query
	}

	records, err := app.FindRecordsByFilter(constants.CollectionArtworks, filter, "-created", 12, 0, params)
	if err != nil {
		return nil, err
	}

	out := make([]dto.ItineraryPickerWork, 0, len(records))
	for _, artwork := range records {
		work := dto.ItineraryPickerWork{
			ArtworkID: artwork.Id,
			Title:     artwork.GetString("title"),
			ImageURL:  resolveArtworkImage(app, artwork, url.DeliveryProfileSearchRow),
			Added:     itineraryworkflow.IsAdded(app, owner, artwork.Id),
		}
		if year := artwork.GetInt("date_start"); year > 0 {
			work.Date = strconv.Itoa(year)
		}
		if errs := app.ExpandRecord(artwork, []string{"author"}, nil); len(errs) == 0 {
			author := artwork.ExpandedOne("author")
			if !hasCompleteItineraryArtistIdentity(author) {
				continue
			}
			work.Artist = author.GetString("filing_name")
		} else {
			continue
		}
		out = append(out, work)
	}

	return out, nil
}

// loadIndexView assembles the public listed-itinerary index.
func loadIndexView(app core.App) (dto.ItineraryIndexView, error) {
	view := dto.ItineraryIndexView{Itineraries: []dto.ItinerarySummary{}}

	records, err := itineraryworkflow.ListPublished(app, 50)
	if err != nil {
		return view, err
	}

	view.Total = len(records)

	for _, record := range records {
		stops, err := itineraryworkflow.LoadStops(app, record.Id)
		if err != nil {
			stops = nil
		}
		view.Itineraries = append(view.Itineraries, dto.ItinerarySummary{
			Title:     record.GetString("title"),
			Creator:   displayCreator(record.GetString("creator")),
			URL:       "/itineraries/" + record.GetString("token"),
			Note:      record.GetString("intro"),
			Count:     len(stops),
			Duration:  itineraryworkflow.EstimateDuration(stops),
			Published: formatPublished(record),
		})
	}

	return view, nil
}

// loadConfirmationView assembles the session-owned publication receipt.
func loadConfirmationView(app core.App, record *core.Record) dto.ItineraryPublishedView {
	stops, err := itineraryworkflow.LoadStops(app, record.Id)
	if err != nil {
		stops = nil
	}

	token := record.GetString("token")
	return dto.ItineraryPublishedView{
		Title:     record.GetString("title"),
		URL:       utils.AssetUrl("/itineraries/" + token),
		TokenURL:  "/itineraries/" + token,
		Creator:   displayCreator(record.GetString("creator")),
		Expires:   record.GetDateTime("expires_at").Time().Format("2006-01-02"),
		Published: formatPublished(record),
		Duration:  itineraryworkflow.EstimateDuration(stops),
		StopCount: len(stops),
		Listed:    record.GetBool("listed"),
	}
}

// formatPublished renders a publication date in the reference's pre-formatted
// "22 JUL 2026" form.
func formatPublished(record *core.Record) string {
	published := record.GetDateTime("published")
	if published.IsZero() {
		return ""
	}

	return strings.ToUpper(published.Time().Format("02 Jan 2006"))
}

// resolveArtworkImage returns the no-image fallback when the artwork has none.
func resolveArtworkImage(app core.App, artwork *core.Record, profile url.DeliveryProfile) string {
	if artwork.GetString("image") == "" {
		return utils.AssetUrl("/assets/images/no-image.png")
	}

	return url.GenerateArtworkImageURL(artwork, profile, "")
}

func displayCreator(creator string) string {
	if creator == "" {
		return "Anon."
	}

	return creator
}
