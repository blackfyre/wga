package utils

import (
	"testing"

	"github.com/blackfyre/wga/internal/config"
)

func TestAssetUrlUsesConfiguredPublicURL(t *testing.T) {
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{
			"WGA_ENV":                "development",
			"WGA_PROTOCOL":           "https",
			"WGA_HOSTNAME":           "gallery.example",
			"WGA_SENDER_NAME":        "WGA",
			"WGA_SENDER_ADDRESS":     "sender@example.com",
			"WGA_POSTCARD_FREQUENCY": "*/1 * * * *",
		}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}

	ConfigurePublicURL(server.PublicURL)

	if got, want := AssetUrl("assets/images/logo.png"), "https://gallery.example/assets/images/logo.png"; got != want {
		t.Fatalf("expected URL %q, got %q", want, got)
	}
}
