package artworks

import (
	"strings"
	"testing"
	"unicode/utf8"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestArtworkSearchTitle(t *testing.T) {
	labels := artworkSearchTitleLabels{
		schools: map[string]string{"florentine": "Florentine", "venetian": "Venetian"},
		forms:   map[string]string{"painting": "painting"},
		types:   map[string]string{"religious": "religious"},
		periods: map[string]string{"period00001": "Early Renaissance"},
	}
	for _, test := range []struct {
		name      string
		filters   filters
		labels    artworkSearchTitleLabels
		page      int
		pageCount int
		want      string
	}{
		{name: "unfiltered first page", page: 1, pageCount: 41, want: "Artworks Search"},
		{
			name:    "spec example",
			filters: filters{Query: "madonna", SchoolValues: []string{"florentine"}},
			labels:  labels, page: 3, pageCount: 41,
			want: "“madonna” · Artworks Search · Florentine · p. 3/41",
		},
		{
			name:    "rail order with normal-case labels",
			filters: filters{TechniqueString: "fresco", SchoolValues: []string{"florentine", "venetian"}, ArtFormString: "painting"},
			labels:  labels, page: 1, pageCount: 2,
			want: "Artworks Search · technique “fresco” · Florentine, Venetian · painting",
		},
		{
			name:    "period collection and year",
			filters: filters{PeriodString: "period00001", VenueString: "Uffizi, Florence", YearFrom: "1400"},
			labels:  labels, page: 2, pageCount: 3,
			want: "Artworks Search · Early Renaissance · Uffizi · 1400–1900 · p. 2/3",
		},
		{
			name:    "unresolved values are left out",
			filters: filters{SchoolValues: []string{"unknown-school", "florentine"}, PeriodString: "k3j2h1g0f9e8d7c"},
			labels:  labels, page: 1, pageCount: 1,
			want: "Artworks Search · Florentine",
		},
		{
			name:    "exact artist scope replaces the artist text filter",
			filters: filters{ArtistID: "artist000000001", ArtistString: "giotto"},
			labels:  artworkSearchTitleLabels{artistScope: "GIOTTO di Bondone"}, page: 1, pageCount: 1,
			want: "GIOTTO di Bondone · Artworks Search",
		},
		{
			name:    "presentation state is excluded",
			filters: filters{View: "list", Sort: "date", SortDir: "desc"},
			page:    1, pageCount: 5,
			want: "Artworks Search",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := artworkSearchTitle(&test.filters, test.labels, test.page, test.pageCount)
			if got != test.want {
				t.Fatalf("artworkSearchTitle() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestArtworkSearchTitleReducesLongState(t *testing.T) {
	f := filters{
		Query:           "madonna and child with saints",
		TechniqueString: "tempera on panel",
		SchoolValues:    []string{"florentine", "venetian"},
		ArtFormString:   "painting",
		YearFrom:        "1300",
		YearTo:          "1450",
	}
	labels := artworkSearchTitleLabels{
		artistScope: "LIPPI, Fra Filippo di Tommaso di Lorenzo",
		schools:     map[string]string{"florentine": "Florentine", "venetian": "Venetian"},
	}

	got := artworkSearchTitle(&f, labels, 3, 41)
	if length := utf8.RuneCountInString(got + tmplUtils.TitleSuffix); length > tmplUtils.TitleMaxLength {
		t.Fatalf("title length %d exceeds %d: %q", length, tmplUtils.TitleMaxLength, got)
	}
	if !strings.HasPrefix(got, "“madonna") || !strings.Contains(got, "Artworks Search") || strings.Contains(got, "p. 3/41") {
		t.Fatalf("artworkSearchTitle() = %q, want query first, page name kept and position removed", got)
	}
	if !strings.Contains(got, "+") {
		t.Fatalf("artworkSearchTitle() = %q, want removed filters counted", got)
	}
}
