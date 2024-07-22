package utils

import (
    "github.com/labstack/echo/v5"
    "net/http"
)

func IsHtmxRequestMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        if c.Request().Header.Get("HX-Request") != "true" {
            return ServerFaultError(c)
        }

        return next(c) // proceed with the request chain
    }
}
