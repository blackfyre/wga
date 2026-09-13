package artworks

import (
	"cmp"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
)

type filters struct {
	Query           string
	Title           string
	SchoolString    string
	SchoolValues    []string
	ArtFormString   string
	ArtFormValues   []string
	ArtTypeString   string
	ArtistString    string
	ArtistID        string
	TechniqueString string
	PeriodString    string
	VenueString     string
	VenueQuery      string
	VenueConflict   bool
	// LocationString is retained only as an internal compatibility alias for
	// callers that construct filters directly. Request parsing immediately
	// translates the obsolete location parameter into VenueString, and URLs
	// never serialise location.
	LocationString string
	YearFrom       string
	YearTo         string
	View           string
	Sort           string
	SortDir        string
	Page           string
	SchoolExpanded bool
	FormExpanded   bool
}

// AnyFilterActive checks if any filter is active. Sort and view are presentation
// state, not filters, so they are intentionally excluded.
func (f *filters) AnyFilterActive() bool {
	return f.ActiveFilterCount() > 0
}

// ActiveFilterCount returns the number of result-shaping filters. The
// collection option search is deliberately excluded because it narrows only
// the facet choices, not the artwork result set. The two year bounds count as
// one range filter.
func (f *filters) ActiveFilterCount() int {
	active := []bool{
		f.Query != "",
		f.Title != "",
		len(f.schoolValues()) > 0,
		len(f.formValues()) > 0,
		f.ArtTypeString != "",
		f.ArtistString != "",
		f.ArtistID != "",
		f.TechniqueString != "",
		f.PeriodString != "",
		f.selectedVenue() != "",
		f.YearFrom != "" || f.YearTo != "",
	}

	count := 0
	for _, isActive := range active {
		if isActive {
			count++
		}
	}

	return count
}

// FingerPrint returns a unique fingerprint string based on the filter values.
func (f *filters) FingerPrint() string {
	return f.Query + ":" + f.Title + ":" + strings.Join(f.schoolValues(), ",") + ":" + strings.Join(f.formValues(), ",") + ":" + f.ArtTypeString + ":" + f.ArtistString + ":" + f.ArtistID + ":" + f.TechniqueString + ":" + f.PeriodString + ":" + f.selectedVenue() + ":" + f.VenueQuery + ":" + f.YearFrom + ":" + f.YearTo + ":" + f.View + ":" + f.Sort + ":" + f.SortDir + ":" + f.Page
}

// BuildFilter builds the PocketBase filter string and parameters for the
// relation and scalar filters.
func (f *filters) BuildFilter() (string, dbx.Params) {
	filterString := "published = true && author:length > 0"
	params := dbx.Params{}

	if f.Query != "" {
		filterString = filterString + " && (title ~ {:query} || author.filing_name ~ {:query})"
		params["query"] = f.Query
	}

	if f.Title != "" {
		filterString = filterString + " && title ~ {:title}"
		params["title"] = f.Title
	}

	if values := f.schoolValues(); len(values) > 0 {
		filterString += " && " + anyFacetFilter("school.slug", "art_school", values, params)
	}

	if values := f.formValues(); len(values) > 0 {
		filterString += " && " + anyFacetFilter("form.slug", "art_form", values, params)
	}

	if f.ArtTypeString != "" {
		filterString = filterString + " && type.slug = {:art_type}"
		params["art_type"] = f.ArtTypeString
	}

	if f.ArtistID != "" {
		filterString = filterString + " && author.id ?= {:artist_id}"
		params["artist_id"] = f.ArtistID
	} else if f.ArtistString != "" {
		filterString = filterString + " && author.filing_name ~ {:artist}"
		params["artist"] = f.ArtistString
	}

	if f.TechniqueString != "" {
		filterString = filterString + " && technique ~ {:technique}"
		params["technique"] = f.TechniqueString
	}

	if f.PeriodString != "" {
		filterString = filterString + " && art_period_id = {:period}"
		params["period"] = f.PeriodString
	}

	if f.selectedVenue() != "" {
		filterString = filterString + " && current_location_id = {:location}"
		params["location"] = f.selectedVenue()
	}

	if f.YearFrom != "" {
		filterString = filterString + " && year >= {:year_from}"
		params["year_from"] = f.YearFrom
	}

	if f.YearTo != "" {
		filterString = filterString + " && year <= {:year_to}"
		params["year_to"] = f.YearTo
	}

	return filterString, params
}

// BuildFilterString builds a query string based on the values of the filters struct.
func (f *filters) BuildFilterString() string {
	return f.queryValues().Encode()
}

func (f *filters) BuildPath(basePath string) string {
	filterString := f.BuildFilterString()

	if filterString == "" {
		return basePath
	}

	return basePath + "?" + filterString
}

