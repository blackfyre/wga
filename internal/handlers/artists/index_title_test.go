package artists

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func titleTestView(mutate func(*pages.ArtistsView)) pages.ArtistsView {
	view := pages.ArtistsView{
		Schools:        []dto.SelectOption{{Label: "ALL SCHOOLS", Value: "", Selected: true}, {Label: "Italian", Value: "italian"}},
		Periods:        []dto.SelectOption{{Label: "ALL PERIODS", Value: "", Selected: true}, {Label: "Early Renaissance", Value: "early-renaissance"}},
		HasBirthBounds: true,
		BornRange:      dto.RangeField{FromValue: 1100, ToValue: 1900, Min: 1100, Max: 1900},
		Page:           1,
		PageCount:      7,
	}
	if mutate != nil {
		mutate(&view)
	}
	return view
}

func selectOption(options []dto.SelectOption, value string) {
	for index := range options {
		options[index].Selected = options[index].Value == value
	}
}

func TestArtistIndexTitle(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*pages.ArtistsView)
		want   string
	}{
		{name: "unfiltered first page", want: "Artists"},
		{
			name:   "later page adds position",
			mutate: func(view *pages.ArtistsView) { view.Page = 2 },
			want:   "Artists · p. 2/7",
		},
		{
			name: "letter and school on first page",
			mutate: func(view *pages.ArtistsView) {
				view.SelectedLetter = "B"
				selectOption(view.Schools, "italian")
			},
			want: "Artists · Letter B · Italian",
		},
		{
			name: "all state in control order drops position first",
			mutate: func(view *pages.ArtistsView) {
				view.NameQuery = "van"
				view.SelectedLetter = "B"
				selectOption(view.Schools, "italian")
				selectOption(view.Periods, "early-renaissance")
				view.BornRange.FromValue = 1400
				view.BornRange.ToValue = 1500
				view.Page = 3
			},
			want: "“van” · Artists · Letter B · Italian · Early Renaissance · born 1400–1500",
		},
		{
			name: "presentation state is excluded",
			mutate: func(view *pages.ArtistsView) {
				view.View = "list"
				view.Sort = "za"
				view.SortLabel = "Z–A"
			},
			want: "Artists",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := artistIndexTitle(titleTestView(test.mutate)); got != test.want {
				t.Fatalf("artistIndexTitle() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestArtistIndexTitleFitsLongQuery(t *testing.T) {
	view := titleTestView(func(view *pages.ArtistsView) {
		view.NameQuery = strings.Repeat("rembrandt ", 8)
		view.SelectedLetter = "R"
		selectOption(view.Schools, "italian")
		view.Page = 2
	})

	got := artistIndexTitle(view)
	if length := utf8.RuneCountInString(got + tmplUtils.TitleSuffix); length > tmplUtils.TitleMaxLength {
		t.Fatalf("title length %d exceeds %d: %q", length, tmplUtils.TitleMaxLength, got)
	}
	if !strings.HasPrefix(got, "“rembrandt") || !strings.Contains(got, " · Artists") {
		t.Fatalf("artistIndexTitle() = %q, want truncated query then page name", got)
	}
}
