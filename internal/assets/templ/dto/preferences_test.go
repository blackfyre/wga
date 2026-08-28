package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPaletteOptionsCountAndDefault(t *testing.T) {
	if len(PaletteOptions) != 11 {
		t.Fatalf("got %d palettes, want 11", len(PaletteOptions))
	}
	if PaletteOptions[0].Key != DefaultPaletteKey {
		t.Fatalf("first palette = %q, want default %q", PaletteOptions[0].Key, DefaultPaletteKey)
	}
	if DefaultPaletteKey != "bone" {
		t.Fatalf("default palette = %q, want bone", DefaultPaletteKey)
	}
}

func TestPaletteOptionsCompleteAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, option := range PaletteOptions {
		if seen[option.Key] {
			t.Fatalf("duplicate palette key %q", option.Key)
		}
		seen[option.Key] = true
		if option.Label == "" || option.Group == "" || option.Desc == "" || option.Paper == "" || option.Ink == "" || option.Theme == "" {
			t.Fatalf("palette %q has an empty presentation field", option.Key)
		}
	}
}

func TestThemeTableMatchesPaletteThemeMapping(t *testing.T) {
	want := map[string][2]string{
		"bone":          {"wga-rams", "wga-rams-dark"},
		"classic":       {"wga-classic", "wga-classic-dark"},
		"verdigris":     {"wga-verdigris", "wga-verdigris-dark"},
		"gothic":        {"wga-gothic", "wga-gothic-dark"},
		"renaissance":   {"wga-renaissance", "wga-renaissance-dark"},
		"baroque":       {"wga-baroque", "wga-baroque"},
		"rococo":        {"wga-rococo", "wga-rococo-dark"},
		"classical":     {"wga-classical", "wga-classical-dark"},
		"impressionist": {"wga-impressionist", "wga-impressionist-dark"},
		"catppuccin":    {"wga-catppuccin", "wga-catppuccin-dark"},
		"tokyo":         {"wga-tokyo", "wga-tokyo"},
	}

	var got map[string][2]string
	if err := json.Unmarshal([]byte(ThemeTableJSON()), &got); err != nil {
		t.Fatalf("unmarshal theme table: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("theme table has %d entries, want %d", len(got), len(want))
	}
	for key, pair := range want {
		if got[key] != pair {
			t.Errorf("theme table[%q] = %v, want %v", key, got[key], pair)
		}
	}
}

func TestThemeFor(t *testing.T) {
	tests := []struct {
		name    string
		palette string
		scheme  string
		want    string
	}{
		{name: "bone light", palette: "bone", scheme: "light", want: "wga-rams"},
		{name: "bone dark", palette: "bone", scheme: "dark", want: "wga-rams-dark"},
		{name: "classic light", palette: "classic", scheme: "light", want: "wga-classic"},
		{name: "classic dark", palette: "classic", scheme: "dark", want: "wga-classic-dark"},
		{name: "baroque light resolves dark", palette: "baroque", scheme: "light", want: "wga-baroque"},
		{name: "baroque dark", palette: "baroque", scheme: "dark", want: "wga-baroque"},
		{name: "tokyo light resolves dark", palette: "tokyo", scheme: "light", want: "wga-tokyo"},
		{name: "unknown palette defaults bone", palette: "neon", scheme: "dark", want: "wga-rams-dark"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ThemeFor(test.palette, test.scheme); got != test.want {
				t.Fatalf("ThemeFor(%q, %q) = %q, want %q", test.palette, test.scheme, got, test.want)
			}
		})
	}
}

