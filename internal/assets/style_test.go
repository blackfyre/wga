package assets

// Focused contract test for resources/css/style.pcss (task 3.1). It proves,
// against the generated golden fixture derived from the accepted visual-overhaul
// reference (theme-board-tokens.css + wga-rams.css at 629089b), that the
// production stylesheet carries the complete palette x scheme token ledger and
// the structural invariants the design requires. It reads the source PCSS
// rather than the git-ignored built output, so it runs in a clean checkout.

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

//go:embed testdata/theme-golden.json
var themeGolden []byte

type golden struct {
	WgaRoles map[string]map[string]string `json:"wgaRoles"`
}

// paletteTheme maps the native palette/scheme axes to the immutable golden keys.
var paletteTheme = map[string]struct{ light, dark string }{
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

var darkOnly = map[string]bool{"baroque": true, "tokyo": true}

// The complete --wga-* interface role set every palette x scheme must define.
var wgaRoles = []string{
	"bg", "surface", "surface-2", "fill", "fill-line",
	"ink", "inv-bg", "inv-fg",
	"text", "text-2", "muted", "faint", "faint-2",
	"accent", "accent-2", "accent-bg", "accent-bg-2",
	"accent-tint", "accent-tint-2", "accent-soft", "accent-soft-2", "accent-strong",
	"rule-soft", "rule", "rule-2", "rule-3", "rule-4", "rule-control", "scrim",
	"series-0", "series-1", "series-2", "series-3", "series-4", "series-5", "series-6",
	"lane-movements", "lane-movements-fg",
	"lane-artists", "lane-artists-fg",
	"lane-works", "lane-works-fg",
	"lane-buildings", "lane-buildings-fg",
	"lane-events", "lane-events-fg",
	"lane-music", "lane-music-fg",
}

var laneNames = []string{"movements", "artists", "works", "buildings", "events", "music"}

// The 33 --t-* relative type-scale tokens, in rem, exactly as the reference
// declares them on :root.
var typeScale = map[string]string{
	"t-9": "0.6875rem", "t-95": "0.71875rem", "t-10": "0.75rem", "t-105": "0.78125rem",
	"t-11": "0.8125rem", "t-115": "0.84375rem", "t-12": "0.875rem", "t-125": "0.90625rem",
	"t-13": "0.9375rem", "t-135": "0.953125rem", "t-14": "0.96875rem", "t-15": "1rem",
	"t-16": "1.0625rem", "t-17": "1.09375rem", "t-18": "1.15625rem", "t-19": "1.21875rem",
	"t-20": "1.28125rem", "t-21": "1.34375rem", "t-22": "1.40625rem", "t-26": "1.625rem",
	"t-27": "1.6875rem", "t-28": "1.75rem", "t-30": "1.875rem", "t-32": "2rem",
	"t-34": "2.125rem", "t-36": "2.25rem", "t-38": "2.375rem", "t-40": "2.5rem",
	"t-44": "2.75rem", "t-46": "2.875rem", "t-48": "3rem", "t-52": "3.25rem",
	"t-56": "3.5rem",
}

const referenceFontSans = `-apple-system, "Helvetica Neue", Arial, sans-serif`

type contrastException struct {
	theme, role, ground string
	floor               float64
}

// acceptedContrastExceptions are the immutable reference's exact 53
// token/ground pairs below their documented floor. Everything else remains
// strict: an added, removed, or now-readable exception fails this contract.
var acceptedContrastExceptions = []contrastException{
	{"wga-gothic-dark", "series-6", "bg", 3}, {"wga-renaissance-dark", "series-6", "bg", 3}, {"wga-baroque", "series-6", "bg", 3}, {"wga-rococo-dark", "series-6", "bg", 3}, {"wga-classical-dark", "series-6", "bg", 3}, {"wga-catppuccin-dark", "series-6", "bg", 3}, {"wga-tokyo", "series-6", "bg", 3},
	{"wga-rams", "series-3", "bg", 3}, {"wga-rams", "series-4", "bg", 3}, {"wga-rams", "series-5", "bg", 3}, {"wga-rams", "series-6", "bg", 3},
	{"wga-classic", "series-4", "bg", 3}, {"wga-classic", "series-5", "bg", 3}, {"wga-classic", "series-6", "bg", 3},
	{"wga-verdigris", "series-3", "bg", 3}, {"wga-verdigris", "series-4", "bg", 3}, {"wga-verdigris", "series-5", "bg", 3}, {"wga-verdigris", "series-6", "bg", 3},
	{"wga-gothic", "series-4", "bg", 3}, {"wga-gothic", "series-5", "bg", 3}, {"wga-gothic", "series-6", "bg", 3},
	{"wga-renaissance", "series-4", "bg", 3}, {"wga-renaissance", "series-5", "bg", 3}, {"wga-renaissance", "series-6", "bg", 3},
	{"wga-rococo", "series-3", "bg", 3}, {"wga-rococo", "series-4", "bg", 3}, {"wga-rococo", "series-5", "bg", 3}, {"wga-rococo", "series-6", "bg", 3},
	{"wga-classical", "series-3", "bg", 3}, {"wga-classical", "series-4", "bg", 3}, {"wga-classical", "series-5", "bg", 3}, {"wga-classical", "series-6", "bg", 3},
	{"wga-impressionist", "series-3", "bg", 3}, {"wga-impressionist", "series-4", "bg", 3}, {"wga-impressionist", "series-5", "bg", 3}, {"wga-impressionist", "series-6", "bg", 3},
	{"wga-catppuccin", "series-3", "bg", 3}, {"wga-catppuccin", "series-4", "bg", 3}, {"wga-catppuccin", "series-5", "bg", 3}, {"wga-catppuccin", "series-6", "bg", 3},
	{"wga-classic", "faint", "fill", 4.5}, {"wga-verdigris", "faint", "surface-2", 4.5}, {"wga-verdigris", "faint", "fill", 4.5}, {"wga-gothic-dark", "faint", "fill", 4.5}, {"wga-renaissance", "faint", "fill", 4.5}, {"wga-classical", "faint", "fill", 4.5}, {"wga-catppuccin", "faint", "surface-2", 4.5}, {"wga-catppuccin", "faint", "fill", 4.5}, {"wga-catppuccin-dark", "faint", "fill", 4.5},
	{"wga-catppuccin", "rule-control", "bg", 3}, {"wga-catppuccin", "rule-control", "surface", 3}, {"wga-catppuccin", "rule-control", "surface-2", 3}, {"wga-catppuccin", "rule-control", "fill", 3},
}

func readSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../resources/css/style.pcss")
	if err != nil {
		t.Fatalf("read resources/css/style.pcss: %v", err)
	}
	return string(b)
}

