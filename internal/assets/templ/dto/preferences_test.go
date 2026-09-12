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
		if option.Label == "" || option.Group == "" || option.Desc == "" || option.Paper == "" || option.Ink == "" {
			t.Fatalf("palette %q has an empty presentation field", option.Key)
		}
	}
}

func TestPaletteTableMatchesPaletteOptions(t *testing.T) {
	want := map[string]bool{
		"bone": false, "classic": false, "verdigris": false,
		"gothic": false, "renaissance": false, "baroque": true,
		"rococo": false, "classical": false, "impressionist": false,
		"catppuccin": false, "tokyo": true,
	}

	var got map[string]bool
	if err := json.Unmarshal([]byte(PaletteTableJSON()), &got); err != nil {
		t.Fatalf("unmarshal palette table: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("theme table has %d entries, want %d", len(got), len(want))
	}
	for key, darkOnly := range want {
		if got[key] != darkOnly {
			t.Errorf("palette table[%q] = %v, want %v", key, got[key], darkOnly)
		}
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
		`"wga-theme"`,
		`"wga_light"`,
		`"wga_dark"`,
		`"bone"`,
		`"baroque":true`,
		`"tokyo":true`,
		`prefers-color-scheme: dark`,
		`document.documentElement.dataset.palette`,
		`document.documentElement.dataset.theme`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("resolver script missing %q", want)
		}
	}
	// The corrected scheme contract must not introduce a separate scheme key.
	for _, forbidden := range []string{`wga-scheme`, `wga_scheme`, `wga_palette`, `wga_theme`, `document.cookie`} {
		if strings.Contains(script, forbidden) {
			t.Errorf("resolver script must not reference %q", forbidden)
		}
	}
}

func TestThemeResolverScriptPaletteFallback(t *testing.T) {
	script := ThemeResolverScript()
	local := strings.Index(script, `readLocalStorage("wga-palette")`)
	fallback := strings.Index(script, `palette = DEFAULT_PALETTE`)
	if local < 0 || fallback < 0 {
		t.Fatal("resolver must read local palette before using the default")
	}
	if local >= fallback {
		t.Fatalf("expected local < default resolution order, got local=%d default=%d", local, fallback)
	}
	guarded := "if (!Object.prototype.hasOwnProperty.call(PALETTES, palette)) {\n\t\tpalette = DEFAULT_PALETTE;"
	if !strings.Contains(script, guarded) {
		t.Errorf("resolver missing validity-guarded fallback %q", guarded)
	}
}
