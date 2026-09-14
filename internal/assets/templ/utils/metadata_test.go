package utils_test

import (
	"context"
	"strings"
	"testing"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestMetadataDefaultsAreComplete(t *testing.T) {
	ctx := context.Background()
	if description := templutils.GetDescription(ctx); strings.TrimSpace(description) == "" {
		t.Fatal("default description must not be empty")
	}

	openGraph := templutils.GetOpenGraphTags(ctx)
	for key, expected := range map[string]string{
		"og:title":        "Web Gallery of Art",
		"og:description":  templutils.GetDescription(ctx),
		"og:type":         "website",
		"og:site_name":    "Web Gallery of Art",
		"og:image:alt":    "Web Gallery of Art",
		"og:image:width":  "1080",
		"og:image:height": "1080",
	} {
		if got := openGraph[key]; got != expected {
			t.Errorf("Open Graph %s = %q, want %q", key, got, expected)
		}
	}
	if !strings.HasSuffix(openGraph["og:image"], "/assets/images/smo_cover_1080x1080.png") {
		t.Errorf("default Open Graph image = %q", openGraph["og:image"])
	}

	twitter := templutils.GetTwitterTags(ctx)
	for key, expected := range map[string]string{
		"twitter:card":        "summary_large_image",
		"twitter:title":       "Web Gallery of Art",
		"twitter:description": templutils.GetDescription(ctx),
		"twitter:image:alt":   "Web Gallery of Art",
	} {
		if got := twitter[key]; got != expected {
			t.Errorf("Twitter %s = %q, want %q", key, got, expected)
		}
	}
}

func TestMetadataUsesPageSpecificCanonicalImageDetails(t *testing.T) {
	ctx := templutils.DecorateContext(context.Background(), templutils.TitleKey, "The Arnolfini Portrait")
	ctx = templutils.DecorateContext(ctx, templutils.DescriptionKey, "Artwork description")
	ctx = templutils.DecorateContext(ctx, templutils.CanonicalUrlKey, "https://gallery.example/artworks/arnolfini")
	ctx = templutils.DecorateContext(ctx, templutils.OgImageKey, "https://gallery.example/image.jpg")
	ctx = templutils.DecorateContext(ctx, templutils.OgImageAltKey, "The Arnolfini Portrait")
	ctx = templutils.DecorateContext(ctx, templutils.TwitterImageAltKey, "The Arnolfini Portrait")

	openGraph := templutils.GetOpenGraphTags(ctx)
	for key, expected := range map[string]string{
		"og:title":        "The Arnolfini Portrait",
		"og:description":  "Artwork description",
		"og:url":          "https://gallery.example/artworks/arnolfini",
		"og:image":        "https://gallery.example/image.jpg",
		"og:image:alt":    "The Arnolfini Portrait",
	} {
		if got := openGraph[key]; got != expected {
			t.Errorf("Open Graph %s = %q, want %q", key, got, expected)
		}
	}
	if got := templutils.GetTwitterTags(ctx)["twitter:image:alt"]; got != "The Arnolfini Portrait" {
		t.Errorf("Twitter image alt = %q", got)
	}
}
