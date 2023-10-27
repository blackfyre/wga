package search

import "github.com/labstack/echo/v5"

type filters struct {
	Query   string
	School  string
	ArtForm string
	ArtType string
}

func buildFilters(c echo.Context) *filters {
	f := &filters{
		Query:   c.QueryParamDefault("q", ""),
		School:  c.QueryParamDefault("art_school", ""),
		ArtForm: c.QueryParamDefault("art_form", ""),
		ArtType: c.QueryParamDefault("art_type", ""),
	}

	if f.School == "na" {
		f.School = ""
	}

	if f.ArtForm == "na" {
		f.ArtForm = ""
	}

	if f.ArtType == "na" {
		f.ArtType = ""
	}

	return f
}
