package search

import (
	"blackfyre.ninja/wga/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type filters struct {
	Q       string
	School  string
	ArtForm string
	ArtType string
}

func (f *filters) AnyFilterActive() bool {
	return f.Q != "" || f.School != "" || f.ArtForm != "" || f.ArtType != ""
}

func (f *filters) FingerPrint() string {
	return f.Q + ":" + f.School + ":" + f.ArtForm + ":" + f.ArtType
}

func (f *filters) BuildFilter() (string, dbx.Params) {
	filterString := "published = true"
	params := dbx.Params{}

	if f.Q != "" {
		filterString = filterString + " && name ~ {:q}"
		params["q"] = f.Q
	}

	if f.School != "" {
		filterString = filterString + " && school = {:art_school}"
		params["art_school"] = f.School
	}

	if f.ArtForm != "" {
		filterString = filterString + " && form = {:art_form}"
		params["art_form"] = f.ArtForm
	}

	if f.ArtType != "" {
		filterString = filterString + " && art_type = {:art_type}"
		params["art_type"] = f.ArtType
	}

	return filterString, params
}

func buildFilters(app *pocketbase.PocketBase, c echo.Context) *filters {
	f := &filters{
		Q:       c.QueryParamDefault("q", ""),
		School:  c.QueryParamDefault("art_school", ""),
		ArtForm: c.QueryParamDefault("art_form", ""),
		ArtType: c.QueryParamDefault("art_type", ""),
	}

	if f.School == "na" {
		f.School = ""
	} else {
		if app.Cache().Has("search:schools:" + f.School) {
			f.School = app.Cache().Get("search:schools:" + f.School).(string)
		} else {
			r, err := models.GetSchoolBySlug(app.Dao(), f.School)

			if err != nil {
				f.School = ""
			} else {
				f.School = r.GetId()
			}
		}
	}

	if f.ArtForm == "na" {
		f.ArtForm = ""
	} else {
		if app.Cache().Has("search:forms:" + f.ArtForm) {
			f.ArtForm = app.Cache().Get("search:forms:" + f.ArtForm).(string)
		} else {
			r, err := models.GetArtFormBySlug(app.Dao(), f.ArtForm)

			if err != nil {
				f.ArtForm = ""
			} else {
				f.ArtForm = r.GetId()
			}
		}
	}

	if f.ArtType == "na" {
		f.ArtType = ""
	} else {
		if app.Cache().Has("search:types:" + f.ArtType) {
			f.ArtType = app.Cache().Get("search:types:" + f.ArtType).(string)
		} else {
			r, err := models.GetArtTypeBySlug(app.Dao(), f.ArtType)

			if err != nil {
				f.ArtType = ""
			} else {
				f.ArtType = r.GetId()
			}
		}
	}

	return f
}