func TestPublicSourcesUseWgaOwnedStyleVocabulary(t *testing.T) {
	t.Helper()
	forbidden := regexp.MustCompile(`(?:^|[\s"'=])((?:bg|text|border|outline|ring|shadow|stroke|fill|divide)-(?:base-(?:100|200|300|content)|primary(?:-content)?|secondary(?:-content)?|accent(?:-content)?|neutral(?:-content)?|info(?:-content)?|success(?:-content)?|warning(?:-content)?|error(?:-content)?)|btn(?:-(?:primary|secondary|outline|ghost|neutral|sm))?|badge(?:-(?:primary|secondary))?|card-(?:body|title|actions)|modal-(?:box|backdrop)|table-sm|input-bordered|select-bordered|link-primary|alert-(?:info|warning|error|success|horizontal)|skeleton|rounded-box|label-text)\b`)

	roots := []string{"templ", "../../resources/js", "../../resources/css"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			ext := filepath.Ext(path)
			if entry.IsDir() || (ext != ".templ" && ext != ".ts" && ext != ".pcss") || strings.HasSuffix(path, ".test.ts") {
				return nil
			}
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if match := forbidden.FindSubmatch(source); match != nil {
				t.Errorf("%s retains third-party style vocabulary %q", path, match[1])
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan public style consumers under %s: %v", root, err)
		}
	}
}