func (f *filters) queryValues() url.Values {
	values := url.Values{}

	if f.Title != "" {
		values.Set("title", f.Title)
	}

	if f.Query != "" {
		values.Set("q", f.Query)
	}

	for _, selected := range f.schoolValues() {
		values.Add("art_school", selected)
	}

	for _, selected := range f.formValues() {
		values.Add("art_form", selected)
	}

	if f.ArtTypeString != "" {
		values.Set("art_type", f.ArtTypeString)
	}

	if f.ArtistID != "" {
		values.Set("artist_id", f.ArtistID)
	} else if f.ArtistString != "" {
		values.Set("artist", f.ArtistString)
	}

	if f.TechniqueString != "" {
		values.Set("technique", f.TechniqueString)
	}

	if f.PeriodString != "" {
		values.Set("period", f.PeriodString)
	}

	if f.selectedVenue() != "" {
		values.Set("venue", f.selectedVenue())
	}

	if f.VenueQuery != "" {
		values.Set("venue_q", f.VenueQuery)
	}

	if f.YearFrom != "" {
		values.Set("year_from", f.YearFrom)
	}

	if f.YearTo != "" {
		values.Set("year_to", f.YearTo)
	}

	if f.View == "list" {
		values.Set("view", f.View)
	}

	if f.Sort != "" && f.Sort != sortTitle {
		values.Set("sort", f.Sort)
	}

	if f.SortDir == sortDesc {
		values.Set("dir", f.SortDir)
	}

	if f.Page != "" && f.Page != "1" {
		values.Set("page", f.Page)
	}
	if f.SchoolExpanded {
		values.Set("school_all", "1")
	}
	if f.FormExpanded {
		values.Set("form_all", "1")
	}

	return values
}

func buildFilters(values url.Values) *filters {
	yearFrom, yearTo := normalizeArtworkYearBounds(values.Get("year_from"), values.Get("year_to"))

	sort := cmp.Or(strings.TrimSpace(values.Get("sort")), sortTitle)
	_, validSort := artworkSortCriterionFor(sort)
	rawDir := strings.TrimSpace(values.Get("dir"))
	dir := cmp.Or(rawDir, sortAsc)
	if !validSort || (dir != sortAsc && dir != sortDesc) {
		sort = sortTitle
		dir = sortAsc
	}

	venue := strings.TrimSpace(values.Get("venue"))
	legacyLocation := strings.TrimSpace(values.Get("location"))
	venueConflict := venue != "" && legacyLocation != "" && venue != legacyLocation
	if venue == "" {
		venue = legacyLocation
	}

	f := &filters{
		Query:           cmp.Or(values.Get("q"), ""),
		Title:           cmp.Or(values.Get("title"), ""),
		SchoolValues:    normalizedRepeatedValues(values["art_school"]),
		ArtFormValues:   normalizedRepeatedValues(values["art_form"]),
		ArtTypeString:   cmp.Or(values.Get("art_type"), ""),
		ArtistString:    cmp.Or(values.Get("artist"), ""),
		ArtistID:        strings.TrimSpace(values.Get("artist_id")),
		TechniqueString: cmp.Or(values.Get("technique"), ""),
		PeriodString:    cmp.Or(values.Get("period"), ""),
		VenueString:     venue,
		VenueQuery:      strings.TrimSpace(values.Get("venue_q")),
		VenueConflict:   venueConflict,
		LocationString:  venue,
		YearFrom:        yearFrom,
		YearTo:          yearTo,
		View:            artworkSearchView(values.Get("view")),
		Sort:            sort,
		SortDir:         dir,
		Page:            cmp.Or(values.Get("page"), ""),
		SchoolExpanded:  values.Get("school_all") == "1",
		FormExpanded:    values.Get("form_all") == "1",
	}
	if len(f.SchoolValues) > 0 {
		f.SchoolString = f.SchoolValues[0]
	}
	if len(f.ArtFormValues) > 0 {
		f.ArtFormString = f.ArtFormValues[0]
	}
	if f.ArtistID != "" {
		f.ArtistString = ""
	}

	return f
}

func normalizedRepeatedValues(values []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized
}

func (f *filters) schoolValues() []string {
	if len(f.SchoolValues) > 0 {
		return f.SchoolValues
	}
	if f.SchoolString != "" {
		return []string{f.SchoolString}
	}
	return nil
}

func (f *filters) formValues() []string {
	if len(f.ArtFormValues) > 0 {
		return f.ArtFormValues
	}
	if f.ArtFormString != "" {
		return []string{f.ArtFormString}
	}
	return nil
}

func anyFacetFilter(field string, parameter string, values []string, params dbx.Params) string {
	conditions := make([]string, 0, len(values))
	for index, value := range values {
		key := fmt.Sprintf("%s_%d", parameter, index)
		conditions = append(conditions, field+" = {:"+key+"}")
		params[key] = value
	}
	return "(" + strings.Join(conditions, " || ") + ")"
}

func (f *filters) selectedVenue() string {
	if f.VenueString != "" {
		return f.VenueString
	}

	return f.LocationString
}

func artworkSearchView(value string) string {
	if value == "list" {
		return "list"
	}

	return "grid"
}

// normalizeArtworkYearBounds parses the year request state exactly once.
// Malformed values fall back to the range defaults, each bound clamps to the
// 200–1900 span, and reversed bounds are swapped so the canonical state always
// reads from <= to. Bounds equal to their defaults are returned empty so the
// canonical URL omits them, matching the result predicate and facet summary.
func normalizeArtworkYearBounds(rawFrom string, rawTo string) (string, string) {
	from := parseArtworkYearBound(rawFrom, artworkYearMin)
	to := parseArtworkYearBound(rawTo, artworkYearMax)

	from = clampArtworkYear(from)
	to = clampArtworkYear(to)

	if from > to {
		from, to = to, from
	}

	return artworkYearBoundString(from, artworkYearMin), artworkYearBoundString(to, artworkYearMax)
}

func parseArtworkYearBound(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}

	return value
}

func clampArtworkYear(value int) int {
	if value < artworkYearMin {
		return artworkYearMin
	}
	if value > artworkYearMax {
		return artworkYearMax
	}

	return value
}

func artworkYearBoundString(value int, defaultValue int) string {
	if value == defaultValue {
		return ""
	}

	return strconv.Itoa(value)
}
