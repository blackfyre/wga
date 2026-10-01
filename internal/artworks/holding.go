package artworks

import (
	"strings"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

// summarySeparator divides the catalogue summary that the importer writes into
// an artwork comment when no commentary exists: date · location · dimensions.
const summarySeparator = " · "

// LocationAndDimensions extracts the location and dimensions from an artwork's
// catalogue summary comment. The importer writes date · location · dimensions,
// or date · location when the source has no dimensions. A comment without a
// separator yields empty values, so callers omit the details rather than guess
// them.
func LocationAndDimensions(comment string) (location string, dimensions string) {
	parts := strings.Split(tmplUtils.StripHtmlTags(comment), summarySeparator)
	switch {
	case len(parts) >= 3:
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
	case len(parts) == 2:
		return strings.TrimSpace(parts[1]), ""
	default:
		return "", ""
	}
}

// CurrentLocation returns the name of the artwork's source-backed present
// holding, or an empty string when the relation is unset or unresolvable.
func CurrentLocation(app core.App, artwork *core.Record) string {
	if app == nil || artwork == nil {
		return ""
	}
	locationIDs := artwork.GetStringSlice("current_location_id")
	if len(locationIDs) == 0 {
		return ""
	}
	location, err := app.FindRecordById(constants.CollectionLocations, locationIDs[0])
	if err != nil {
		return ""
	}

	return strings.TrimSpace(location.GetString("name"))
}

// FindPublished returns the published artwork with the given id.
func FindPublished(app core.App, id string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(constants.CollectionArtworks, "id = {:id} && published = true", map[string]any{"id": id})
}
