package components

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestFooterRendersBrowseDestinationsInReferenceOrder(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	assertDestinationOrder(t, rendered, []string{
		`href="/artists"`,
		`href="/artworks"`,
		`href="/timeline"`,
		`href="/inspire"`,
		`href="/tours"`,
		`href="/itineraries"`,
		`href="/glossary"`,
	})
}

func TestFooterRendersServicesDestinationsInReferenceOrder(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	assertDestinationOrder(t, rendered, []string{
		`href="/postcard"`,
		`<li>Period music</li>`,
		`href="/dual-mode"`,
		`href="/guestbook"`,
	})
	if strings.Contains(rendered, `>Period music</a>`) {
		t.Fatal("period music must be plain text, not an anchor")
	}
}

func TestFooterRendersAboutDestinationsInReferenceOrder(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	assertDestinationOrder(t, rendered, []string{
		`href="/pages/privacy-policy"`,
		`href="/pages/about"`,
		`href="/statistics"`,
		`href="/contributors"`,
		`href="/open-source-licences"`,
		`data-wga-cookie-settings`,
		`>Contact</li>`,
	})
	if !strings.Contains(rendered, `href="/open-source-licences" hx-get="/open-source-licences"`) {
		t.Fatal("open-source licences must use its implemented public route")
	}
}

func TestFooterRetainsPreferenceAndConsentMounts(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`data-wga-preferences-control`,
		`data-wga-preferences-open`,
		`aria-haspopup="dialog"`,
		`data-wga-preferences-summary`,
		`data-wga-preferences-swatch`,
		`data-wga-preferences-panel`,
		`id="wga-preferences"`,
		`aria-label="Preferences"`,
		`data-wga-preferences-close`,
		`data-wga-scheme="light"`,
		`data-wga-scheme="dark"`,
		`data-wga-scheme-explanation`,
		`data-wga-bionic-control`,
		`data-wga-bionic-toggle`,
		`data-cc="show-preferencesModal"`,
		`data-wga-cookie-settings`,
		`aria-hidden="true" class="hidden hover:text-primary"`,
		`tabindex="-1"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected footer contract %q", expected)
		}
	}
}

func TestFooterHidesPreferencesControlUntilJavaScriptInitialises(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `data-wga-preferences-control aria-hidden="true" class="hidden"`) {
		t.Fatal("expected preferences control to remain hidden before JavaScript initialises")
	}
}

func TestFooterGroupsPaletteChoicesByProvenance(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	groupOrder := []string{`>THIS ARCHIVE</h3>`, `>FROM THE COLLECTION</h3>`, `>BORROWED</h3>`}
	last := -1
	for _, heading := range groupOrder {
		index := strings.Index(rendered, heading)
		if index < 0 {
			t.Fatalf("expected provenance heading %q", heading)
		}
		if index <= last {
			t.Fatalf("provenance headings out of order at %q", heading)
		}
		last = index
	}

	for _, key := range []string{"bone", "classic", "verdigris", "gothic", "renaissance", "baroque", "rococo", "classical", "impressionist", "catppuccin", "tokyo"} {
		if !strings.Contains(rendered, `data-wga-palette="`+key+`"`) {
			t.Fatalf("expected palette radio %q", key)
		}
		if !strings.Contains(rendered, `role="radio"`) {
			t.Fatal("expected palette choices to be radio rows")
		}
	}
	if !strings.Contains(rendered, `data-wga-palette-label="BONE"`) {
		t.Fatal("expected palette rows to carry their label")
	}
}

func TestFooterMarksDefaultPaletteAndUnsetSchemeTruthfully(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `>BONE · LIGHT</span>`) {
		t.Fatal("expected the trigger summary to state the default bone light combination")
	}
	if !strings.Contains(rendered, `data-wga-palette="bone" data-wga-palette-label="BONE" aria-checked="true"`) {
		t.Fatal("expected the default bone palette row to be marked in use")
	}
	for _, unset := range []string{
		`data-wga-scheme="light" aria-pressed="false"`,
		`data-wga-scheme="dark" aria-pressed="false"`,
	} {
		if !strings.Contains(rendered, unset) {
			t.Fatalf("expected unset scheme to leave %q unmarked", unset)
		}
	}
}

func TestFooterPaletteRowsCarryReconciliationTargets(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	for _, key := range []string{"bone", "classic", "verdigris", "gothic", "renaissance", "baroque", "rococo", "classical", "impressionist", "catppuccin", "tokyo"} {
		if !strings.Contains(rendered, `data-wga-palette="`+key+`" data-wga-palette-label=`) {
			t.Fatalf("expected palette row %q to carry its label", key)
		}
		if !strings.Contains(rendered, `data-wga-palette-name`) {
			t.Fatal("expected palette rows to expose their label text target")
		}
	}
	// The default bone row is active, so exactly one row carries the marker.
	if count := strings.Count(rendered, `data-wga-palette-in-use`); count != 1 {
		t.Fatalf("expected exactly one in-use marker, got %d", count)
	}
	if !strings.Contains(rendered, `data-wga-palette="bone"`) {
		t.Fatal("expected the bone palette row to be present")
	}
}

func TestFooterMarksPaletteSchemeAndDarkOnlyFromCookies(t *testing.T) {
	tests := []struct {
		name    string
		cookies []*http.Cookie
		expect  []string
	}{
		{
			name:    "palette and scheme",
			cookies: []*http.Cookie{{Name: "wga_palette", Value: "verdigris"}, {Name: "wga_theme", Value: "dark"}},
			expect: []string{
				`>VERDIGRIS · DARK</span>`,
				`data-wga-palette="verdigris" data-wga-palette-label="VERDIGRIS" aria-checked="true"`,
				`data-wga-scheme="dark" aria-pressed="true"`,
			},
		},
		{
			name:    "dark-only palette disables light",
			cookies: []*http.Cookie{{Name: "wga_palette", Value: "baroque"}, {Name: "wga_theme", Value: "light"}},
			expect: []string{
				`>BAROQUE · DARK</span>`,
				`data-wga-palette="baroque" data-wga-palette-label="BAROQUE" aria-checked="true"`,
				`data-wga-scheme="light" aria-pressed="false" disabled title="BAROQUE is a dark-only palette"`,
				`data-wga-scheme="dark" aria-pressed="true"`,
				`BAROQUE has no light build, so light is unavailable while it is chosen.`,
			},
		},
		{
			name:    "bionic summary",
			cookies: []*http.Cookie{{Name: "wga_bionic", Value: "on"}},
			expect:  []string{`>BONE · LIGHT · BIONIC</span>`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			for _, cookie := range test.cookies {
				request.AddCookie(cookie)
			}

			var output bytes.Buffer
			if err := Footer().Render(utils.ContextFromRequest(request), &output); err != nil {
				t.Fatalf("render footer: %v", err)
			}

			rendered := output.String()
			for _, expected := range test.expect {
				if !strings.Contains(rendered, expected) {
					t.Errorf("expected footer contract %q", expected)
				}
			}
		})
	}
}

func TestFooterMarksBionicReadingFromRequestCookie(t *testing.T) {
	tests := []struct {
		name   string
		cookie *http.Cookie
		button string
	}{
		{name: "absent", button: `aria-checked="false" data-wga-bionic-toggle class="border border-base-content/20 bg-base-100`},
		{name: "off", cookie: &http.Cookie{Name: "wga_bionic", Value: "off"}, button: `aria-checked="false" data-wga-bionic-toggle class="border border-base-content/20 bg-base-100`},
		{name: "on", cookie: &http.Cookie{Name: "wga_bionic", Value: "on"}, button: `aria-checked="true" data-wga-bionic-toggle class="border border-primary bg-primary`},
		{name: "malformed", cookie: &http.Cookie{Name: "wga_bionic", Value: "enabled"}, button: `aria-checked="false" data-wga-bionic-toggle class="border border-base-content/20 bg-base-100`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			if tt.cookie != nil {
				request.AddCookie(tt.cookie)
			}

			var output bytes.Buffer
			if err := Footer().Render(utils.ContextFromRequest(request), &output); err != nil {
				t.Fatalf("render footer: %v", err)
			}

			rendered := output.String()
			if !strings.Contains(rendered, `data-wga-bionic-control class="hidden`) {
				t.Fatal("expected bionic control to remain hidden before JavaScript initialises")
			}
			if !strings.Contains(rendered, tt.button) {
				t.Fatalf("expected bionic button %q", tt.button)
			}
		})
	}
}

