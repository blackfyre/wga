package utils_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func titleLength(title string) int {
	return utf8.RuneCountInString(title + templutils.TitleSuffix)
}

func TestFitTitleKeepsShortTitlesInLongForm(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: "madonna", Role: templutils.TitleLead, Quoted: true},
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
		{Key: "school", Long: "Florentine", Short: "Flor.", Role: templutils.TitleFilter},
		{Key: "position", Long: "p. 3/41", Role: templutils.TitlePosition},
	}
	steps := []templutils.TitleStep{
		{Kind: templutils.TitleRemove, Key: "position"},
		{Kind: templutils.TitleShorten, Key: "school"},
	}

	got := templutils.FitTitle(parts, steps)
	if want := "“madonna” · Artworks Search · Florentine · p. 3/41"; got != want {
		t.Fatalf("FitTitle = %q, want %q", got, want)
	}
}

func TestFitTitleSkipsEmptyParts(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: " ", Role: templutils.TitleLead, Quoted: true},
		{Key: "page", Long: "Artists", Role: templutils.TitlePageName},
	}

	if got := templutils.FitTitle(parts, nil); got != "Artists" {
		t.Fatalf("FitTitle = %q, want %q", got, "Artists")
	}
}

func TestFitTitleAppliesStepsInOrderAndStopsWhenFitting(t *testing.T) {
	filler := strings.Repeat("x", 30)
	parts := []templutils.TitlePart{
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
		{Key: "school", Long: filler, Short: "short", Role: templutils.TitleFilter},
		{Key: "form", Long: filler, Short: "brief", Role: templutils.TitleFilter},
		{Key: "position", Long: "p. 3/41", Role: templutils.TitlePosition},
	}
	steps := []templutils.TitleStep{
		{Kind: templutils.TitleRemove, Key: "position"},
		{Kind: templutils.TitleShorten, Key: "form"},
		{Kind: templutils.TitleShorten, Key: "school"},
	}

	got := templutils.FitTitle(parts, steps)
	want := "Artworks Search · " + filler + " · brief"
	if got != want {
		t.Fatalf("FitTitle = %q, want %q", got, want)
	}
	if titleLength(got) > templutils.TitleMaxLength {
		t.Fatalf("title length %d exceeds %d", titleLength(got), templutils.TitleMaxLength)
	}
}

func TestFitTitleCountsRemovedFilters(t *testing.T) {
	filler := strings.Repeat("y", 30)
	parts := []templutils.TitlePart{
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
		{Key: "school", Long: "Florentine", Role: templutils.TitleFilter},
		{Key: "form", Long: filler, Role: templutils.TitleFilter},
		{Key: "type", Long: filler, Role: templutils.TitleFilter},
		{Key: "position", Long: "p. 2/7", Role: templutils.TitlePosition},
	}
	steps := []templutils.TitleStep{
		{Kind: templutils.TitleRemove, Key: "type"},
		{Kind: templutils.TitleRemove, Key: "form"},
	}

	got := templutils.FitTitle(parts, steps)
	if want := "Artworks Search · Florentine · +2 · p. 2/7"; got != want {
		t.Fatalf("FitTitle = %q, want %q", got, want)
	}
}

func TestFitTitleNeverRemovesPageNameOrLead(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: "madonna", Role: templutils.TitleLead, Quoted: true},
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
		{Key: "school", Long: strings.Repeat("z", 60), Role: templutils.TitleFilter},
	}
	steps := []templutils.TitleStep{
		{Kind: templutils.TitleRemove, Key: "page"},
		{Kind: templutils.TitleRemove, Key: "q"},
		{Kind: templutils.TitleTruncate, Key: "page", Limit: 3},
	}

	got := templutils.FitTitle(parts, steps)
	if want := "“madonna” · Artworks Search · +1"; got != want {
		t.Fatalf("FitTitle = %q, want %q", got, want)
	}
}

func TestFitTitleTruncatesLeadInsideQuotes(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: strings.Repeat("a", 70), Role: templutils.TitleLead, Quoted: true},
		{Key: "page", Long: "Search", Role: templutils.TitlePageName},
	}
	steps := []templutils.TitleStep{{Kind: templutils.TitleTruncate, Key: "q", Limit: 20}}

	got := templutils.FitTitle(parts, steps)
	if want := "“" + strings.Repeat("a", 20) + "…” · Search"; got != want {
		t.Fatalf("FitTitle = %q, want %q", got, want)
	}
}

