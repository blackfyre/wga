package search

import "github.com/labstack/echo/v5"

type filters struct {
	Query string
}

func buildFilters(c echo.Context) *filters {
	return &filters{
		Query: c.QueryParamDefault("q", ""),
	}
}
