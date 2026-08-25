package timeline

import (
	"net/url"
	"testing"
)

func TestParseWindowDefaultsToFullRange(t *testing.T) {
	got := parseWindow(url.Values{}, 1300, 1800)
	if got.from != 1300 || got.to != 1800 {
		t.Errorf("parseWindow(defaults) = (%d, %d), want (1300, 1800)", got.from, got.to)
	}
}

func TestParseWindowAcceptsValidWindow(t *testing.T) {
	got := parseWindow(url.Values{"from": {"1400"}, "to": {"1600"}}, 1300, 1800)
	if got.from != 1400 || got.to != 1600 {
		t.Errorf("parseWindow(valid) = (%d, %d), want (1400, 1600)", got.from, got.to)
	}
}

func TestParseWindowRejectsInvalidValues(t *testing.T) {
	got := parseWindow(url.Values{"from": {"not-a-year"}, "to": {"  "}}, 1300, 1800)
	if got.from != 1300 || got.to != 1800 {
		t.Errorf("parseWindow(invalid) = (%d, %d), want full range (1300, 1800)", got.from, got.to)
	}
}

func TestParseWindowClampsOutOfRangeValues(t *testing.T) {
	got := parseWindow(url.Values{"from": {"1000"}, "to": {"9999"}}, 1300, 1800)
	if got.from != 1300 || got.to != 1800 {
		t.Errorf("parseWindow(clamped) = (%d, %d), want (1300, 1800)", got.from, got.to)
	}
}

func TestParseWindowReordersReversedBounds(t *testing.T) {
	got := parseWindow(url.Values{"from": {"1700"}, "to": {"1400"}}, 1300, 1800)
	if got.from != 1400 || got.to != 1700 {
		t.Errorf("parseWindow(reversed) = (%d, %d), want (1400, 1700)", got.from, got.to)
	}
}

func TestParseWindowWithoutKnownBounds(t *testing.T) {
	for _, bounds := range [][2]int{{0, 0}, {1300, 0}, {0, 1800}} {
		got := parseWindow(url.Values{"from": {"1400"}, "to": {"1600"}}, bounds[0], bounds[1])
		if got.from != 0 || got.to != 0 {
			t.Errorf("parseWindow(bounds %d,%d) = (%d, %d), want (0, 0)", bounds[0], bounds[1], got.from, got.to)
		}
	}
}

func TestWindowPathOmitsDefaults(t *testing.T) {
	got := (window{from: 1300, to: 1800}).path(1300, 1800)
	if got != "/timeline" {
		t.Errorf("path(full) = %q, want /timeline", got)
	}
}

func TestWindowPathEmitsSelectedWindow(t *testing.T) {
	got := (window{from: 1400, to: 1600}).path(1300, 1800)
	if got != "/timeline?from=1400&to=1600" {
		t.Errorf("path(partial) = %q", got)
	}
}

func TestWindowPathEmitsPartialDefault(t *testing.T) {
	got := (window{from: 1300, to: 1600}).path(1300, 1800)
	if got != "/timeline?to=1600" {
		t.Errorf("path(from default) = %q, want /timeline?to=1600", got)
	}
}

func TestWindowPathEmitsPage(t *testing.T) {
	got := (window{from: 1300, to: 1800, page: 3}).path(1300, 1800)
	if got != "/timeline?page=3" {
		t.Errorf("path(page) = %q, want /timeline?page=3", got)
	}
}

func TestWindowPathOmitsFirstPage(t *testing.T) {
	if got := (window{from: 1400, to: 1600, page: 1}).path(1300, 1800); got != "/timeline?from=1400&to=1600" {
		t.Errorf("path(page=1) = %q, want /timeline?from=1400&to=1600", got)
	}
	if got := (window{from: 1300, to: 1800, page: 1}).path(1300, 1800); got != "/timeline" {
		t.Errorf("path(page=1 full) = %q, want /timeline", got)
	}
}

func TestWindowWithPage(t *testing.T) {
	base := window{from: 1400, to: 1600}
	got := base.withPage(2)
	if got.from != 1400 || got.to != 1600 || got.page != 2 {
		t.Errorf("withPage = %#v, want from 1400 to 1600 page 2", got)
	}
	if base.page != 0 {
		t.Errorf("withPage mutated the receiver: %#v", base)
	}
}

