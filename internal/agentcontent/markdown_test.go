package agentcontent

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderArtistIncludesBoundedPublicContent(t *testing.T) {
	artist := Artist{
		Name:         "O'Keeffe, Georgia *Painter*",
		CanonicalURL: "https://gallery.example/artists/georgia-okeeffe",
		Lifespan:     "1887–1986",
		Profession:   "Painter [Modernist]",
		School:       "American Modernism",
		Biography:    "Known for <script>unsafe()</script> flowers & landscapes.",
		Attribution:  Link{Label: "Web Gallery of Art", URL: "https://gallery.example/pages/about"},
		RelatedArtworks: []Link{
			{Label: "Red Canna", URL: "https://gallery.example/artists/georgia-okeeffe/red-canna"},
			{Label: "Blue and Green Music", URL: "https://gallery.example/artists/georgia-okeeffe/blue-green-music"},
		},
	}

	first, err := RenderArtist(artist)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderArtist(artist)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("artist rendering is not deterministic")
	}

	content := string(first)
	for _, expected := range []string{
		`# O&#39;Keeffe, Georgia \*Painter\*`,
		`[View the canonical artist record](<https://gallery.example/artists/georgia-okeeffe>)`,
		`- Lifespan: 1887–1986`,
		`- Profession: Painter \[Modernist\]`,
		`- School: American Modernism`,
		`Known for &lt;script&gt;unsafe\(\)&lt;/script&gt; flowers &amp; landscapes.`,
		`[Blue and Green Music](<https://gallery.example/artists/georgia-okeeffe/blue-green-music>)`,
		`[Red Canna](<https://gallery.example/artists/georgia-okeeffe/red-canna>)`,
		`[Web Gallery of Art](<https://gallery.example/pages/about>)`,
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("artist Markdown missing %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "<script>") {
		t.Fatal("artist Markdown retained executable HTML")
	}
}

func TestRenderArtworkIncludesPublicFieldsAttributionAndLinks(t *testing.T) {
	artwork := Artwork{
		Title:        "Water [Lilies] #1",
		CanonicalURL: "https://gallery.example/artists/claude-monet/water-lilies",
		Artist:       Link{Label: "Monet, Claude", URL: "https://gallery.example/artists/claude-monet"},
		Date:         "c. 1916",
		Technique:    "Oil on canvas",
		Dimensions:   "200 × 180 cm",
		Location:     "Musée de l'Orangerie, Paris",
		Commentary:   "A reflective *surface* with <img src=x onerror=unsafe()>.",
		Attribution:  Link{Label: "Web Gallery of Art", URL: "https://gallery.example/pages/about"},
		RelatedArtworks: []Link{
			{Label: "The Japanese Footbridge", URL: "https://gallery.example/artists/claude-monet/japanese-footbridge"},
		},
	}

	rendered, err := RenderArtwork(artwork)
	if err != nil {
		t.Fatal(err)
	}
	content := string(rendered)
	for _, expected := range []string{
		`# Water \[Lilies\] \#1`,
		`[View the canonical artwork record](<https://gallery.example/artists/claude-monet/water-lilies>)`,
		`- Artist: [Monet, Claude](<https://gallery.example/artists/claude-monet>)`,
		`- Date: c. 1916`,
		`- Technique: Oil on canvas`,
		`- Dimensions: 200 × 180 cm`,
		`- Location: Musée de l&#39;Orangerie, Paris`,
		`A reflective \*surface\* with &lt;img src=x onerror=unsafe\(\)&gt;.`,
		`[The Japanese Footbridge](<https://gallery.example/artists/claude-monet/japanese-footbridge>)`,
		`[Web Gallery of Art](<https://gallery.example/pages/about>)`,
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("artwork Markdown missing %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "<img") {
		t.Fatal("artwork Markdown retained embedded HTML")
	}
}

func TestRenderBoundsProseAndRelatedLinks(t *testing.T) {
	related := make([]Link, MaxRelatedLinks+5)
	for index := range related {
		related[index] = Link{
			Label: fmt.Sprintf("Work %02d", index),
			URL:   fmt.Sprintf("https://gallery.example/artists/artist/work-%02d", index),
		}
	}
	rendered, err := RenderArtist(Artist{
		Name:            "Artist",
		CanonicalURL:    "https://gallery.example/artists/artist",
		Biography:       strings.Repeat("a", maxProseRunes+100),
		Attribution:     Link{Label: "WGA", URL: "https://gallery.example/pages/about"},
		RelatedArtworks: related,
	})
	if err != nil {
		t.Fatal(err)
	}
	content := string(rendered)
	if count := strings.Count(content, "\n- [Work"); count != MaxRelatedLinks {
		t.Fatalf("related links = %d; want %d", count, MaxRelatedLinks)
	}
	biography := content[strings.Index(content, "## Biography")+len("## Biography") : strings.Index(content, "## Related artworks")]
	if !strings.Contains(biography, "…") || utf8.RuneCountInString(strings.TrimSpace(biography)) != maxProseRunes {
		t.Fatalf("biography was not bounded to %d runes", maxProseRunes)
	}
}

func TestRenderRejectsNonCanonicalOrAdministrativeLinks(t *testing.T) {
	base := Artist{
		Name:         "Artist",
		CanonicalURL: "https://gallery.example/artists/artist",
		Attribution:  Link{Label: "WGA", URL: "https://gallery.example/pages/about"},
	}

	for _, test := range []struct {
		name   string
		mutate func(*Artist)
	}{
		{name: "relative canonical URL", mutate: func(value *Artist) { value.CanonicalURL = "/artists/artist" }},
		{name: "canonical query", mutate: func(value *Artist) { value.CanonicalURL += "?private=true" }},
		{name: "administrative attribution", mutate: func(value *Artist) { value.Attribution.URL = "https://gallery.example/_/admin" }},
		{name: "cross-host related record", mutate: func(value *Artist) {
			value.RelatedArtworks = []Link{{Label: "Other", URL: "https://other.example/artwork"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := base
			test.mutate(&value)
			if _, err := RenderArtist(value); err == nil {
				t.Fatal("render succeeded; want canonical URL error")
			}
		})
	}
}

func TestPublicProjectionsExcludePrivateDataCarriers(t *testing.T) {
	for _, projection := range []any{Artist{}, Artwork{}} {
		typeOf := reflect.TypeOf(projection)
		for index := range typeOf.NumField() {
			name := strings.ToLower(typeOf.Field(index).Name)
			for _, forbidden := range []string{"itinerary", "cookie", "script", "admin", "password", "token", "email", "created", "updated", "collection"} {
				if strings.Contains(name, forbidden) {
					t.Errorf("%s projection exposes forbidden field %q", typeOf.Name(), typeOf.Field(index).Name)
				}
			}
		}
	}
}
