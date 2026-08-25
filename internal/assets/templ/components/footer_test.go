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
		`<li>Open-source licences</li>`,
		`data-wga-cookie-settings`,
		`>Contact</li>`,
	})
	if strings.Contains(rendered, `>Open-source licences</a>`) {
		t.Fatal("open-source licences must be plain text, not an anchor")
	}
}

func TestFooterRetainsPreferenceAndConsentMounts(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`data-wga-bionic-control`,
		`data-wga-bionic-toggle`,
		`aria-hidden="true" class="hidden flex-wrap items-center gap-[10px]" data-wga-theme-toggle`,
		`>APPEARANCE</span>`,
		`data-wga-theme-toggle`,
		`data-wga-theme="light" aria-pressed="false"`,
		`data-wga-theme="dark" aria-pressed="false"`,
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

func TestFooterHidesThemeControlsUntilJavaScriptInitialises(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `aria-hidden="true" class="hidden flex-wrap items-center gap-[10px]" data-wga-theme-toggle`) {
		t.Fatal("expected appearance controls to remain hidden before JavaScript initialises")
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
	if !strings.Contains(rendered, `grid-cols-2 gap-8 px-4 text-sm lg:grid-cols-[1.6fr_1fr_1fr_1fr]`) {
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
	if strings.Count(rendered, `wrap-anywhere`) != 3 {
		t.Fatal("expected footer link lists to wrap within their grid items")
	}
}
