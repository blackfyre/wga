package search

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestSearchTitle(t *testing.T) {
	for _, test := range []struct {
		term string
		want string
	}{
		{term: "", want: "Search"},
		{term: "giotto", want: "“giotto” · Search"},
	} {
		if got := searchTitle(test.term); got != test.want {
			t.Errorf("searchTitle(%q) = %q, want %q", test.term, got, test.want)
		}
	}
}

func renderSearchForTitle(t *testing.T, target string, htmx bool) string {
	t.Helper()
	app := newSearchTestApp(t)
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if htmx {
		request.Header.Set("HX-Request", "true")
	}
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}
	if err := render(app, event); err != nil {
		t.Fatalf("render(%s) error = %v", target, err)
	}
	return recorder.Body.String()
}

func TestSearchResultsFragmentStartsWithRootTitle(t *testing.T) {
	body := renderSearchForTitle(t, "/search/results?q=giotto", true)

	if want := "<title>“giotto” · Search - WGA</title>"; !strings.HasPrefix(strings.TrimSpace(body), want) {
		t.Fatalf("fragment does not start with %q:\n%.300s", want, body)
	}
	if count := strings.Count(body, "<title>"); count != 1 {
		t.Fatalf("fragment has %d <title> elements, want 1", count)
	}
}

func TestSearchFullPageTitles(t *testing.T) {
	for _, test := range []struct {
		target string
		want   string
	}{
		{target: "/search", want: "<title>Search - WGA"},
		{target: "/search?q=giotto", want: "<title>“giotto” · Search - WGA"},
	} {
		body := renderSearchForTitle(t, test.target, false)
		head, _, _ := strings.Cut(body, "</head>")
		if !strings.Contains(head, test.want) {
			t.Errorf("%s head missing %q", test.target, test.want)
		}
		if count := strings.Count(body, "<title>"); count != 1 {
			t.Errorf("%s has %d <title> elements, want 1", test.target, count)
		}
	}
}

// The page body defaults hx-select to #mc-area, which the results fragment does
// not contain. The debounced form must opt out so the swap keeps its target.
func TestSearchFormSwapDoesNotInheritPageSelect(t *testing.T) {
	page := renderSearchForTitle(t, "/search?q=giotto", false)
	if want := `hx-get="/search/results" hx-target="#global-search-results" hx-swap="outerHTML" hx-select="unset"`; !strings.Contains(page, want) {
		t.Fatalf("search form missing %q", want)
	}

	fragment := renderSearchForTitle(t, "/search/results?q=giotto", true)
	if strings.Contains(fragment, `id="mc-area"`) {
		t.Fatal("results fragment unexpectedly contains #mc-area")
	}
	if !strings.Contains(fragment, `<div id="global-search-results"`) {
		t.Fatal("results fragment missing its #global-search-results swap root")
	}
}
