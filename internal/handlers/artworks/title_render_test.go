package artworks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func renderArtworkSearchForTitle(t *testing.T, app *pocketbase.PocketBase, target string, headers map[string]string) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}
	checkpoint := func(context.Context, string) error { return nil }
	if err := searchWithCheckpoint(app, event, checkpoint); err != nil {
		t.Fatalf("searchWithCheckpoint(%s) error = %v", target, err)
	}
	return recorder.Body.String()
}

func seedArtworkTitleSearch(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttitle0001", "Title Artist")
	saveSearchTaxonomy(t, app, "schools", "titleschool0001", "dutch", "Dutch")
	saveSearchLocation(t, app, "loctitlezebra01", "ZZZ Museum")
	saveSearchArtwork(t, app, searchArtworkSeed{
		id:        "worktitle000001",
		title:     "Madonna Title Work",
		authors:   []string{"artisttitle0001"},
		school:    "titleschool0001",
		location:  "loctitlezebra01",
		published: true,
	})
	return app
}

func TestArtworkSearchHTMXResponsesStartWithRootTitle(t *testing.T) {
	app := seedArtworkTitleSearch(t)
	want := "<title>“madonna” · Artworks Search - WGA</title>"

	for _, test := range []struct {
		name    string
		target  string
		headers map[string]string
	}{
		{name: "results fragment", target: "/artworks/results?q=madonna", headers: map[string]string{"HX-Request": "true"}},
		{name: "search block", target: "/artworks?q=madonna", headers: map[string]string{"HX-Request": "true", "HX-Target": "artwork-search"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := renderArtworkSearchForTitle(t, app, test.target, test.headers)
			if !strings.HasPrefix(strings.TrimSpace(body), want) {
				t.Fatalf("response does not start with %q:\n%.300s", want, body)
			}
			if count := strings.Count(body, "<title>"); count != 1 {
				t.Fatalf("response has %d <title> elements, want 1", count)
			}
		})
	}
}

func TestArtworkSearchFullPageHasOneHeadTitle(t *testing.T) {
	app := seedArtworkTitleSearch(t)

	body := renderArtworkSearchForTitle(t, app, "/artworks?q=madonna", nil)
	if count := strings.Count(body, "<title>"); count != 1 {
		t.Fatalf("full page has %d <title> elements, want 1", count)
	}
	head, _, found := strings.Cut(body, "</head>")
	if !found || !strings.Contains(head, "<title>“madonna” · Artworks Search - WGA") {
		t.Fatalf("full page head does not carry the state title:\n%.600s", head)
	}
}

func TestArtworkSearchTitleEscapesQueryMarkup(t *testing.T) {
	app := seedArtworkTitleSearch(t)

	body := renderArtworkSearchForTitle(t, app, "/artworks/results?q=%3Cb%3E", map[string]string{"HX-Request": "true"})
	if !strings.HasPrefix(strings.TrimSpace(body), "<title>“&lt;b&gt;” · Artworks Search - WGA</title>") {
		t.Fatalf("title does not escape the query:\n%.300s", body)
	}
}

func documentTitleOf(t *testing.T, body string) string {
	t.Helper()
	_, rest, found := strings.Cut(body, "<title>")
	if !found {
		t.Fatalf("response has no <title>:\n%.300s", body)
	}
	title, _, _ := strings.Cut(rest, "</title>")
	return title
}

func TestArtworkSearchTitleMatchesAcrossResponseKinds(t *testing.T) {
	app := seedArtworkTitleSearch(t)
	query := "?q=madonna&art_school=dutch&artist_id=artisttitle0001&venue=loctitlezebra01"
	want := "“madonna” · Title Artist · Artworks Search · Dutch · ZZZ Museum - WGA"

	for _, test := range []struct {
		name    string
		target  string
		headers map[string]string
	}{
		{name: "full page", target: "/artworks" + query},
		{name: "search block", target: "/artworks" + query, headers: map[string]string{"HX-Request": "true", "HX-Target": "artwork-search"}},
		{name: "results fragment", target: "/artworks/results" + query, headers: map[string]string{"HX-Request": "true"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := documentTitleOf(t, renderArtworkSearchForTitle(t, app, test.target, test.headers)); got != want {
				t.Fatalf("title = %q, want %q", got, want)
			}
		})
	}
}

func TestArtworkSearchResultsTitleLoadsOnlyActiveLabels(t *testing.T) {
	app := seedArtworkTitleSearch(t)
	request := httptest.NewRequest(http.MethodGet, "/artworks/results?art_school=dutch&artist_id=artisttitle0001", nil)
	request.Header.Set("HX-Request", "true")
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: httptest.NewRecorder()}}
	stages := []string{}
	checkpoint := func(_ context.Context, stage string) error {
		stages = append(stages, stage)
		return nil
	}

	if err := searchWithCheckpoint(app, event, checkpoint); err != nil {
		t.Fatalf("searchWithCheckpoint() error = %v", err)
	}
	want := []string{"artworks.search.count", "artworks.search.records", "artworks.search.projection", "artworks.search.artist_scope", "artworks.search.title_labels"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("started stages = %v, want %v", stages, want)
	}
	if _, ok := utils.GetCachedValue[map[string]string](app, artSchoolsCacheKey); !ok {
		t.Error("school labels were not loaded for the active school filter")
	}
	for name, key := range map[string]string{"forms": artFormsCacheKey, "types": artTypesCacheKey} {
		if _, ok := utils.GetCachedValue[map[string]string](app, key); ok {
			t.Errorf("%s labels were loaded without an active filter", name)
		}
	}
	if _, ok := utils.GetCachedValue[facetOptions](app, artPeriodsCacheKey); ok {
		t.Error("period labels were loaded without an active filter")
	}
}
