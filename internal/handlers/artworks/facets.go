package artworks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
)

const (
	artworkYearMin = 200
	artworkYearMax = 1900
)

func buildArtworkSearchFacets(
	f *filters,
	schoolGroup dto.ChipGroup,
	formGroup dto.ChipGroup,
	typeGroup dto.ChipGroup,
	periodGroup dto.ChipGroup,
	venueOptions venueFacetOptions,
) pages.ArtworkSearchFacets {
	yearFrom := artworkFacetYear(f.YearFrom, artworkYearMin)
	yearTo := artworkFacetYear(f.YearTo, artworkYearMax)
	yearActive := f.YearFrom != "" || f.YearTo != ""
	venueActive := f.selectedVenue() != ""

	collectionOptions := make([]pages.ArtworkSearchCollectionOption, 0, len(venueOptions.entries))
	venueSummary := f.selectedVenue()
	for _, option := range venueOptions.entries {
		selected := option.value == f.selectedVenue()
		collectionOptions = append(collectionOptions, pages.ArtworkSearchCollectionOption{
			Label:    option.label,
			Value:    option.value,
			Count:    option.count,
			Selected: selected,
		})
		if selected {
			venueSummary = option.label
		}
	}

	return pages.ArtworkSearchFacets{
		ActiveCount: f.ActiveFilterCount(),
		Query: pages.ArtworkSearchFacet{
			Label:   "TITLE OR ARTIST",
			Summary: quotedFacetSummary(f.Query),
			Active:  f.Query != "",
			Open:    true,
		},
		Technique: pages.ArtworkSearchFacet{
			Label:   "TECHNIQUE",
			Summary: quotedFacetSummary(f.TechniqueString),
			Active:  f.TechniqueString != "",
			Open:    f.TechniqueString != "",
		},
		School: chipFacet("SCHOOL", schoolGroup, f.SchoolString, true),
		Form:   chipFacet("FORM", formGroup, f.ArtFormString, false),
		Type:   chipFacet("TYPE", typeGroup, f.ArtTypeString, false),
		Period: chipFacet("PERIOD", periodGroup, f.PeriodString, false),
		Collection: pages.ArtworkSearchCollectionFacet{
			Facet: pages.ArtworkSearchFacet{
				Label:   "COLLECTION",
				Summary: collectionFacetSummary(venueSummary),
				Active:  venueActive,
				Open:    venueActive || f.VenueQuery != "",
			},
			Name: "venue",
			QueryField: dto.Field{
				ID:          "artwork-venue-query",
				Name:        "venue_q",
				Label:       "FILTER COLLECTIONS BY NAME",
				Type:        "search",
				Value:       f.VenueQuery,
				Placeholder: "filter collections",
			},
			Options:         collectionOptions,
			Note:            venueFacetNote(venueOptions),
			TotalOptions:    venueOptions.totalOptions,
			OmittedOptions:  venueOptions.omittedOptions,
			OmittedHoldings: venueOptions.omittedHoldings,
		},
		Year: pages.ArtworkSearchFacet{
			Label:   "YEAR RANGE",
			Summary: fmt.Sprintf("%d–%d", yearFrom, yearTo),
			Active:  yearActive,
			Open:    yearActive,
			Last:    true,
		},
		YearRange: dto.RangeField{
			Label:     "YEAR RANGE",
			FromID:    "year_from",
			FromName:  "year_from",
			FromValue: yearFrom,
			ToID:      "year_to",
			ToName:    "year_to",
			ToValue:   yearTo,
			Min:       artworkYearMin,
			Max:       artworkYearMax,
			Step:      10,
			Inline:    true,
		},
	}
}

func chipFacet(label string, group dto.ChipGroup, selected string, openByDefault bool) pages.ArtworkSearchFacet {
	summary := "ANY"
	active := false
	for _, option := range group.Options {
		if option.Checked && option.Value != "" {
			summary = strings.ToUpper(option.Label)
			active = true
			break
		}
	}

	// A non-empty selected value that no option matched is an unknown direct
	// value. It remains a real result predicate, so the facet must stay active
	// and auto-open with a truthful summary derived from the selected value
	// rather than misleadingly reading ANY.
	if !active && selected != "" {
		summary = strings.ToUpper(selected)
		active = true
	}

	return pages.ArtworkSearchFacet{
		Label:   label,
		Summary: summary,
		Active:  active,
		Open:    openByDefault || active,
	}
}

func quotedFacetSummary(value string) string {
	if value == "" {
		return "ANY"
	}

	return "“" + value + "”"
}

func collectionFacetSummary(value string) string {
	if value == "" {
		return "ANY"
	}

	houseName := strings.TrimSpace(strings.Split(value, ",")[0])
	return strings.ToUpper(houseName)
}

func artworkFacetYear(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
