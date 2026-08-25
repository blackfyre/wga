package layouts

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/observability"
)

func TestLayoutBaseSentryConfiguration(t *testing.T) {
	tests := []struct {
		serverDSN  string
		name       string
		browserDSN string
	}{
		{
			serverDSN:  "https://server@example.ingest.sentry.io/1",
			name:       "configured browser monitoring",
			browserDSN: "https://browser@example.ingest.sentry.io/2",
		},
		{
			name: "disabled browser monitoring",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureLayoutSentry(t, test.serverDSN, test.browserDSN)

			var output bytes.Buffer
			if err := LayoutBase("", "").Render(context.Background(), &output); err != nil {
				t.Fatalf("render layout: %v", err)
			}

			content := output.String()
			if !strings.Contains(content, `name="sentry-dsn" content="`+test.browserDSN+`"`) {
				t.Fatalf("expected browser DSN %q in layout", test.browserDSN)
			}
			if test.serverDSN != "" && strings.Contains(content, test.serverDSN) {
				t.Fatal("server DSN must not be rendered")
			}
			if !strings.Contains(content, `name="sentry-environment" content="development"`) {
				t.Fatal("expected deployment environment in layout")
			}
			if !strings.Contains(content, `<script type="module" src="/assets/js/app.js"></script>`) {
				t.Fatal("expected browser bootstrap script in layout")
			}
		})
	}
}

func TestLayoutMainShowsNonProductionBuildInfo(t *testing.T) {
	ctx := utils.DecorateContext(context.Background(), utils.EnvironmentKey, "staging")
	var output bytes.Buffer
	if err := LayoutMain().Render(ctx, &output); err != nil {
		t.Fatalf("render main layout: %v", err)
	}

	if !strings.Contains(output.String(), "STAGING") || !strings.Contains(output.String(), "DEVELOPMENT BUILD — NOT FOR PUBLIC USE. CONTRIBUTE ON GITHUB.") {
		t.Fatal("expected non-production build information")
	}
}

func TestLayoutMainHidesBuildInfoInProduction(t *testing.T) {
	ctx := utils.DecorateContext(context.Background(), utils.EnvironmentKey, "production")
	var output bytes.Buffer
	if err := LayoutMain().Render(ctx, &output); err != nil {
		t.Fatalf("render main layout: %v", err)
	}

	if strings.Contains(output.String(), ">STAGING<") {
		t.Fatal("did not expect production build information")
	}
}

func TestLayoutFeedbackLinksToGitHubIssues(t *testing.T) {
	var output bytes.Buffer
	if err := LayoutMain().Render(context.Background(), &output); err != nil {
		t.Fatalf("render main layout: %v", err)
	}

	rendered := output.String()
	const feedbackURL = `https://github.com/blackfyre/wga/issues?q=sort%3Aupdated-desc+is%3Aissue+state%3Aopen+`
	feedback := rendered[strings.Index(rendered, `href="`+feedbackURL):]
	if !strings.Contains(feedback, `href="`+feedbackURL+`"`) {
		t.Fatalf("expected feedback link to GitHub issues: %s", feedbackURL)
	}
	for _, attribute := range []string{`hx-get=`, `hx-on:click=`, `hx-target=`, `hx-select=`, `hx-swap=`} {
		if strings.Contains(feedback, attribute) {
			t.Fatalf("feedback link must not contain %s", attribute)
		}
	}
}