func TestFooterUsesReferenceColumnTiers(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `grid-cols-2 gap-[32px] px-4 text-sm lg:grid-cols-[1.6fr_1fr_1fr_1fr]`) {
		t.Fatal("expected two-column footer until the lg tier")
	}
	if strings.Contains(rendered, `md:grid-cols-[1.6fr_1fr_1fr_1fr]`) {
		t.Fatal("footer must not switch to four columns at the md tier")
	}
	if !strings.Contains(rendered, `>BROWSE</p>`) {
		t.Fatal("expected BROWSE footer heading")
	}
	if strings.Count(rendered, `class="min-w-0"`) != 4 {
		t.Fatal("expected each footer grid item to be able to shrink")
	}
	if strings.Count(rendered, `wrap-anywhere`) != 4 {
		t.Fatal("expected footer copy and link lists to wrap within their grid items")
	}
}

func TestFooterMarksSingularAndPluralRecordAliases(t *testing.T) {
	tests := []struct {
		path string
		href string
	}{
		{path: "/artist/albrecht-durer-a1", href: "/artists"},
		{path: "/artists/albrecht-durer-a1", href: "/artists"},
		{path: "/artist/albrecht-durer-a1/melencolia-work1", href: "/artworks"},
		{path: "/artists/albrecht-durer-a1/melencolia-work1", href: "/artworks"},
		{path: "/artist/albrecht-durer-a1/selections/selection1", href: "/artists"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			var output bytes.Buffer
			if err := Footer().Render(utils.ContextFromRequest(request), &output); err != nil {
				t.Fatalf("render footer: %v", err)
			}
			if !strings.Contains(output.String(), `href="`+tt.href+`" hx-get="`+tt.href+`" aria-current="page"`) {
				t.Fatalf("expected active footer destination %q", tt.href)
			}
		})
	}
}
