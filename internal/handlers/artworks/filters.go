package artworks

import (
	"cmp"
	"net/url"
	"strings"

	"github.com/pocketbase/dbx"
)

type filters struct {
	Query           string
	Title           string
	SchoolString    string
	ArtFormString   string
	ArtTypeString   string
	ArtistString    string
	TechniqueString string
	PeriodString    string
	LocationString  string
	YearFrom        string
	YearTo          string
	View            string
	Sort            string
	SortDir         string
	Page            string
}

// AnyFilterActive checks if any filter is active. Sort and view are presentation
// state, not filters, so they are intentionally excluded.
func (f *filters) AnyFilterActive() bool {
	return f.Query != "" || f.Title != "" || f.SchoolString != "" || f.ArtFormString != "" || f.ArtTypeString != "" || f.ArtistString != "" || f.TechniqueString != "" || f.PeriodString != "" || f.LocationString != "" || f.YearFrom != "" || f.YearTo != ""
}

// FingerPrint returns a unique fingerprint string based on the filter values.
func (f *filters) FingerPrint() string {
	return f.Query + ":" + f.Title + ":" + f.SchoolString + ":" + f.ArtFormString + ":" + f.ArtTypeString + ":" + f.ArtistString + ":" + f.TechniqueString + ":" + f.PeriodString + ":" + f.LocationString + ":" + f.YearFrom + ":" + f.YearTo + ":" + f.View + ":" + f.Sort + ":" + f.SortDir + ":" + f.Page
}

// BuildFilter builds the PocketBase filter string and parameters for the
// relation and scalar filters.
func (f *filters) BuildFilter() (string, dbx.Params) {
	filterString := "published = true && author:length > 0"
	params := dbx.Params{}

	if f.Query != "" {
		filterString = filterString + " && (title ~ {:query} || author.name ~ {:query})"
		params["query"] = f.Query
	}

	if f.Title != "" {
		filterString = filterString + " && title ~ {:title}"
		params["title"] = f.Title
	}

	if f.SchoolString != "" {
		filterString = filterString + " && school.slug = {:art_school}"
		params["art_school"] = f.SchoolString
	}

	if f.ArtFormString != "" {
		filterString = filterString + " && form.slug = {:art_form}"
		params["art_form"] = f.ArtFormString
	}

	if f.ArtTypeString != "" {
		filterString = filterString + " && type.slug = {:art_type}"
		params["art_type"] = f.ArtTypeString
	}

	if f.ArtistString != "" {
		filterString = filterString + " && author.name ~ {:artist}"
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

	if f.LocationString != "" {
		filterString = filterString + " && current_location_id = {:location}"
		params["location"] = f.LocationString
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

	if f.SchoolString != "" {
		values.Set("art_school", f.SchoolString)
	}

	if f.ArtFormString != "" {
		values.Set("art_form", f.ArtFormString)
	}

	if f.ArtTypeString != "" {
		values.Set("art_type", f.ArtTypeString)
	}

	if f.ArtistString != "" {
		values.Set("artist", f.ArtistString)
	}

	if f.TechniqueString != "" {
		values.Set("technique", f.TechniqueString)
	}

	if f.PeriodString != "" {
		values.Set("period", f.PeriodString)
	}

	if f.LocationString != "" {
		values.Set("location", f.LocationString)
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

	if f.Sort != "" && f.Sort != sortCatalogue {
		values.Set("sort", f.Sort)
	}

	if f.SortDir == sortDesc {
		values.Set("dir", f.SortDir)
	}

	if f.Page != "" && f.Page != "1" {
		values.Set("page", f.Page)
	}

	return values
}

func buildFilters(values url.Values) *filters {
	yearFrom := cmp.Or(values.Get("year_from"), "")
	if yearFrom == "200" {
		yearFrom = ""
	}

	yearTo := cmp.Or(values.Get("year_to"), "")
	if yearTo == "1900" {
		yearTo = ""
	}

	sort := cmp.Or(strings.TrimSpace(values.Get("sort")), "")
	if _, ok := artworkSortCriterionFor(sort); !ok {
		sort = sortCatalogue
	}

	dir := cmp.Or(strings.TrimSpace(values.Get("dir")), "")
	if dir != sortDesc {
		dir = sortAsc
	}

	f := &filters{
		Query:           cmp.Or(values.Get("q"), ""),
		Title:           cmp.Or(values.Get("title"), ""),
		SchoolString:    cmp.Or(values.Get("art_school"), ""),
		ArtFormString:   cmp.Or(values.Get("art_form"), ""),
		ArtTypeString:   cmp.Or(values.Get("art_type"), ""),
		ArtistString:    cmp.Or(values.Get("artist"), ""),
		TechniqueString: cmp.Or(values.Get("technique"), ""),
		PeriodString:    cmp.Or(values.Get("period"), ""),
		LocationString:  cmp.Or(values.Get("location"), ""),
		YearFrom:        yearFrom,
		YearTo:          yearTo,
		View:            artworkSearchView(values.Get("view")),
		Sort:            sort,
		SortDir:         dir,
		Page:            cmp.Or(values.Get("page"), ""),
	}

	return f
}

func artworkSearchView(value string) string {
	if value == "list" {
		return "list"
	}

	return "grid"
}