func TestPublicSourcesUseRelativeTypeScale(t *testing.T) {
	forbidden := regexp.MustCompile(`\btext-\[[0-9]+(?:\.[0-9]+)?px\]`)
	lowestRung := regexp.MustCompile(`text-\(length:--t-9(?:5)?\)`)
	opacityHover := regexp.MustCompile(`(?:group-)?hover:opacity`)

	for _, root := range []string{"templ", "../../resources/js"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			ext := filepath.Ext(path)
			if entry.IsDir() || (ext != ".templ" && ext != ".ts") || strings.HasSuffix(path, ".test.ts") {
				return nil
			}
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if match := forbidden.Find(source); match != nil {
				t.Errorf("%s retains pixel-based typography %q; use the --t-* rem scale", path, match)
			}
			if filepath.Base(path) != "keyboard.templ" {
				if match := lowestRung.Find(source); match != nil {
					t.Errorf("%s uses the smallest type rung %q outside the keyboard hint", path, match)
				}
			}
			if filepath.Base(path) != "statistics.templ" {
				if match := opacityHover.Find(source); match != nil {
					t.Errorf("%s retains opacity-based hover feedback %q; use action colour, record tint, or chip border-and-tint feedback", path, match)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan public typography consumers under %s: %v", root, err)
		}
	}

	css := readSource(t)
	if match := regexp.MustCompile(`font-size:\s*[0-9.]+(?:px|rem)`).FindString(css); match != "" {
		t.Errorf("resources/css/style.pcss retains literal typography %q; use the --t-* rem scale", match)
	}
	for _, webfont := range []string{"Lexend", "Merriweather", "Noto"} {
		if strings.Contains(css, webfont) {
			t.Errorf("resources/css/style.pcss retains non-system font %q", webfont)
		}
	}
}

func TestThirdPartyStyleDependencyIsAbsent(t *testing.T) {
	t.Helper()
	needle := strings.Join([]string{"daisy", "ui"}, "")
	for _, path := range []string{
		"../../package.json",
		"../../bun.lock",
		"../../package-lock.json",
		"../../yarn.lock",
		"../../resources/css/style.pcss",
		"../licences/manifest.json",
		"views/open-source-licences.html",
	} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(strings.ToLower(string(source)), needle) {
			t.Errorf("%s retains removed style dependency metadata", path)
		}
	}
}

func TestExternalLinkMarkerOptOutIsScoped(t *testing.T) {
	source := readSource(t)
	selector := `)):not(.has-sm-icon):not(.no-external-link-marker):after {`
	if !strings.Contains(source, selector) {
		t.Fatalf("external-link marker selector must exclude the explicit opt-out class")
	}
	if strings.Contains(source, `.no-external-link-marker:after`) {
		t.Fatalf("marker suppression must not define a standalone pseudo-element rule")
	}
}

func loadGolden(t *testing.T) golden {
	t.Helper()
	var g golden
	if err := json.Unmarshal(themeGolden, &g); err != nil {
		t.Fatalf("unmarshal golden fixture: %v", err)
	}
	return g
}

// normColor lowercases and strips whitespace so equivalent colour literals
// (case and rgba() spacing) compare equal.
func normColor(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(s, " ", "")
}

func stripComments(css string) string {
	return regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(css, "")
}

// parseWgaRoles maps each native palette/scheme selector (and :root) to the
// immutable golden key for its --wga-* declaration set.
func parseWgaRoles(css string) map[string]map[string]string {
	css = stripComments(css)
	out := map[string]map[string]string{}
	blockRe := regexp.MustCompile(`([^{}]+)\{([^{}]*--wga-[a-z0-9-]+\s*:[^{}]*)\}`)
	paletteRe := regexp.MustCompile(`\[data-palette="([^"]+)"\]`)
	schemeRe := regexp.MustCompile(`\[data-theme="(light|dark)"\]`)
	roleRe := regexp.MustCompile(`--wga-([a-z0-9-]+)\s*:\s*([^;]+);`)
	for _, m := range blockRe.FindAllStringSubmatch(css, -1) {
		selector, body := m[1], m[2]
		roles := map[string]string{}
		for _, r := range roleRe.FindAllStringSubmatch(body, -1) {
			roles[r[1]] = strings.TrimSpace(r[2])
		}
		if len(roles) == 0 {
			continue
		}
		names := []string{}
		paletteMatch := paletteRe.FindStringSubmatch(selector)
		if paletteMatch != nil {
			palette := paletteMatch[1]
			pair, ok := paletteTheme[palette]
			if ok {
				name := pair.light
				if scheme := schemeRe.FindStringSubmatch(selector); scheme != nil && scheme[1] == "dark" {
					name = pair.dark
				}
				names = append(names, name)
			}
		}
		if strings.HasPrefix(strings.TrimSpace(selector), ":root,") || strings.TrimSpace(selector) == ":root" {
			names = append(names, ":root")
		}
		for _, name := range names {
			out[name] = roles
		}
	}
	return out
}