func TestParsePage(t *testing.T) {
	for input, want := range map[string]int{"": 1, "0": 1, "-3": 1, "junk": 1, "1": 1, "42": 42} {
		if got := parsePage(url.Values{"page": {input}}); got != want {
			t.Errorf("parsePage(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestEffectiveEnd(t *testing.T) {
	if got := effectiveEnd(1500, 1600); got != 1600 {
		t.Errorf("effectiveEnd(1500, 1600) = %d", got)
	}
	if got := effectiveEnd(1500, 0); got != 1500 {
		t.Errorf("effectiveEnd(1500, 0) = %d, want 1500", got)
	}
}

func TestOverlaps(t *testing.T) {
	cases := []struct {
		name  string
		start int
		end   int
		from  int
		to    int
		want  bool
	}{
		{"fully inside", 1500, 1600, 1400, 1700, true},
		{"start before window, end inside", 1300, 1450, 1400, 1700, true},
		{"start inside, end after window", 1600, 1900, 1400, 1700, true},
		{"single year inside", 1500, 1500, 1400, 1700, true},
		{"single year without end inside", 1500, 0, 1400, 1700, true},
		{"entirely before", 1200, 1300, 1400, 1700, false},
		{"entirely after", 1800, 1900, 1400, 1700, false},
		{"unknown start", 0, 0, 1400, 1700, false},
		{"boundary start equals to", 1700, 1700, 1400, 1700, true},
		{"boundary end equals from", 1400, 1400, 1400, 1700, true},
	}

	for _, tc := range cases {
		if got := overlaps(tc.start, tc.end, tc.from, tc.to); got != tc.want {
			t.Errorf("overlaps(%d, %d, %d, %d) = %t, want %t (%s)", tc.start, tc.end, tc.from, tc.to, got, tc.want, tc.name)
		}
	}
}

func TestClipSpan(t *testing.T) {
	cases := []struct {
		start, end, from, to int
		wantFrom, wantTo     int
		wantOK               bool
	}{
		{1400, 1600, 1450, 1550, 1450, 1550, true},
		{100, 500, 1450, 1550, 0, 0, false},
		{1500, 2000, 1450, 1550, 1500, 1550, true},
		{1200, 1500, 1450, 1550, 1450, 1500, true},
		{1200, 0, 1450, 1550, 0, 0, false},
	}

	for _, tc := range cases {
		gotFrom, gotTo, gotOK := clipSpan(tc.start, tc.end, tc.from, tc.to)
		if gotFrom != tc.wantFrom || gotTo != tc.wantTo || gotOK != tc.wantOK {
			t.Errorf("clipSpan(%d, %d, %d, %d) = (%d, %d, %t), want (%d, %d, %t)",
				tc.start, tc.end, tc.from, tc.to, gotFrom, gotTo, gotOK, tc.wantFrom, tc.wantTo, tc.wantOK)
		}
	}
}

func TestDecadeBins(t *testing.T) {
	got := decadeBins(1401, 1555)
	want := []int{1400, 1410, 1420, 1430, 1440, 1450, 1460, 1470, 1480, 1490, 1500, 1510, 1520, 1530, 1540, 1550}
	if len(got) != len(want) {
		t.Fatalf("decadeBins(1401, 1555) length = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("decadeBins[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestLeftPct(t *testing.T) {
	if got := leftPct(1500, 1400, 1600); !almostEqual(got, 50) {
		t.Errorf("leftPct(1500, 1400, 1600) = %v, want 50", got)
	}
	if got := leftPct(1400, 1400, 1600); got != 0 {
		t.Errorf("leftPct(min) = %v, want 0", got)
	}
	if got := leftPct(1600, 1400, 1600); got != 100 {
		t.Errorf("leftPct(max) = %v, want 100", got)
	}
	if got := leftPct(9999, 1400, 1600); got != 100 {
		t.Errorf("leftPct(clamped high) = %v, want 100", got)
	}
	if got := leftPct(1500, 1500, 1500); got != 0 {
		t.Errorf("leftPct(single year) = %v, want 0", got)
	}
}

func TestWidthPct(t *testing.T) {
	if got := widthPct(1400, 1600, 1400, 1600); got != 100 {
		t.Errorf("widthPct(full) = %v, want 100", got)
	}
	if got := widthPct(1400, 1400, 1400, 1600); !almostEqual(got, 100.0/201.0) {
		t.Errorf("widthPct(single year of 201) = %v, want %v", got, 100.0/201.0)
	}
	if got := widthPct(1450, 1550, 1400, 1600); !almostEqual(got, 101.0/201.0*100) {
		t.Errorf("widthPct(clipped middle) = %v, want %v", got, 101.0/201.0*100)
	}
}

func almostEqual(a float64, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}

	return diff < 1e-9
}

func TestFormatRange(t *testing.T) {
	if got := formatRange(1300, 1800); got != "1300–1800" {
		t.Errorf("formatRange(1300, 1800) = %q", got)
	}
	if got := formatRange(1500, 1500); got != "1500" {
		t.Errorf("formatRange(1500, 1500) = %q, want 1500", got)
	}
}

func TestSpanYears(t *testing.T) {
	if got := spanYears(1300, 1800); got != 501 {
		t.Errorf("spanYears(1300, 1800) = %d, want 501", got)
	}
	if got := spanYears(1500, 1500); got != 1 {
		t.Errorf("spanYears(1500, 1500) = %d, want 1", got)
	}
	if got := spanYears(1800, 1300); got != 0 {
		t.Errorf("spanYears(reversed) = %d, want 0", got)
	}
}

func TestFormatDateLabel(t *testing.T) {
	cases := []struct {
		start, end int
		circa      bool
		qualifier  string
		want       string
	}{
		{1500, 1600, false, "", "1500–1600"},
		{1500, 0, false, "", "1500"},
		{200, 200, true, "", "circa 200"},
		{150, 0, false, "after", "after 150"},
		{150, 200, true, "around", "around 150–200"},
		{0, 200, false, "", ""},
	}

	for _, tc := range cases {
		if got := formatDateLabel(tc.start, tc.end, tc.circa, tc.qualifier); got != tc.want {
			t.Errorf("formatDateLabel(%d, %d, %t, %q) = %q, want %q", tc.start, tc.end, tc.circa, tc.qualifier, got, tc.want)
		}
	}
}
