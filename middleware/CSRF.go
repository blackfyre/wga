package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

var CSRF_IDENTIFIER = "csrf_token"
var CSRF_HMAC_SECRET = os.Getenv("CSRF_HMAC_SECRET")

func CSRF(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get CSRF token from the session
		csrfCookie, err := c.Cookie(CSRF_IDENTIFIER)
		if err != nil {
			// then the cookie is not preset
			// e.g. on first request of user session, or after cookie expiration
			csrfCookie = generateCSRFCookie() // generates a new token
			// only set the cookie when a new token has been generated
			c.SetCookie(csrfCookie)
		}

		// Store the CSRF token in the request context to that it can be used in <input type="hidden" value={ middleware.CSRF_IDENTIFIER }>
		c.Set(CSRF_IDENTIFIER, csrfCookie.Value)

		// On POST requests, check for a CSRF token
		if isPostRequestAndShouldCSRFTokenBeEvaluated(c) {
			formToken := c.FormValue(CSRF_IDENTIFIER)
			if formToken == "" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "CSRF token is missing",
				})
			}
			if formToken != csrfCookie.Value {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "CSRF token is invalid",
				})
			}
		}

		// Call the next handler
		return next(c)
	}
}

func generateCSRFCookie() *http.Cookie {
	cookie := new(http.Cookie)
	cookie.Name = CSRF_IDENTIFIER
	cookie.Value = generateCSRFToken()
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteStrictMode
	return cookie
}

func generateCSRFToken() string {
	secretKey := []byte(CSRF_HMAC_SECRET)
	// message should be something unpredictable - like a request id (e.g. by echo requeset id middleware)
	message := []byte(time.Now().String())
	hash := hmac.New(sha256.New, secretKey)
	hash.Write(message)
	return hex.EncodeToString(hash.Sum(nil))
}

func isPostRequestAndShouldCSRFTokenBeEvaluated(c echo.Context) bool {
	return c.Request().Method == http.MethodPost &&
		!strings.HasPrefix(c.Path(), "/_") &&
		(strings.Contains(c.Request().Header.Get("Content-Type"), "application/x-www-form-urlencoded") || strings.Contains(c.Request().Header.Get("Content-Type"), "multipart/form-data"))
}