func parseTypeTokens(css string) map[string]string {
	css = stripComments(css)
	out := map[string]string{}
	for _, r := range regexp.MustCompile(`--(t-[0-9.]+)\s*:\s*([^;]+);`).FindAllStringSubmatch(css, -1) {
		out[r[1]] = strings.TrimSpace(r[2])
	}
	return out
}

func parseThemeVar(css, name string) (string, bool) {
	m := regexp.MustCompile(`--` + regexp.QuoteMeta(name) + `\s*:\s*([^;]+);`).FindStringSubmatch(stripComments(css))
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

// ---------------------------------------------------------------------------
// Colour math (WCAG 2.x relative luminance / contrast).
// ---------------------------------------------------------------------------

func hexRGB(s string) ([3]uint8, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return [3]uint8{}, false
	}
	var v uint32
	if _, err := fmt.Sscanf(s, "%x", &v); err != nil {
		return [3]uint8{}, false
	}
	return [3]uint8{uint8(v >> 16), uint8(v >> 8), uint8(v)}, true
}

func luminance(c [3]uint8) float64 {
	lin := func(ch uint8) float64 {
		v := float64(ch) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c[0]) + 0.7152*lin(c[1]) + 0.0722*lin(c[2])
}

func contrastRatio(a, b [3]uint8) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func requireContrast(t *testing.T, theme, fg, bg string, floor float64) {
	t.Helper()
	f, ok := hexRGB(fg)
	if !ok {
		t.Errorf("%s: foreground %q is not an opaque hex colour", theme, fg)
		return
	}
	b, ok := hexRGB(bg)
	if !ok {
		t.Errorf("%s: background %q is not an opaque hex colour", theme, bg)
		return
	}
	if r := contrastRatio(f, b); r < floor {
		t.Errorf("%s: %s on %s contrast %.2f:1 < %.1f:1", theme, fg, bg, r, floor)
	}
}

func contrastExceptionKey(theme, role, ground string, floor float64) string {
	return fmt.Sprintf("%s|%s|%s|%.1f", theme, role, ground, floor)
}

func contrastExceptionSet(t *testing.T) map[string]struct{} {
	t.Helper()
	if len(acceptedContrastExceptions) != 53 {
		t.Fatalf("accepted contrast exception ledger has %d entries, want exactly 53", len(acceptedContrastExceptions))
	}
	set := make(map[string]struct{}, len(acceptedContrastExceptions))
	for _, exception := range acceptedContrastExceptions {
		key := contrastExceptionKey(exception.theme, exception.role, exception.ground, exception.floor)
		if _, exists := set[key]; exists {
			t.Fatalf("duplicate accepted contrast exception %s", key)
		}
		set[key] = struct{}{}
	}
	return set
}

func requireLedgerContrast(t *testing.T, accepted map[string]struct{}, seen map[string]struct{}, theme, role, ground, fg, bg string, floor float64) {
	t.Helper()
	f, ok := hexRGB(fg)
	if !ok {
		t.Errorf("%s: --wga-%s %q is not an opaque hex colour", theme, role, fg)
		return
	}
	b, ok := hexRGB(bg)
	if !ok {
		t.Errorf("%s: %s ground %q is not an opaque hex colour", theme, ground, bg)
		return
	}
	requireLedgerRatio(t, accepted, seen, theme, role, ground, fg, bg, contrastRatio(f, b), floor)
}

