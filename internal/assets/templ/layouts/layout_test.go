package layouts

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

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
