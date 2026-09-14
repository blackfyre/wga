package pages

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func renderHome(t *testing.T, content HomePage) string {
	t.Helper()

	var output bytes.Buffer
	if err := HomePageContent(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render home page: %v", err)
	}

	return output.String()
}

func TestHomeUsesOnlyTheSharedHeadAndReturnsAnHTMXTitle(t *testing.T) {
	var full bytes.Buffer
	if err := HomePageWrapped(HomePage{}).Render(context.Background(), &full); err != nil {
		t.Fatalf("render full home page: %v", err)
	}
	if count := strings.Count(full.String(), "<head>"); count != 1 {
		t.Fatalf("full document head count = %d, want 1", count)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("HX-Request", "true")
	ctx := templutils.DecorateContext(templutils.ContextFromRequest(request), templutils.TitleKey, "Home")
	var fragment bytes.Buffer
	if err := HomePageContent(HomePage{}).Render(ctx, &fragment); err != nil {
		t.Fatalf("render HTMX home fragment: %v", err)
	}
	if strings.Contains(fragment.String(), "<head>") || !strings.Contains(fragment.String(), "<title>Home - WGA</title>") {
		t.Fatalf("HTMX fragment must carry a root title without a nested head: %s", fragment.String())
	}
}

func TestHomeRendersCollectionDiscoveryAndWorks(t *testing.T) {
	rendered := renderHome(t, HomePage{
		ArtistCount: "4,012", ArtworkCount: "49,610", SchoolCount: "128",
		FeaturedArtwork: HomeFeaturedArtwork{Title: "The Annunciation", Artist: "Fra Angelico", Year: "1438", URL: "/artworks/annunciation", Image: "/images/annunciation.jpg"},
		RecentArtworks:  []HomeRecentArtwork{{Title: "The Birth of Venus", Artist: "Sandro Botticelli", Year: "1485", URL: "/artworks/birth-of-venus", Image: "/images/venus.jpg"}},
	})

	for _, expected := range []string{
		"Explore artists and artworks", "WORK OF THE DAY", "The Annunciation", "Fra Angelico · 1438",
		"49,610", "4,012", "128", "ARTWORKS", "ARTISTS", "SCHOOLS", "3RD–19TH", "PERIOD",
		"RECENT ADDITIONS", "ALL WORKS", "The Birth of Venus", "Sandro Botticelli · 1485",
		"Compare two works side by side", "Send any work as a postcard", "Help sustain the archive",
		`href="/artworks/annunciation"`, `href="/artworks/birth-of-venus"`,
		`src="/images/annunciation.jpg" alt="The Annunciation"`,
		`src="/images/venus.jpg" alt="The Birth of Venus" loading="lazy"`,
		`<dl`, `<dt`, `<dd`, `<h1`, `<h2 id="work-of-the-day-title"`, `<h2 id="recent-additions-title"`, `<ul`, `<li>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected rendered home to contain %q\ngot: %s", expected, rendered)
		}
	}

	for _, route := range []string{"/artists", "/artworks", "/inspire"} {
		if !strings.Contains(rendered, `href="`+route+`"`) {
			t.Errorf("home discovery links must include %q", route)
		}
	}
	if strings.Contains(rendered, "data-viewer") || strings.Contains(rendered, "data-caret-list") {
		t.Fatal("home browsing links must not mount viewer or caret-list hooks")
	}
}

func TestHomeRendersHonestEmptyWorkStates(t *testing.T) {
	rendered := renderHome(t, HomePage{ArtistCount: "0", ArtworkCount: "0", SchoolCount: "0"})

	for _, expected := range []string{
		"WORK OF THE DAY",
		"Today’s featured work is not available. Browse the collection to find an artwork.",
		"RECENT ADDITIONS",
		"No recent additions are available yet.",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected rendered empty home to contain %q\ngot: %s", expected, rendered)
		}
	}
}
