package dto

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// PaletteOption describes one of the eleven reference palettes. Paper and Ink
// are the pair's split-swatch colours.
type PaletteOption struct {
	Key      string
	Label    string
	Group    string
	Desc     string
	Paper    string
	Ink      string
	DarkOnly bool
}

// DefaultPaletteKey is the palette served when no explicit palette is stored.
const DefaultPaletteKey = "bone"

// PaletteOptions is the single source of truth for the eleven reference
// palettes, in presentation order. The first entry is the default.
var PaletteOptions = []PaletteOption{
	{Key: "bone", Label: "BONE", Group: "THIS ARCHIVE", Desc: "Bone white and deep navy — the default", Paper: "#f4f2ed", Ink: "#003366"},
	{Key: "classic", Label: "CLASSIC", Group: "THIS ARCHIVE", Desc: "The blue and orange of the original wga.hu", Paper: "#f0f6ff", Ink: "#003366"},
	{Key: "verdigris", Label: "VERDIGRIS", Group: "THIS ARCHIVE", Desc: "Oxidised copper on grey-green paper", Paper: "#edf1ec", Ink: "#1f5e55"},
	{Key: "gothic", Label: "GOTHIC", Group: "FROM THE COLLECTION", Desc: "Gold ground, ultramarine, vermilion", Paper: "#f7efdc", Ink: "#24418f"},
	{Key: "renaissance", Label: "RENAISSANCE", Group: "FROM THE COLLECTION", Desc: "Fresco plaster, sanguine chalk, azurite", Paper: "#f3ede1", Ink: "#8a3324"},
	{Key: "baroque", Label: "BAROQUE", Group: "FROM THE COLLECTION", Desc: "Tenebrism — dark only", Paper: "#17110c", Ink: "#e0b450", DarkOnly: true},
	{Key: "rococo", Label: "ROCOCO", Group: "FROM THE COLLECTION", Desc: "Pale rose, celadon and gilt", Paper: "#f7eff2", Ink: "#9b2f5f"},
	{Key: "classical", Label: "CLASSICAL", Group: "FROM THE COLLECTION", Desc: "Wedgwood ground, marble, Pompeian red", Paper: "#edeff1", Ink: "#3a5a7a"},
	{Key: "impressionist", Label: "IMPRESSIONIST", Group: "FROM THE COLLECTION", Desc: "Cerulean and lilac shadow on a pale field", Paper: "#f4f1f5", Ink: "#1a5f80"},
	{Key: "catppuccin", Label: "CATPPUCCIN", Group: "BORROWED", Desc: "Latte by day, Mocha by night", Paper: "#eff1f5", Ink: "#1e66f5"},
	{Key: "tokyo", Label: "TOKYO NIGHT", Group: "BORROWED", Desc: "Dark only — no light half", Paper: "#1a1b26", Ink: "#7aa2f7", DarkOnly: true},
}

// Preferences is the validated server projection of a visitor's stored
// appearance choices. Palette is always a known key (defaults to bone). Scheme
// is "light" or "dark", or "" when the visitor has not chosen (follow the
// operating system).
type Preferences struct {
	Palette string
	Scheme  string
	Bionic  bool
}

// PaletteByKey returns the option for key and whether it matched. An unknown
// key returns the default bone option with ok=false.
func PaletteByKey(key string) (PaletteOption, bool) {
	for _, option := range PaletteOptions {
		if option.Key == key {
			return option, true
		}
	}
	return PaletteOptions[0], false
}

// NormalizePalette returns key when it is a known palette, else "".
func NormalizePalette(key string) string {
	if _, ok := PaletteByKey(key); ok {
		return key
	}
	return ""
}

// NormalizeScheme maps a stored scheme value to "light" or "dark". It accepts
// the legacy "wga_light" and "wga_dark" values used by earlier builds, and
// returns "" for anything else (unset).
func NormalizeScheme(value string) string {
	switch value {
	case "light", "wga_light":
		return "light"
	case "dark", "wga_dark":
		return "dark"
	default:
		return ""
	}
}