func TestLayoutMainRetainsSharedMounts(t *testing.T) {
	var output bytes.Buffer
	if err := LayoutMain().Render(context.Background(), &output); err != nil {
		t.Fatalf("render main layout: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`id="mc-area"`,
		`<dialog id="d" aria-label="Dialog" aria-modal="true"`,
		`href="https://github.com/blackfyre/wga/issues?q=sort%3Aupdated-desc+is%3Aissue+state%3Aopen+"`,
		`wga-feedback-anchor`,
		`id="toast-container"`,
		`id="keyboard-palette"`,
		`src="/assets/js/app.js"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected shared layout mount %q", expected)
		}
	}
	feedback := rendered[strings.Index(rendered, `class="wga-feedback-anchor`):]
	if strings.Contains(feedback, `hx-get="/feedback"`) || strings.Contains(feedback, `hx-on:click="wga.dialog.open()"`) || strings.Contains(feedback, `href="#"`) {
		t.Fatal("feedback control must be an ordinary external link")
	}
}

func TestLayoutBaseAppliesThemeBeforeStylesheet(t *testing.T) {
	var output bytes.Buffer
	if err := LayoutBase("", "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	rendered := output.String()
	script := strings.Index(rendered, `<script>`)
	stylesheet := strings.Index(rendered, `<link rel="stylesheet" href="/assets/css/style.css">`)
	if script < 0 || stylesheet < 0 || script > stylesheet {
		t.Fatal("expected theme script before stylesheet")
	}
	for _, expected := range []string{
		`"light"`,
		`"dark"`,
		`"wga_light"`,
		`"wga_dark"`,
		`"wga_theme=light"`,
		`"wga_theme=dark"`,
		`"wga-rams"`,
		`"wga-rams-dark"`,
		`prefers-color-scheme: dark`,
	} {
		if !strings.Contains(rendered[script:stylesheet], expected) {
			t.Fatalf("expected inline theme script to handle %q", expected)
		}
	}
}

func TestLayoutBaseRendersTrustedHeadMarkupVerbatim(t *testing.T) {
	const markup = `<script src="/assets/js/trusted.js"></script>`

	ctx := utils.WithTrustedHeadMarkup(context.Background(), markup)
	var output bytes.Buffer
	if err := LayoutBase("", "").Render(ctx, &output); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	rendered := output.String()
	if strings.Count(rendered, markup) != 1 {
		t.Fatalf("trusted markup must render exactly once, got %d occurrences", strings.Count(rendered, markup))
	}
	if strings.Contains(rendered, `&lt;script src="/assets/js/trusted.js"&gt;&lt;/script&gt;`) {
		t.Fatal("trusted markup must not be HTML-escaped")
	}

	start := strings.Index(rendered, markup)
	headClose := strings.Index(rendered, "</head>")
	if start < 0 || headClose < 0 || start+len(markup) != headClose {
		t.Fatalf("trusted markup must render immediately before </head> (start=%d headClose=%d)", start, headClose)
	}
}

func TestLayoutBaseOmitsTrustedHeadMarkupWhenEmpty(t *testing.T) {
	ctx := utils.WithTrustedHeadMarkup(context.Background(), "")

	var output bytes.Buffer
	if err := LayoutBase("", "").Render(ctx, &output); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	rendered := output.String()
	// The theme-colour element is the last static head element; an empty trusted
	// fragment must leave the closing head tag directly adjacent to it.
	if !strings.Contains(rendered, `<meta name="theme-color" content="#013365"></head>`) {
		t.Fatal("empty trusted markup must not inject content before </head>")
	}
}

func TestLayoutBaseStillEscapesOrdinaryExpressions(t *testing.T) {
	ctx := utils.WithTrustedHeadMarkup(context.Background(), `<script src="/assets/js/trusted.js"></script>`)
	ctx = utils.DecorateContext(ctx, utils.TitleKey, `<em>Art</em>`)

	var output bytes.Buffer
	if err := LayoutBase("", "").Render(ctx, &output); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `&lt;em&gt;Art&lt;/em&gt;`) {
		t.Fatal("ordinary title expression must remain HTML-escaped")
	}
	if strings.Contains(rendered, `<em>Art</em>`) {
		t.Fatal("ordinary title expression must not be rendered raw")
	}
	if !strings.Contains(rendered, `<script src="/assets/js/trusted.js"></script>`) {
		t.Fatal("trusted markup must still render raw alongside escaped expressions")
	}
}

func configureLayoutSentry(t *testing.T, serverDSN string, browserDSN string) {
	t.Helper()
	values := map[string]string{
		"WGA_ENV":                "development",
		"WGA_PROTOCOL":           "http",
		"WGA_HOSTNAME":           "localhost:8090",
		"WGA_SENDER_NAME":        "WGA",
		"WGA_SENDER_ADDRESS":     "do-not-reply@example.com",
		"WGA_POSTCARD_FREQUENCY": "*/1 * * * *",
		"WGA_SENTRY_DSN":         serverDSN,
		"WGA_SENTRY_BROWSER_DSN": browserDSN,
	}
	server, err := config.LoadFrom(func(key string) string {
		return values[key]
	}).Server()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	observability.Configure(server.Sentry, server.Environment, slog.Default())
}
