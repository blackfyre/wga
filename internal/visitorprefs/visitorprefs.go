// Package visitorprefs reads the visitor's remembered presentation
// preferences: the wga_aw_prefs cookie that resources/js/search-prefs.ts
// writes, honoured only while the CookieConsent record accepts the optional
// preferences category.
package visitorprefs

import (
	"context"
	"encoding/json"
	"net/http"
	neturl "net/url"
	"slices"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// CookieName is the first-party preference cookie. Its value is the
	// URL-encoded JSON object also kept in localStorage["wga-aw-prefs"].
	CookieName = "wga_aw_prefs"
	// ConsentCookieName is the CookieConsent record.
	ConsentCookieName = "cc_cookie"
	// PreferencesCategory is the optional consent category that permits
	// remembered preferences.
	PreferencesCategory = "preferences"

	// ViewGrid and ViewList are the listing result views. Grid is the default.
	ViewGrid = "grid"
	ViewList = "list"
)

// ConsentGranted reports whether the CookieConsent record accepts the
// optional preferences category.
func ConsentGranted(r *http.Request) bool {
	cookie, err := r.Cookie(ConsentCookieName)
	if err != nil {
		return false
	}
	raw, err := neturl.PathUnescape(cookie.Value)
	if err != nil {
		return false
	}
	var record struct {
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return false
	}
	return slices.Contains(record.Categories, PreferencesCategory)
}

// HasCookie reports whether the request carries a remembered preference
// cookie at all, valid or not. Responses to such requests are never shared, so
// privacy does not depend on the cookie decoding or on current consent.
func HasCookie(r *http.Request) bool {
	_, err := r.Cookie(CookieName)
	return err == nil
}

// Fields decodes the remembered preference object. It returns false when the
// cookie is absent or undecodable, or when preference consent is missing.
func Fields(r *http.Request) (map[string]json.RawMessage, bool) {
	if !ConsentGranted(r) {
		return nil, false
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return nil, false
	}
	raw, err := neturl.PathUnescape(cookie.Value)
	if err != nil {
		return nil, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil || fields == nil {
		return nil, false
	}
	return fields, true
}

// RememberedView returns the valid remembered listing view, or "" when none
// is usable.
func RememberedView(r *http.Request) string {
	fields, ok := Fields(r)
	if !ok {
		return ""
	}
	return ParseView(fields["view"])
}

// ParseView returns a stored view value when it is "grid" or "list".
func ParseView(raw json.RawMessage) string {
	var view string
	if err := json.Unmarshal(raw, &view); err != nil {
		return ""
	}
	if view == ViewGrid || view == ViewList {
		return view
	}
	return ""
}

// FallbackView is the view a listing renders when its request carries no
// view parameter: the remembered view, or the grid.
func FallbackView(r *http.Request) string {
	if view := RememberedView(r); view != "" {
		return view
	}
	return ViewGrid
}

// ExplicitView normalises an explicit view parameter: list stays list and any
// other value is the grid.
func ExplicitView(value string) string {
	if value == ViewList {
		return ViewList
	}
	return ViewGrid
}

// EmitView reports whether a listing URL must carry its view parameter. A
// missing parameter means "use the fallback", so the grid is written
// explicitly only when the visitor's fallback is list; list is always written.
func EmitView(view string, fallback string) bool {
	if view == ViewList {
		return true
	}
	return view == ViewGrid && fallback == ViewList
}

// MarkResponse records that a listing response depends on the preference
// cookie: it always varies on Cookie, and a request carrying the cookie is
// never served from or stored in a shared cache.
func MarkResponse(c *core.RequestEvent) {
	c.Response.Header().Add("Vary", "Cookie")
	if HasCookie(c.Request) {
		utils.KeepResponsePrivate(c)
	}
}

type fallbackViewKey struct{}

// WithFallbackView carries the visitor's fallback view to listing URL
// builders, so they know whether a grid URL must write its view parameter.
func WithFallbackView(ctx context.Context, view string) context.Context {
	return context.WithValue(ctx, fallbackViewKey{}, view)
}

// FallbackViewFrom returns the fallback view carried by ctx, or "" when none
// was set, which URL builders treat as the grid.
func FallbackViewFrom(ctx context.Context) string {
	view, _ := ctx.Value(fallbackViewKey{}).(string)
	return view
}
