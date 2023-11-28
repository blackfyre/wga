package artworks

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type filters struct {
	Title         string
	SchoolString  string
	ArtFormString string
	ArtTypeString string
	ArtistString  string
}

func (f *filters) AnyFilterActive() bool {
	return f.Title != "" || f.SchoolString != "" || f.ArtFormString != "" || f.ArtTypeString != "" || f.ArtistString != ""
}

func (f *filters) FingerPrint() string {
	return f.Title + ":" + f.SchoolString + ":" + f.ArtFormString + ":" + f.ArtTypeString + ":" + f.ArtistString
}

func (f *filters) BuildFilter() (string, dbx.Params) {
	filterString := "published = true"
	params := dbx.Params{}

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

	return filterString, params
}

func buildFilters(app *pocketbase.PocketBase, c echo.Context) *filters {
	f := &filters{
		Title:         c.QueryParamDefault("title", ""),
		SchoolString:  c.QueryParamDefault("art_school", ""),
		ArtFormString: c.QueryParamDefault("art_form", ""),
		ArtTypeString: c.QueryParamDefault("art_type", ""),
		ArtistString:  c.QueryParamDefault("artist", ""),
	}

	return f
}
