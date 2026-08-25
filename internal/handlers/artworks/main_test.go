package artworks

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
)

func TestArtworkSearchThumbnail(t *testing.T) {
	for _, test := range []struct {
		name     string
		view     string
		dualMode bool
		want     urlutils.DeliveryProfile
	}{
		{name: "grid", view: "grid", want: urlutils.DeliveryProfileCardAndArtistIndex},
		{name: "list", view: "list", want: urlutils.DeliveryProfileSearchRow},
		{name: "dual mode list", view: "list", dualMode: true, want: urlutils.DeliveryProfileCardAndArtistIndex},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := artworkSearchThumbnail(test.view, test.dualMode); got != test.want {
				t.Fatalf("artworkSearchThumbnail() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetDualModeSearchContext(t *testing.T) {
	got := getDualModeSearchContext(url.Values{
		"dual_left":            {"/artists/left-123"},
		"dual_right":           {"/artworks/right-456"},
		"dual_left_render_to":  {"left"},
		"dual_right_render_to": {"right"},
		"dual_target":          {"right"},
	})
	if got == nil {
		t.Fatal("expected Dual Mode search context")
	}

	want := &pages.ArtworkSearchDualMode{
		LeftPath:      "/artists/left-123",
		RightPath:     "/artworks/right-456",
		LeftRenderTo:  "left",
		RightRenderTo: "right",
		Target:        "right",
	}

	if *got != *want {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestGetDualModeSearchContextRequiresTarget(t *testing.T) {
	if got := getDualModeSearchContext(url.Values{"dual_left": {"/artists/left-123"}}); got != nil {
		t.Fatalf("expected no Dual Mode search context, got %#v", got)
	}
}

func TestBuildArtworkSearchPathPreservesDualModeContext(t *testing.T) {
	dualModeContext := &pages.ArtworkSearchDualMode{
		LeftPath:      "/artists/left-123",
		RightPath:     "/artworks/right-456",
		LeftRenderTo:  "left",
		RightRenderTo: "right",
		Target:        "right",
	}

	searchURL, err := url.Parse(buildArtworkSearchPath("/artworks/results", &filters{
		Title:    "A Couple",
		YearFrom: "1902",
		YearTo:   "1911",
		View:     "list",
		Page:     "2",
	}, dualModeContext))
	if err != nil {
		t.Fatalf("expected valid search URL: %v", err)
	}

	if searchURL.Path != "/artworks/results" {
		t.Fatalf("expected /artworks/results path, got %q", searchURL.Path)
	}

	for key, want := range map[string]string{
		"title":                "A Couple",
		"year_from":            "1902",
		"year_to":              "1911",
		"view":                 "list",
		"page":                 "2",
		"dual_left":            dualModeContext.LeftPath,
		"dual_right":           dualModeContext.RightPath,
		"dual_left_render_to":  dualModeContext.LeftRenderTo,
		"dual_right_render_to": dualModeContext.RightRenderTo,
		"dual_target":          dualModeContext.Target,
	} {
		if got := searchURL.Query().Get(key); got != want {
			t.Errorf("expected %s=%q, got %q", key, want, got)
		}
	}
}

func TestBuildDualModeArtworkURL(t *testing.T) {
	dualModeContext := &pages.ArtworkSearchDualMode{
		LeftPath:      "/artists/left-123",
		RightPath:     "/artworks/right-456",
		LeftRenderTo:  "left",
		RightRenderTo: "right",
		Target:        "right",
	}

	dualModeURL, err := url.Parse(buildDualModeArtworkURL("/artworks/selected-789", dualModeContext))
	if err != nil {
		t.Fatalf("expected valid Dual Mode URL: %v", err)
	}

	if dualModeURL.Path != "/dual-mode" {
		t.Fatalf("expected /dual-mode path, got %q", dualModeURL.Path)
	}

	for key, want := range map[string]string{
		"left":            dualModeContext.LeftPath,
		"right":           "/artworks/selected-789",
		"left_render_to":  dualModeContext.LeftRenderTo,
		"right_render_to": dualModeContext.RightRenderTo,
	} {
		if got := dualModeURL.Query().Get(key); got != want {
			t.Errorf("expected %s=%q, got %q", key, want, got)
		}
	}
}

func TestArtworkSearchPushURL(t *testing.T) {
	for _, test := range []struct {
		name      string
		path      string
		canonical string
		want      string
	}{
		{name: "full page with query", path: "/artworks", canonical: "/artworks?q=milkmaid", want: "/artworks?q=milkmaid"},
		{name: "full page without query", path: "/artworks", canonical: "/artworks", want: "/artworks"},
		{name: "fragment with query", path: "/artworks/results", canonical: "/artworks?q=milkmaid", want: "/artworks/results?q=milkmaid"},
		{name: "fragment without query", path: "/artworks/results", canonical: "/artworks", want: "/artworks/results"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := artworkSearchPushURL(test.path, test.canonical); got != test.want {
				t.Fatalf("artworkSearchPushURL(%q, %q) = %q, want %q", test.path, test.canonical, got, test.want)
			}
		})
	}
}

func TestBuildSortOptionsLabelsEachCriterion(t *testing.T) {
	f := buildFilters(url.Values{})
	options := buildSortOptions(f, nil)

	if len(options) != 4 {
		t.Fatalf("expected 4 sort options, got %d", len(options))
	}

	wantKeys := []string{"catalogue", "date", "artist", "title"}
	wantLabels := map[string]string{
		"catalogue": "CATALOGUE",
		"date":      "DATE",
		"artist":    "ARTIST",
		"title":     "TITLE",
	}

	for index, option := range options {
		if option.Key != wantKeys[index] {
			t.Errorf("option %d key = %q, want %q", index, option.Key, wantKeys[index])
		}
		if option.Label != wantLabels[option.Key] {
			t.Errorf("option %d label = %q, want %q", index, option.Label, wantLabels[option.Key])
		}
		if option.Active != (option.Key == "catalogue") {
			t.Errorf("option %q active = %v, want only catalogue active", option.Key, option.Active)
		}
	}
}

func TestSortDefaultDirectionIsCriterionSpecific(t *testing.T) {
	for _, test := range []struct {
		sort string
		dir  string
		want string
	}{
		{sort: "catalogue", dir: "asc", want: "↑ ARCHIVE ORDER"},
		{sort: "catalogue", dir: "desc", want: "↓ REVERSED"},
		{sort: "date", dir: "asc", want: "↑ EARLIEST FIRST"},
		{sort: "date", dir: "desc", want: "↓ LATEST FIRST"},
		{sort: "artist", dir: "asc", want: "↑ A–Z"},
		{sort: "artist", dir: "desc", want: "↓ Z–A"},
		{sort: "title", dir: "asc", want: "↑ A–Z"},
		{sort: "title", dir: "desc", want: "↓ Z–A"},
	} {
		t.Run(test.sort+"_"+test.dir, func(t *testing.T) {
			f := buildFilters(url.Values{"sort": {test.sort}, "dir": {test.dir}})
			if got := f.sortDirLabel(); got != test.want {
				t.Fatalf("sortDirLabel() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSortStringResolvesDeterministicOrder(t *testing.T) {
	for _, test := range []struct {
		sort string
		dir  string
		want string
	}{
		{sort: "catalogue", dir: "asc", want: "+source_row,+id"},
		{sort: "catalogue", dir: "desc", want: "-source_row,+id"},
		{sort: "date", dir: "asc", want: "+date_start,+id"},
		{sort: "date", dir: "desc", want: "-date_start,+id"},
		{sort: "artist", dir: "asc", want: "+author.name,+id"},
		{sort: "artist", dir: "desc", want: "-author.name,+id"},
		{sort: "title", dir: "asc", want: "+title,+id"},
		{sort: "title", dir: "desc", want: "-title,+id"},
	} {
		t.Run(test.sort+"_"+test.dir, func(t *testing.T) {
			f := buildFilters(url.Values{"sort": {test.sort}, "dir": {test.dir}})
			if got := f.sortString(); got != test.want {
				t.Fatalf("sortString() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSortPrefixOrderBy(t *testing.T) {
	for _, test := range []struct {
		sort string
		want string
	}{
		{sort: "catalogue", want: "(source_row = 0) ASC"},
		{sort: "date", want: "(date_start = 0) ASC"},
		{sort: "title", want: ""},
		{sort: "artist", want: ""},
	} {
		t.Run(test.sort, func(t *testing.T) {
			f := buildFilters(url.Values{"sort": {test.sort}})
			if got := f.sortPrefixOrderBy(); got != test.want {
				t.Fatalf("sortPrefixOrderBy() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildFiltersRejectsUnknownSort(t *testing.T) {
	f := buildFilters(url.Values{"sort": {"nonsense"}, "dir": {"nonsense"}})
	if f.Sort != "catalogue" {
		t.Errorf("Sort = %q, want catalogue", f.Sort)
	}
	if f.SortDir != "asc" {
		t.Errorf("SortDir = %q, want asc", f.SortDir)
	}
}

// Keep the legacy httptest import exercised by existing request-based tests.
var _ = httptest.NewRequest
