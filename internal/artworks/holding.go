package artworks

import (
	"html"
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
	parts := strings.Split(html.UnescapeString(tmplUtils.StripHtmlTags(comment)), summarySeparator)
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

// CurrentLocations resolves the source-backed present holding names of several
// artworks in one lookup, keyed by artwork id. Artworks whose relation is unset
// or unresolvable are absent from the map; a failed lookup yields an empty map,
// so callers omit the holding rather than guess it.
func CurrentLocations(app core.App, artworks []*core.Record) map[string]string {
	names := map[string]string{}
	if app == nil || len(artworks) == 0 {
		return names
	}

	locationByArtwork := make(map[string]string, len(artworks))
	locationIDs := make([]string, 0, len(artworks))
	seen := map[string]bool{}
	for _, artwork := range artworks {
		if artwork == nil {
			continue
		}
		ids := artwork.GetStringSlice("current_location_id")
		if len(ids) == 0 {
			continue
		}
		locationByArtwork[artwork.Id] = ids[0]
		if !seen[ids[0]] {
			seen[ids[0]] = true
			locationIDs = append(locationIDs, ids[0])
		}
	}
	if len(locationIDs) == 0 {
		return names
	}

	locations, err := app.FindRecordsByIds(constants.CollectionLocations, locationIDs)
	if err != nil {
		return names
	}
	nameByLocation := make(map[string]string, len(locations))
	for _, location := range locations {
		if name := strings.TrimSpace(location.GetString("name")); name != "" {
			nameByLocation[location.Id] = name
		}
	}
	for artworkID, locationID := range locationByArtwork {
		if name, ok := nameByLocation[locationID]; ok {
			names[artworkID] = name
		}
	}

	return names
}

// FindPublished returns the published artwork with the given id.
func FindPublished(app core.App, id string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(constants.CollectionArtworks, "id = {:id} && published = true", map[string]any{"id": id})
}
