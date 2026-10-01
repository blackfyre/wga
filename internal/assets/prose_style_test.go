package assets

// Contract tests for the design type scale on prose and links
// (openspec/changes/fix-type-scale-drift). They read the PCSS and Templ
// sources rather than the git-ignored build output.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// layerBody returns the body of the first `@layer <name> {` block in css.
func layerBody(t *testing.T, css, name string) string {
	t.Helper()
	open := "@layer " + name + " {"
	start := strings.Index(css, open)
	if start < 0 {
		t.Fatalf("style.pcss has no %q block", open)
	}
	depth := 0
	for i := start + len(open) - 1; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[start+len(open) : i]
			}
		}
	}
	t.Fatalf("style.pcss %q block is not closed", open)
	return ""
}

// ruleBody returns the nested body of the first rule whose selector line is
// exactly selector followed by " {".
func ruleBody(t *testing.T, css, selector string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(selector) + ` \{`).FindStringIndex(css)
	if m == nil {
		t.Fatalf("rule %q not found", selector)
	}
	depth := 0
	for i := m[1] - 1; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[m[1]:i]
			}
		}
	}
	t.Fatalf("rule %q is not closed", selector)
	return ""
}

func TestProseRuleUsesDesignScaleInsideComponentsLayer(t *testing.T) {
	css := stripComments(readSource(t))
	outside := strings.Replace(css, layerBody(t, css, "components"), "", 1)
	if regexp.MustCompile(`(?m)^\.content\s*\{`).MatchString(outside) {
		t.Fatal(".content must sit inside @layer components so surface type-scale utilities win over it")
	}

	prose := ruleBody(t, layerBody(t, css, "components"), ".content")
	for _, want := range []string{
		"font-size: var(--t-16);",
		"line-height: 1.7;",
		"color: var(--wga-text);",
	} {
		if !strings.Contains(prose, want) {
			t.Errorf(".content must declare %q", want)
		}
	}
	if strings.Contains(prose, "@apply") {
		t.Error(".content must use design tokens rather than Tailwind's named type sizes")
	}

	link := ruleBody(t, prose, "& a")
	for _, want := range []string{
		"font-weight: inherit;",
		"color: var(--wga-accent);",
		"text-decoration-line: underline;",
		"text-decoration-thickness: 1px;",
		"text-underline-offset: 0.18em;",
	} {
		if !strings.Contains(link, want) {
			t.Errorf(".content a must declare %q", want)
		}
	}

	// The bare-link default must be layered below components, otherwise its
	// weight beats the prose link rule regardless of specificity.
	if !strings.Contains(layerBody(t, css, "base"), "a:not([class]) {") {
		t.Error("a:not([class]) must be declared in @layer base")
	}
	if regexp.MustCompile(`(?m)^a:not\(\[class\]\)\s*\{`).MatchString(css) {
		t.Error("a:not([class]) must not remain unlayered")
	}
}

func TestFocusRingAndDeadProseRulesAreNotDuplicated(t *testing.T) {
	css := stripComments(readSource(t))
	if strings.Contains(css, ".home-prose") {
		t.Error("the unused .home-prose rule must not return")
	}
	if m := regexp.MustCompile(`:focus-visible\s*\{\s*outline-color:\s*#[0-9a-fA-F]+`).FindString(css); m != "" {
		t.Errorf("per-palette focus colours must come from --wga-accent, found %q", m)
	}
	focus := ruleBody(t, layerBody(t, css, "base"), ":focus-visible")
	if !strings.Contains(focus, "outline: 2px solid var(--wga-accent);") {
		t.Error("the base focus ring must use the --wga-accent role")
	}
}

func TestPublicSourcesStayOnTheTypeScale(t *testing.T) {
	rungs := parseTypeTokens(readSource(t))
	reference := regexp.MustCompile(`--t-[0-9]+\b`)
	named := regexp.MustCompile(`\btext-(?:lg|xl|2xl)\b`)

	check := func(path string, source []byte) {
		for _, ref := range reference.FindAll(source, -1) {
			if _, ok := rungs[strings.TrimPrefix(string(ref), "--")]; !ok {
				t.Errorf("%s uses %s, which is not a rung of the type scale", path, ref)
			}
		}
		if match := named.Find(source); match != nil {
			t.Errorf("%s uses Tailwind size %q; use the matching --t-* rung", path, match)
		}
	}

	check("resources/css/style.pcss", []byte(stripComments(readSource(t))))
	for _, root := range []string{"templ", "../../resources/js"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			ext := filepath.Ext(path)
			if entry.IsDir() || (ext != ".templ" && ext != ".ts") {
				return nil
			}
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			check(path, source)
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
}