func TestFitTitleCountsCodePoints(t *testing.T) {
	// 60 multi-byte code points fit although they exceed 80 bytes.
	lead := strings.Repeat("é", 60)
	parts := []templutils.TitlePart{
		{Key: "q", Long: lead, Role: templutils.TitleLead, Quoted: true},
		{Key: "page", Long: "Search", Role: templutils.TitlePageName},
	}
	steps := []templutils.TitleStep{{Kind: templutils.TitleTruncate, Key: "q", Limit: 20}}

	got := templutils.FitTitle(parts, steps)
	if want := "“" + lead + "” · Search"; got != want {
		t.Fatalf("FitTitle = %q, want %q", got, want)
	}
}

func TestFitTitleFinalGuardFitsOversizedLead(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "artist", Long: strings.Repeat("Ö", 120), Role: templutils.TitleLead},
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
		{Key: "school", Long: "Florentine", Role: templutils.TitleFilter},
		{Key: "position", Long: "p. 3/41", Role: templutils.TitlePosition},
	}

	got := templutils.FitTitle(parts, nil)
	if length := titleLength(got); length != templutils.TitleMaxLength {
		t.Fatalf("title length %d, want exactly %d: %q", length, templutils.TitleMaxLength, got)
	}
	if !strings.HasSuffix(got, "… · Artworks Search · +1") {
		t.Fatalf("FitTitle = %q, want truncated lead, page name and filter count", got)
	}
}

func TestFitTitleIsDeterministic(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: strings.Repeat("b", 90), Role: templutils.TitleLead, Quoted: true},
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
		{Key: "school", Long: "Florentine", Role: templutils.TitleFilter},
	}
	first := templutils.FitTitle(parts, nil)
	for range 5 {
		if got := templutils.FitTitle(parts, nil); got != first {
			t.Fatalf("FitTitle = %q, want stable %q", got, first)
		}
	}
}

func TestTitlePagePosition(t *testing.T) {
	for _, test := range []struct {
		page, pageCount int
		want            string
	}{
		{page: 1, pageCount: 7, want: ""},
		{page: 1, pageCount: 0, want: ""},
		{page: 2, pageCount: 7, want: "p. 2/7"},
		{page: 41, pageCount: 41, want: "p. 41/41"},
	} {
		if got := templutils.TitlePagePosition(test.page, test.pageCount); got != test.want {
			t.Errorf("TitlePagePosition(%d, %d) = %q, want %q", test.page, test.pageCount, got, test.want)
		}
	}
}

func TestFitTitleFinalGuardTruncatesLastLeadFirst(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: "madonna", Role: templutils.TitleLead, Quoted: true},
		{Key: "scope", Long: strings.Repeat("S", 120), Role: templutils.TitleLead},
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
	}

	got := templutils.FitTitle(parts, nil)
	if length := titleLength(got); length != templutils.TitleMaxLength {
		t.Fatalf("title length %d, want exactly %d: %q", length, templutils.TitleMaxLength, got)
	}
	if !strings.HasPrefix(got, "“madonna” · SSS") {
		t.Fatalf("FitTitle = %q, want the first lead intact", got)
	}
}

func TestFitTitleFinalGuardFitsTwoOversizedLeads(t *testing.T) {
	parts := []templutils.TitlePart{
		{Key: "q", Long: strings.Repeat("q", 120), Role: templutils.TitleLead, Quoted: true},
		{Key: "scope", Long: strings.Repeat("s", 120), Role: templutils.TitleLead},
		{Key: "page", Long: "Artworks Search", Role: templutils.TitlePageName},
	}

	got := templutils.FitTitle(parts, nil)
	if length := titleLength(got); length > templutils.TitleMaxLength {
		t.Fatalf("title length %d exceeds %d: %q", length, templutils.TitleMaxLength, got)
	}
	if !strings.HasSuffix(got, " · Artworks Search") {
		t.Fatalf("FitTitle = %q, want page name kept", got)
	}
}