// DarkOnlyPalette reports whether key is a palette with no light build.
func DarkOnlyPalette(key string) bool {
	option, _ := PaletteByKey(key)
	return option.DarkOnly
}

// PaletteLabel returns the display label for key, defaulting to bone.
func PaletteLabel(key string) string {
	option, _ := PaletteByKey(key)
	return option.Label
}

// PaletteSwatchStyle returns the inline split paper/ink swatch background for
// key, defaulting to bone.
func PaletteSwatchStyle(key string) string {
	option, _ := PaletteByKey(key)
	return "background:linear-gradient(135deg," + option.Paper + " 0 50%," + option.Ink + " 50% 100%)"
}

// PaletteGroups returns the distinct provenance group headings in first-seen
// order, so the flat options list stays the single source of truth.
func PaletteGroups() []string {
	var groups []string
	for _, option := range PaletteOptions {
		if len(groups) == 0 || groups[len(groups)-1] != option.Group {
			groups = append(groups, option.Group)
		}
	}
	return groups
}

// PreferencesSummary states the active palette, scheme, and reading mode as a
// compact cookie-derived line for the footer trigger.
func PreferencesSummary(prefs Preferences) string {
	summary := PaletteLabel(prefs.Palette) + " · "
	if prefs.Scheme == "dark" || DarkOnlyPalette(prefs.Palette) {
		summary += "DARK"
	} else {
		summary += "LIGHT"
	}
	if prefs.Bionic {
		summary += " · BIONIC"
	}
	return summary
}

// PaletteTableJSON returns the known palette keys and whether each is dark-only
// for the inline pre-paint resolver.
func PaletteTableJSON() string {
	table := make(map[string]bool, len(PaletteOptions))
	for _, option := range PaletteOptions {
		table[option.Key] = option.DarkOnly
	}

	bytes, err := json.Marshal(table)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

// resolverScriptBody is the inline pre-paint resolver. The two %s placeholders
// receive the palette table and the quoted default palette key.
const resolverScriptBody = `(function () {
		var PALETTES = %s;
	var DEFAULT_PALETTE = %s;

	function readLocalStorage(key) {
		try {
			return window.localStorage.getItem(key);
		} catch (error) {
			return null;
		}
	}

	function readCookie(name) {
		var prefix = name + "=";
		var cookies = document.cookie ? document.cookie.split("; ") : [];
		for (var i = 0; i < cookies.length; i++) {
			var cookie = cookies[i];
			if (cookie.indexOf(prefix) === 0) {
				return cookie.slice(prefix.length);
			}
		}
		return null;
	}

	function normalizeScheme(value) {
		if (value === "light" || value === "wga_light") {
			return "light";
		}
		if (value === "dark" || value === "wga_dark") {
			return "dark";
		}
		return null;
	}

	var palette = readLocalStorage("wga-palette");
	if (!Object.prototype.hasOwnProperty.call(PALETTES, palette)) {
		palette = readCookie("wga_palette");
	}
	if (!Object.prototype.hasOwnProperty.call(PALETTES, palette)) {
		palette = DEFAULT_PALETTE;
	}

	var scheme = normalizeScheme(readLocalStorage("wga-theme"));
	if (!scheme) {
		scheme = normalizeScheme(readCookie("wga_theme"));
	}
	if (!scheme) {
		scheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
	}

	document.documentElement.dataset.palette = palette;
	document.documentElement.dataset.theme = PALETTES[palette] ? "dark" : scheme;
}());`

// ThemeResolverScript returns the complete inline pre-paint palette/scheme head
// script, derived from PaletteOptions so the palette table cannot drift from the
// rendered preferences panel.
func ThemeResolverScript() string {
	return "<script>\n" + fmt.Sprintf(resolverScriptBody, PaletteTableJSON(), strconv.Quote(DefaultPaletteKey)) + "\n</script>"
}