func requireLedgerRatio(t *testing.T, accepted map[string]struct{}, seen map[string]struct{}, theme, role, ground, fg, bg string, ratio, floor float64) {
	t.Helper()
	key := contrastExceptionKey(theme, role, ground, floor)
	_, isAccepted := accepted[key]
	if ratio < floor {
		if !isAccepted {
			t.Errorf("%s: --wga-%s %s on %s contrast %.2f:1 < %.1f:1 is not an accepted exception", theme, role, fg, bg, ratio, floor)
			return
		}
		seen[key] = struct{}{}
		return
	}
	if isAccepted {
		t.Errorf("%s: --wga-%s on %s contrast %.2f:1 no longer matches accepted exception %.1f:1", theme, role, ground, ratio, floor)
	}
}

// isDarkTheme reports whether a theme selector carries the dark scheme build.
// Legacy "wga_dark" uses an underscore and dark-only palettes carry no "-dark"
// suffix, so neither matches a simple "-dark" suffix test.
func isDarkTheme(name string) bool {
	if name == "wga_dark" || name == "wga-baroque" || name == "wga-tokyo" {
		return true
	}
	return strings.HasSuffix(name, "-dark")
}

// parseRGBA reads an `rgba(r,g,b,a)` literal (spaces optional) into its RGB
// channels and alpha.
func parseRGBA(s string) ([3]uint8, float64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if !strings.HasPrefix(s, "rgba(") || !strings.HasSuffix(s, ")") {
		return [3]uint8{}, 0, false
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(s, "rgba("), ")"), ",")
	if len(parts) != 4 {
		return [3]uint8{}, 0, false
	}
	var c [3]uint8
	for i := 0; i < 3; i++ {
		var v int
		if _, err := fmt.Sscanf(parts[i], "%d", &v); err != nil || v < 0 || v > 255 {
			return [3]uint8{}, 0, false
		}
		c[i] = uint8(v)
	}
	var a float64
	if _, err := fmt.Sscanf(parts[3], "%f", &a); err != nil || a < 0 || a > 1 {
		return [3]uint8{}, 0, false
	}
	return c, a, true
}

// compositeOver alpha-composites an rgba colour over an opaque ground, the same
// result the browser produces when the two layers are painted together.
func compositeOver(rgba [3]uint8, alpha float64, ground [3]uint8) [3]uint8 {
	return [3]uint8{
		uint8(math.Round(alpha*float64(rgba[0]) + (1-alpha)*float64(ground[0]))),
		uint8(math.Round(alpha*float64(rgba[1]) + (1-alpha)*float64(ground[1]))),
		uint8(math.Round(alpha*float64(rgba[2]) + (1-alpha)*float64(ground[2]))),
	}
}

