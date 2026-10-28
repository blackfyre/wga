package artworks

import (
	"encoding/json"
	"net/http"
	neturl "net/url"

	"github.com/blackfyre/wga/internal/visitorprefs"
)

// searchPrefsCookieName is the first-party cookie the browser preference
// module (resources/js/search-prefs.ts) writes. Its value is the URL-encoded
// JSON object also kept in localStorage["wga-aw-prefs"].
const searchPrefsCookieName = visitorprefs.CookieName

// consentCookieName is the CookieConsent record. Remembered preferences are
// optional storage, honoured only while it accepts the preferences category.
const consentCookieName = visitorprefs.ConsentCookieName

// searchPrefs are the visitor's remembered artwork search presentation
// choices. They are presentation state, never part of the query: sort and
// direction only supply defaults for a bare /artworks visit, and the view is
// the shared listing view used by any request without a view parameter. Empty
// fields were absent or invalid in the stored value.
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
	fields, ok := visitorprefs.Fields(r)
	if !ok {
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
	prefs.View = visitorprefs.ParseView(fields["view"])
	var actions bool
	if err := json.Unmarshal(fields["actions"], &actions); err == nil {
		prefs.Actions = actions
	}

	return prefs
}

// fallbackView is the view a request without a view parameter renders.
func (p searchPrefs) fallbackView() string {
	if p.View != "" {
		return p.View
	}
	return visitorprefs.ViewGrid
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
// Only a bare GET /artworks takes the remembered sort and direction: any
// explicit query parameter, and every /artworks/results fragment request, keeps
// the sort and direction it addresses so shared and paginated URLs never
// change meaning. The remembered view is the shared listing view, so it fills
// in for any page or fragment request that carries no view parameter.
func searchRequestValues(u *neturl.URL, prefs searchPrefs) neturl.Values {
	values := u.Query()
	if prefs.View != "" && !values.Has("view") {
		values.Set("view", prefs.View)
	}
	if u.Path != "/artworks" || u.RawQuery != "" {
		return values
	}
	if prefs.Sort != "" {
		values.Set("sort", prefs.Sort)
	}
	if prefs.Dir != "" {
		values.Set("dir", prefs.Dir)
	}
	return values
}