func TestNormalizePalette(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "bone", want: "bone"},
		{input: "tokyo", want: "tokyo"},
		{input: "neon", want: ""},
		{input: "", want: ""},
	}
	for _, test := range tests {
		if got := NormalizePalette(test.input); got != test.want {
			t.Fatalf("NormalizePalette(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestNormalizeScheme(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "light", want: "light"},
		{input: "dark", want: "dark"},
		{input: "wga_light", want: "light"},
		{input: "wga_dark", want: "dark"},
		{input: "sepia", want: ""},
		{input: "", want: ""},
	}
	for _, test := range tests {
		if got := NormalizeScheme(test.input); got != test.want {
			t.Fatalf("NormalizeScheme(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestDarkOnlyPalette(t *testing.T) {
	if !DarkOnlyPalette("baroque") || !DarkOnlyPalette("tokyo") {
		t.Fatal("baroque and tokyo must be dark-only")
	}
	for _, option := range PaletteOptions {
		if option.DarkOnly != DarkOnlyPalette(option.Key) {
			t.Fatalf("DarkOnlyPalette(%q) = %t, want %t", option.Key, DarkOnlyPalette(option.Key), option.DarkOnly)
		}
	}
	if DarkOnlyPalette("bone") {
		t.Fatal("bone must not be dark-only")
	}
}

func TestPreferencesSummary(t *testing.T) {
	tests := []struct {
		name  string
		prefs Preferences
		want  string
	}{
		{name: "default", prefs: Preferences{}, want: "BONE · LIGHT"},
		{name: "dark scheme", prefs: Preferences{Palette: "bone", Scheme: "dark"}, want: "BONE · DARK"},
		{name: "palette and bionic", prefs: Preferences{Palette: "verdigris", Scheme: "light", Bionic: true}, want: "VERDIGRIS · LIGHT · BIONIC"},
		{name: "dark-only palette", prefs: Preferences{Palette: "baroque", Scheme: "light"}, want: "BAROQUE · DARK"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PreferencesSummary(test.prefs); got != test.want {
				t.Fatalf("PreferencesSummary() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPaletteGroupsPreserveOrder(t *testing.T) {
	got := PaletteGroups()
	want := []string{"THIS ARCHIVE", "FROM THE COLLECTION", "BORROWED"}
	if len(got) != len(want) {
		t.Fatalf("got %d groups, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("group %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPaletteSwatchStyle(t *testing.T) {
	if got := PaletteSwatchStyle("verdigris"); got != "background:linear-gradient(135deg,#edf1ec 0 50%,#1f5e55 50% 100%)" {
		t.Fatalf("PaletteSwatchStyle(verdigris) = %q", got)
	}
	if got := PaletteSwatchStyle("neon"); got != "background:linear-gradient(135deg,#f4f2ed 0 50%,#003366 50% 100%)" {
		t.Fatalf("unknown palette must default to the bone swatch, got %q", got)
	}
}

func TestThemeResolverScriptContracts(t *testing.T) {
	script := ThemeResolverScript()
	for _, want := range []string{
		`"wga-palette"`,
		`"wga_palette"`,
		`"wga-theme"`,
		`"wga_theme"`,
		`"wga_light"`,
		`"wga_dark"`,
		`"wga-rams"`,
		`"wga-rams-dark"`,
		`"wga-baroque"`,
		`"wga-tokyo"`,
		`prefers-color-scheme: dark`,
		`document.documentElement.dataset.theme`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("resolver script missing %q", want)
		}
	}
	// The corrected scheme contract must not introduce a separate scheme key.
	for _, forbidden := range []string{`wga-scheme`, `wga_scheme`} {
		if strings.Contains(script, forbidden) {
			t.Errorf("resolver script must not reference %q", forbidden)
		}
	}
}

func TestThemeResolverScriptPaletteFallbackPrecedence(t *testing.T) {
	script := ThemeResolverScript()

	// An invalid local value is truthy, so a || short-circuit would skip the
	// cookie and land on bone. The corrected resolver must validate the local
	// palette before falling back to the cookie.
	if strings.Contains(script, `readLocalStorage("wga-palette") || readCookie("wga_palette")`) {
		t.Fatal("resolver must not short-circuit local to cookie with ||; invalid local values would skip the cookie")
	}

	local := strings.Index(script, `readLocalStorage("wga-palette")`)
	cookie := strings.Index(script, `readCookie("wga_palette")`)
	fallback := strings.Index(script, `palette = DEFAULT_PALETTE`)
	if local < 0 || cookie < 0 || fallback < 0 {
		t.Fatal("resolver must read local palette, then cookie, then default")
	}
	if !(local < cookie && cookie < fallback) {
		t.Fatalf("expected local < cookie < default resolution order, got local=%d cookie=%d default=%d", local, cookie, fallback)
	}

	// Each fallback is validity-guarded, so a valid cookie is selected before
	// the bone default is ever considered.
	for _, guarded := range []string{
		"if (!THEMES[palette]) {\n\t\tpalette = readCookie(\"wga_palette\");",
		"if (!THEMES[palette]) {\n\t\tpalette = DEFAULT_PALETTE;",
	} {
		if !strings.Contains(script, guarded) {
			t.Errorf("resolver missing validity-guarded fallback %q", guarded)
		}
	}
}