func TestStylePCSSThemeContract(t *testing.T) {
	css := readSource(t)
	golden := loadGolden(t)
	wga := parseWgaRoles(css)
	acceptedExceptions := contrastExceptionSet(t)
	seenExceptions := map[string]struct{}{}

	t.Run("font stack", func(t *testing.T) {
		v, ok := parseThemeVar(css, "font-sans")
		if !ok {
			t.Fatal("--font-sans not found in @theme")
		}
		if normColor(v) != normColor(referenceFontSans) {
			t.Fatalf("--font-sans = %q, want reference stack %q", v, referenceFontSans)
		}
		if strings.Contains(strings.ToLower(v), "noto") {
			t.Fatalf("--font-sans still leads with the Noto webfont: %q", v)
		}
	})

	t.Run("type scale", func(t *testing.T) {
		tokens := parseTypeTokens(css)
		if len(tokens) != 33 {
			t.Fatalf("got %d --t-* tokens, want 33", len(tokens))
		}
		for name, want := range typeScale {
			got, ok := tokens[name]
			if !ok {
				t.Errorf("missing --%s", name)
				continue
			}
			if got != want {
				t.Errorf("--%s = %q, want %q", name, got, want)
			}
		}
	})

	t.Run("responsive tiers", func(t *testing.T) {
		md, ok := parseThemeVar(css, "breakpoint-md")
		if !ok || md != "45rem" {
			t.Errorf("--breakpoint-md = %q (ok=%v), want 45rem (720px)", md, ok)
		}
		lg, ok := parseThemeVar(css, "breakpoint-lg")
		if !ok || lg != "67.5rem" {
			t.Errorf("--breakpoint-lg = %q (ok=%v), want 67.5rem (1080px)", lg, ok)
		}
	})

	t.Run("native palette and scheme selectors", func(t *testing.T) {
		for palette := range paletteTheme {
			if darkOnly[palette] {
				if !strings.Contains(css, `:root[data-palette="`+palette+`"] {`) {
					t.Errorf("missing native dark-only selector for %q", palette)
				}
				continue
			}
			for _, scheme := range []string{"light", "dark"} {
				selector := `:root[data-palette="` + palette + `"][data-theme="` + scheme + `"] {`
				if !strings.Contains(css, selector) {
					t.Errorf("missing native selector %s", selector)
				}
			}
		}
	})

	t.Run("native roles follow the operating system without attributes", func(t *testing.T) {
		for _, want := range []string{
			`@media (prefers-color-scheme: dark) {`,
			`:root:not([data-palette]) {`,
			`--wga-bg: #1A1814;`,
			`--wga-ink: #EDEAE1;`,
		} {
			if !strings.Contains(css, want) {
				t.Errorf("missing no-JavaScript operating-system fallback %q", want)
			}
		}
	})

	t.Run("tailwind exposes every WGA role", func(t *testing.T) {
		for _, role := range wgaRoles {
			mapping := `--color-wga-` + role + `: var(--wga-` + role + `);`
			if !strings.Contains(css, mapping) {
				t.Errorf("missing Tailwind role mapping %q", mapping)
			}
		}
	})

	t.Run("third-party theme surface is absent", func(t *testing.T) {
		forbidden := []*regexp.Regexp{
			regexp.MustCompile(`@plugin\s+"[^"]+/theme"`),
			regexp.MustCompile(`--color-(?:base-(?:100|200|300|content)|primary(?:-content)?|secondary(?:-content)?|accent(?:-content)?|neutral(?:-content)?|info(?:-content)?|success(?:-content)?|warning(?:-content)?|error(?:-content)?)\s*:`),
		}
		for _, pattern := range forbidden {
			if match := pattern.FindString(css); match != "" {
				t.Errorf("stylesheet retains third-party theme API %q", match)
			}
		}
	})

	t.Run("role classes", func(t *testing.T) {
		for _, want := range []string{
			".text-muted { color: var(--wga-muted); }",
			".text-faint { color: var(--wga-faint); }",
			".text-faint-2 { color: var(--wga-faint-2); }",
			".border-control { border-color: var(--wga-rule-control); }",
		} {
			if !strings.Contains(css, want) {
				t.Errorf("missing class rule: %s", want)
			}
		}
	})

	t.Run("wga role parity", func(t *testing.T) {
		if len(golden.WgaRoles) != 20 {
			t.Fatalf("golden fixture has %d WgaRoles theme sets, want 20", len(golden.WgaRoles))
		}
		for name, ref := range golden.WgaRoles {
			if len(ref) != 48 {
				t.Fatalf("%s golden role set has %d roles, want 48", name, len(ref))
			}
		}
		for name, ref := range golden.WgaRoles {
			got, ok := wga[name]
			if !ok {
				t.Errorf("--wga-* role set %q missing from source", name)
				continue
			}
			for role, want := range ref {
				g, ok := got[role]
				if !ok {
					t.Errorf("%s: missing --wga-%s", name, role)
					continue
				}
				if g != want {
					t.Errorf("%s: --wga-%s = %q, want %q", name, role, g, want)
				}
			}
		}
	})

	t.Run("role completeness", func(t *testing.T) {
		for name := range wga {
			for _, role := range wgaRoles {
				if _, ok := wga[name][role]; !ok {
					t.Errorf("%s: missing --wga-%s", name, role)
				}
			}
		}
	})

	t.Run("series ramp", func(t *testing.T) {
		for name, roles := range wga {
			scheme := "light"
			if isDarkTheme(name) {
				scheme = "dark"
			}
			var lum []float64
			for i := 0; i < 7; i++ {
				c, ok := hexRGB(roles[fmt.Sprintf("series-%d", i)])
				if !ok {
					t.Errorf("%s: series-%d is not an opaque hex colour: %q", name, i, roles[fmt.Sprintf("series-%d", i)])
					lum = append(lum, -1)
					continue
				}
				lum = append(lum, luminance(c))
			}
			// Monotonic by luminance. Light schemes run dark -> light (series-0
			// is the most prominent / furthest from a pale ground); dark schemes
			// invert so the most prominent step is the lightest.
			for i := 1; i < 7; i++ {
				if lum[i] < 0 || lum[i-1] < 0 {
					continue
				}
				if scheme == "light" && lum[i] <= lum[i-1] {
					t.Errorf("%s: series-%d luminance %.4f not increasing from series-%d (%.4f)", name, i, lum[i], i-1, lum[i-1])
				}
				if scheme == "dark" && lum[i] >= lum[i-1] {
					t.Errorf("%s: series-%d luminance %.4f not decreasing from series-%d (%.4f)", name, i, lum[i], i-1, lum[i-1])
				}
			}
			// Every step clears the 3:1 non-text floor against the page ground,
			// except for the immutable reference pairs in the exact ledger.
			if name == ":root" {
				// :root is the no-JavaScript bootstrap copy of the bone/light
				// selector measured below; do not count it twice in the ledger.
				continue
			}
			for i := 0; i < 7; i++ {
				role := fmt.Sprintf("series-%d", i)
				requireLedgerContrast(t, acceptedExceptions, seenExceptions, name, role, "bg", roles[role], roles["bg"], 3.0)
			}
		}
	})

	t.Run("lane pairs", func(t *testing.T) {
		for name, roles := range wga {
			for _, lane := range laneNames {
				bg, bgOK := roles["lane-"+lane]
				fg, fgOK := roles["lane-"+lane+"-fg"]
				if !bgOK || !fgOK {
					t.Errorf("%s: missing lane-%s / lane-%s-fg", name, lane, lane)
					continue
				}
				// A lane is a non-text colour field; its label is additionally
				// separated by position and the bar's own colour.
				requireContrast(t, name, fg, bg, 3.0)
			}
		}
	})

	t.Run("text contrast floors", func(t *testing.T) {
		for name, roles := range wga {
			// Running text and the three readable grey ranks clear 4.5:1 on
			// every ground they land on (page, surface, surface-2 and fill).
			// --wga-faint-2 is a placeholder/disabled rank and is deliberately
			// not held to the text floor.
			for _, ground := range []string{"bg", "surface", "surface-2", "fill"} {
				for _, role := range []string{"ink", "text", "text-2", "muted", "faint"} {
					requireLedgerContrast(t, acceptedExceptions, seenExceptions, name, role, ground, roles[role], roles[ground], 4.5)
				}
			}
			// The accent ground carries type, so --wga-accent-bg / --wga-inv-fg
			// is measured at the text floor, not the non-text floor.
			requireContrast(t, name, roles["inv-fg"], roles["accent-bg"], 4.5)
		}
	})

	t.Run("rule-control composited contrast", func(t *testing.T) {
		for name, roles := range wga {
			c, a, ok := parseRGBA(roles["rule-control"])
			if !ok {
				t.Errorf("%s: --wga-rule-control %q is not rgba()", name, roles["rule-control"])
				continue
			}
			for _, ground := range []string{"bg", "surface", "surface-2", "fill"} {
				g, ok := hexRGB(roles[ground])
				if !ok {
					t.Errorf("%s: %s ground %q is not an opaque hex colour", name, ground, roles[ground])
					continue
				}
				comp := compositeOver(c, a, g)
				requireLedgerRatio(t, acceptedExceptions, seenExceptions, name, "rule-control", ground, roles["rule-control"], roles[ground], contrastRatio(comp, g), 3.0)
			}
		}
	})

	t.Run("contrast exception ledger", func(t *testing.T) {
		if len(seenExceptions) != len(acceptedExceptions) {
			t.Errorf("observed %d accepted contrast exceptions, want %d", len(seenExceptions), len(acceptedExceptions))
		}
		for key := range acceptedExceptions {
			if _, seen := seenExceptions[key]; !seen {
				t.Errorf("accepted contrast exception missing or ratio changed: %s", key)
			}
		}
	})
}
