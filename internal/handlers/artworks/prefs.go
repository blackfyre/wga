package artworks

import (
	"encoding/json"
	"net/http"
	neturl "net/url"
	"slices"
)

// searchPrefsCookieName is the first-party cookie the browser preference
// module (resources/js/search-prefs.ts) writes. Its value is the URL-encoded
// JSON object also kept in localStorage["wga-aw-prefs"].
const searchPrefsCookieName = "wga_aw_prefs"

// consentCookieName is the CookieConsent record. Remembered preferences are
// optional storage, honoured only while it accepts preferencesCategory.
const (
	consentCookieName   = "cc_cookie"
	preferencesCategory = "preferences"
)

// searchPrefs are the visitor's remembered artwork search presentation
// choices. They are presentation state, never part of the query: they only
// supply defaults for a bare /artworks visit. Empty fields were absent or
// invalid in the stored value.
type searchPrefs struct {
	// Present reports that a decodable preference object was stored, which
	// also means the browser module has run for this visitor before.
	Present bool
	Sort    string
	Dir     string
	View    string
	Actions bool
}

// storedSearchPrefs decodes the remembered preferences. Each field is
// validated independently, so one bad value does not discard the others; an
// absent or undecodable cookie, or one left over without current preference
// consent, yields the zero value.
func storedSearchPrefs(r *http.Request) searchPrefs {
	if !preferenceConsentGranted(r) {
		return searchPrefs{}
	}
	cookie, err := r.Cookie(searchPrefsCookieName)
	if err != nil {
		return searchPrefs{}
	}
	raw, err := neturl.PathUnescape(cookie.Value)
	if err != nil {
		return searchPrefs{}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil || fields == nil {
		return searchPrefs{}
	}

	prefs := searchPrefs{Present: true}
	if sort := jsonString(fields["sort"]); sort != "" {
		if _, ok := artworkSortCriterionFor(sort); ok {
			prefs.Sort = sort
		}
	}
	if dir := jsonString(fields["dir"]); dir == sortAsc || dir == sortDesc {
		prefs.Dir = dir
	}
	if view := jsonString(fields["view"]); view == "grid" || view == "list" {
		prefs.View = view
	}
	var actions bool
	if err := json.Unmarshal(fields["actions"], &actions); err == nil {
		prefs.Actions = actions
	}

	return prefs
}

// preferenceConsentGranted reports whether the CookieConsent record accepts
// the optional preferences category.
func preferenceConsentGranted(r *http.Request) bool {
	cookie, err := r.Cookie(consentCookieName)
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
	return slices.Contains(record.Categories, preferencesCategory)
}

func jsonString(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

// ActionsAttribute is the value for <html data-aw-actions>, or empty when no
// preference was stored so the browser module decides on its first run.
func (p searchPrefs) ActionsAttribute() string {
	if !p.Present {
		return ""
	}
	if p.Actions {
		return "on"
	}
	return "off"
}

// searchRequestValues returns the query state for an artwork search request.
// Only a bare GET /artworks takes the remembered sort, direction, and view:
// any explicit query parameter, and every /artworks/results fragment request,
// is used exactly as addressed so shared and paginated URLs never change
// meaning.
func searchRequestValues(u *neturl.URL, prefs searchPrefs) neturl.Values {
	values := u.Query()
	if u.Path != "/artworks" || u.RawQuery != "" {
		return values
	}
	if prefs.Sort != "" {
		values.Set("sort", prefs.Sort)
	}
	if prefs.Dir != "" {
		values.Set("dir", prefs.Dir)
	}
	if prefs.View != "" {
		values.Set("view", prefs.View)
	}
	return values
}
