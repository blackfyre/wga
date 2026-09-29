package artworks

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"strings"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
)

const artworkSearchPageName = "Artworks Search"

// artworkSearchTitleSteps is the reduction order for artwork-search titles:
// the page position goes first, then filters shorten and are counted from the
// last rail control, and the free-text leads are truncated last.
var artworkSearchTitleSteps = []tmplUtils.TitleStep{
	{Kind: tmplUtils.TitleRemove, Key: "position"},
	{Kind: tmplUtils.TitleShorten, Key: "form"},
	{Kind: tmplUtils.TitleShorten, Key: "school"},
	{Kind: tmplUtils.TitleShorten, Key: "technique"},
	{Kind: tmplUtils.TitleShorten, Key: "artist"},
	{Kind: tmplUtils.TitleShorten, Key: "title"},
	{Kind: tmplUtils.TitleRemove, Key: "year"},
	{Kind: tmplUtils.TitleRemove, Key: "collection"},
	{Kind: tmplUtils.TitleRemove, Key: "period"},
	{Kind: tmplUtils.TitleRemove, Key: "type"},
	{Kind: tmplUtils.TitleRemove, Key: "form"},
	{Kind: tmplUtils.TitleRemove, Key: "school"},
	{Kind: tmplUtils.TitleRemove, Key: "technique"},
	{Kind: tmplUtils.TitleRemove, Key: "artist"},
	{Kind: tmplUtils.TitleRemove, Key: "title"},
	{Kind: tmplUtils.TitleTruncate, Key: "scope", Limit: 30},
	{Kind: tmplUtils.TitleTruncate, Key: "q", Limit: 20},
}

// artworkSearchTitleLabels carries the display labels a title needs. Maps are
// only populated for active filters. A value without a label is left out of
// the title rather than shown as a raw slug or record ID.
type artworkSearchTitleLabels struct {
	artistScope string
	schools     map[string]string
	forms       map[string]string
	types       map[string]string
	periods     map[string]string
	collection  string
}

// resolveArtworkSearchTitleLabels loads only the labels of active filters,
// from the same cached option projections the filter rail uses, and the
// public artist scope when an exact artist filter is set.
func resolveArtworkSearchTitleLabels(ctx context.Context, app *pocketbase.PocketBase, f *filters, checkpoint artworkSearchCheckpoint) (labels artworkSearchTitleLabels, err error) {
	if labels.artistScope, err = resolveArtworkSearchArtistScope(ctx, app, f.ArtistID, checkpoint); err != nil {
		return labels, err
	}

	needsLabels := len(f.schoolValues()) > 0 || len(f.formValues()) > 0 || f.ArtTypeString != "" || f.PeriodString != "" || f.selectedVenue() != ""
	if !needsLabels {
		return labels, nil
	}
	if err = checkpoint(ctx, "artworks.search.title_labels"); err != nil {
		return labels, err
	}
	if len(f.schoolValues()) > 0 {
		if labels.schools, err = getArtSchoolOptions(app); err != nil {
			return labels, err
		}
	}
	if len(f.formValues()) > 0 {
		if labels.forms, err = getArtFormOptions(app); err != nil {
			return labels, err
		}
	}
	if f.ArtTypeString != "" {
		if labels.types, err = getArtTypesOptions(app); err != nil {
			return labels, err
		}
	}
	if f.PeriodString != "" {
		periods, periodErr := getArtPeriodOptions(app)
		if periodErr != nil {
			return labels, periodErr
		}
		labels.periods = make(map[string]string, len(periods.entries))
		for _, entry := range periods.entries {
			labels.periods[entry.value] = entry.label
		}
	}
	if venue := f.selectedVenue(); venue != "" {
		// The collection value is a location record ID. One record lookup
		// avoids the catalogue-wide holdings projection.
		location, lookupErr := app.FindRecordById(constants.CollectionLocations, venue)
		if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
			return labels, lookupErr
		}
		if location != nil {
			labels.collection = strings.TrimSpace(location.GetString("name"))
		}
	}

	return labels, nil
}

// artworkSearchTitle derives the document title from canonical filter state.
// View and sort are presentation only and never contribute.
func artworkSearchTitle(f *filters, labels artworkSearchTitleLabels, page int, pageCount int) string {
	parts := []tmplUtils.TitlePart{
		{Key: "q", Long: f.Query, Role: tmplUtils.TitleLead, Quoted: true},
		{Key: "scope", Long: labels.artistScope, Role: tmplUtils.TitleLead},
		{Key: "page", Long: artworkSearchPageName, Role: tmplUtils.TitlePageName},
	}
	if f.Title != "" {
		parts = append(parts, quotedFilterPart("title", "title", f.Title))
	}
	if f.ArtistID == "" && f.ArtistString != "" {
		parts = append(parts, quotedFilterPart("artist", "artist", f.ArtistString))
	}
	if f.TechniqueString != "" {
		parts = append(parts, quotedFilterPart("technique", "technique", f.TechniqueString))
	}
	parts = append(parts,
		multiValueFilterPart("school", f.schoolValues(), labels.schools),
		multiValueFilterPart("form", f.formValues(), labels.forms),
		tmplUtils.TitlePart{Key: "type", Long: titleLabel(labels.types, f.ArtTypeString), Role: tmplUtils.TitleFilter},
		tmplUtils.TitlePart{Key: "period", Long: titleLabel(labels.periods, f.PeriodString), Role: tmplUtils.TitleFilter},
		tmplUtils.TitlePart{Key: "collection", Long: labels.collection, Role: tmplUtils.TitleFilter},
		tmplUtils.TitlePart{Key: "year", Long: yearTitleLabel(f.YearFrom, f.YearTo), Role: tmplUtils.TitleFilter},
		tmplUtils.TitlePart{Key: "position", Long: tmplUtils.TitlePagePosition(page, pageCount), Role: tmplUtils.TitlePosition},
	)

	return tmplUtils.FitTitle(parts, artworkSearchTitleSteps)
}

func quotedFilterPart(key string, name string, value string) tmplUtils.TitlePart {
	quoted := "“" + value + "”"
	return tmplUtils.TitlePart{Key: key, Long: name + " " + quoted, Short: quoted, Role: tmplUtils.TitleFilter}
}

// multiValueFilterPart lists every selected label in case-insensitive
// alphabetical order, so the same selection always yields the same title
// whatever the parameter order, and shortens to the first label plus a count
// of the others.
func multiValueFilterPart(key string, values []string, labels map[string]string) tmplUtils.TitlePart {
	if len(values) == 0 {
		return tmplUtils.TitlePart{Key: key, Role: tmplUtils.TitleFilter}
	}

	names := make([]string, 0, len(values))
	for _, value := range values {
		if name := titleLabel(labels, value); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return tmplUtils.TitlePart{Key: key, Role: tmplUtils.TitleFilter}
	}
	slices.SortFunc(names, func(left, right string) int {
		if order := strings.Compare(strings.ToLower(left), strings.ToLower(right)); order != 0 {
			return order
		}
		return strings.Compare(left, right)
	})
	part := tmplUtils.TitlePart{Key: key, Long: strings.Join(names, ", "), Role: tmplUtils.TitleFilter}
	if len(names) > 1 {
		part.Short = names[0] + " +" + strconv.Itoa(len(names)-1)
	}
	return part
}

func titleLabel(labels map[string]string, value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(labels[value])
}

func yearTitleLabel(from string, to string) string {
	if from == "" && to == "" {
		return ""
	}
	return strconv.Itoa(artworkFacetYear(from, artworkYearMin)) + "–" + strconv.Itoa(artworkFacetYear(to, artworkYearMax))
}
