package utils_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
)

func TestSiteJSONLDDescribesWebsiteAndPublisher(t *testing.T) {
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

	encoded := templutils.SiteJSONLD()
	if strings.Contains(encoded, "<") {
		t.Fatalf("site JSON-LD must not contain a literal '<': %s", encoded)
	}

	var document struct {
		Context string           `json:"@context"`
		Graph   []map[string]any `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(encoded), &document); err != nil {
		t.Fatalf("site JSON-LD is not valid JSON: %v", err)
	}
	if document.Context != "https://schema.org" {
		t.Errorf("@context = %q", document.Context)
	}

	nodes := map[string]map[string]any{}
	for _, node := range document.Graph {
		nodes[node["@type"].(string)] = node
	}
	if len(document.Graph) != 2 || nodes["WebSite"] == nil || nodes["Organization"] == nil {
		t.Fatalf("graph must hold exactly WebSite and Organization nodes: %s", encoded)
	}

	const root = "https://gallery.example/"
	website := nodes["WebSite"]
	for key, want := range map[string]string{
		"@id":        root + "#website",
		"name":       "Web Gallery of Art",
		"url":        root,
		"inLanguage": "en",
	} {
		if got := website[key]; got != want {
			t.Errorf("WebSite %s = %v, want %q", key, got, want)
		}
	}
	if publisher, _ := website["publisher"].(map[string]any); publisher["@id"] != root+"#organization" {
		t.Errorf("WebSite publisher = %v, want reference to the Organization", website["publisher"])
	}

	organization := nodes["Organization"]
	for key, want := range map[string]string{
		"@id":  root + "#organization",
		"name": "Web Gallery of Art",
		"url":  root,
		"logo": root + "assets/images/logo.png",
	} {
		if got := organization[key]; got != want {
			t.Errorf("Organization %s = %v, want %q", key, got, want)
		}
	}
}

func TestDescriptionIsReadablePlainText(t *testing.T) {
	ctx := templutils.DecorateContext(context.Background(), templutils.DescriptionKey, "<p>First paragraph.</p><p>Second &amp; last.</p>")
	if got, want := templutils.GetDescription(ctx), "First paragraph. Second & last."; got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

func TestDescriptionTruncationKeepsWholeCharacters(t *testing.T) {
	ctx := templutils.DecorateContext(context.Background(), templutils.DescriptionKey, strings.Repeat("é", 200))
	got := templutils.GetDescription(ctx)
	if !utf8.ValidString(got) {
		t.Fatalf("truncated description is not valid UTF-8: %q", got)
	}
	if want := strings.Repeat("é", 157) + "..."; got != want {
		t.Fatalf("description = %q (%d runes), want 157 runes and an ellipsis", got, utf8.RuneCountInString(got))
	}
}
