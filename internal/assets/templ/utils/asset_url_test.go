package utils_test

import (
	"testing"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
)

func TestAssetUrlUsesSharedConfiguredPublicURL(t *testing.T) {
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

	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() {
		apputils.ConfigurePublicURL(config.PublicURL{})
	})

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "application helper",
			url:  apputils.AssetUrl("assets/images/logo.png"),
			want: "https://gallery.example/assets/images/logo.png",
		},
		{
			name: "template helper",
			url:  templutils.AssetUrl("/postcard?p=abc"),
			want: "https://gallery.example/postcard?p=abc",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.url != test.want {
				t.Fatalf("expected URL %q, got %q", test.want, test.url)
			}
		})
	}
}
