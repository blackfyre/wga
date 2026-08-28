package utils

import (
	"context"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

// preferencesContextKey is a private typed key for the validated preference
// projection stored by ContextFromRequest. It is intentionally distinct from
// the generic ContextKey so the structured projection cannot collide with an
// ordinary string context value.
type preferencesContextKey struct{}

// readPreferences projects the visitor's stored palette, scheme, and bionic
// choices from request cookies, validating every value before it is stored. An
// absent or unrecognised palette falls back to the default bone palette; an
// absent or unrecognised scheme stays unset so the page follows the operating
// system.
func readPreferences(request *http.Request) dto.Preferences {
	prefs := dto.Preferences{Palette: dto.DefaultPaletteKey}

	if request == nil {
		return prefs
	}

	if cookie, err := request.Cookie("wga_palette"); err == nil {
		if palette := dto.NormalizePalette(cookie.Value); palette != "" {
			prefs.Palette = palette
		}
	}

	if cookie, err := request.Cookie("wga_theme"); err == nil {
		prefs.Scheme = dto.NormalizeScheme(cookie.Value)
	}

	if cookie, err := request.Cookie("wga_bionic"); err == nil && cookie.Value == "on" {
		prefs.Bionic = true
	}

	return prefs
}

// GetPreferences returns the validated preference projection stored in the
// context, or the zero projection when none was stored.
func GetPreferences(c context.Context) dto.Preferences {
	prefs, ok := c.Value(preferencesContextKey{}).(dto.Preferences)
	if !ok {
		return dto.Preferences{}
	}
	return prefs
}

// GetBionicReading reports whether the visitor enabled bionic reading.
func GetBionicReading(c context.Context) bool {
	return GetPreferences(c).Bionic
}
