package timeline

import (
	"net/url"
	"strconv"
	"strings"
)

// window is a validated, clamped, inclusive year range selected by a visitor,
// plus the selected works page.
type window struct {
	from int
	to   int
	page int
}

// parseWindow reads the `from` and `to` query values and clamps them into the
// published date range [min, max]. Empty or non-numeric values fall back to the
// full range; out-of-range values are clamped; reversed bounds are reordered.
// When min/max indicate there is no chronology (both are zero), the window is
// (0, 0) and the caller renders the honest empty state.
func parseWindow(values url.Values, min int, max int) window {
	if min <= 0 || max <= 0 {
		return window{}
	}

	from := parseYear(values.Get("from"), min)
	to := parseYear(values.Get("to"), max)

	from = clampYear(from, min, max)
	to = clampYear(to, min, max)

	if from > to {
		from, to = to, from
	}

	return window{from: from, to: to}
}

// parsePage returns a positive, one-based works page number, defaulting to 1
// for empty or invalid input.
func parsePage(values url.Values) int {
	value, err := strconv.Atoi(strings.TrimSpace(values.Get("page")))
	if err != nil || value < 1 {
		return 1
	}

	return value
}

// withPage returns a copy of the window with the given page number.
func (w window) withPage(page int) window {
	w.page = page

	return w
}

// parseYear returns the fallback for empty or non-numeric input.
func parseYear(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}

	return value
}

func clampYear(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}

	return value
}

// path returns the canonical /timeline URL for the window, omitting the query
// when the window covers the full published range and the page is the first.
func (w window) path(min int, max int) string {
	parts := []string{}
	if w.from != min {
		parts = append(parts, "from="+strconv.Itoa(w.from))
	}
	if w.to != max {
		parts = append(parts, "to="+strconv.Itoa(w.to))
	}
	if w.page > 1 {
		parts = append(parts, "page="+strconv.Itoa(w.page))
	}

	if len(parts) == 0 {
		return "/timeline"
	}

	return "/timeline?" + strings.Join(parts, "&")
}

// effectiveEnd returns the artwork's end year, treating an absent end as a
// single-year span equal to its start.
func effectiveEnd(start int, end int) int {
	if end > 0 {
		return end
	}

	return start
}

// overlaps reports whether the inclusive span [start, end] intersects the
// inclusive window [from, to]. Spans with an unknown start never overlap.
func overlaps(start int, end int, from int, to int) bool {
	if start <= 0 {
		return false
	}

	return start <= to && effectiveEnd(start, end) >= from
}

// clipSpan clips the inclusive span [start, end] to the inclusive window
// [from, to]. It returns false when the span does not intersect the window.
func clipSpan(start int, end int, from int, to int) (int, int, bool) {
	if !overlaps(start, end, from, to) {
		return 0, 0, false
	}

	clipStart := start
	if clipStart < from {
		clipStart = from
	}
	clipEnd := effectiveEnd(start, end)
	if clipEnd > to {
		clipEnd = to
	}

	return clipStart, clipEnd, true
}

// decadeFloor returns the decade-aligned start year containing the given year.
func decadeFloor(year int) int {
	return year - (year % 10)
}

// decadeBins returns the sorted decade start years whose ten-year bins intersect
// the inclusive window [from, to].
func decadeBins(from int, to int) []int {
	start := decadeFloor(from)
	end := decadeFloor(to)

	bins := []int{}
	for decade := start; decade <= end; decade += 10 {
		bins = append(bins, decade)
	}

	return bins
}

// leftPct maps a year inside [from, to] onto a percentage position. The result
// is clamped to [0, 100] so edge input can never escape the plot.
func leftPct(year int, from int, to int) float64 {
	if to <= from {
		return 0
	}

	position := float64(year-from) / float64(to-from) * 100
	if position < 0 {
		return 0
	}
	if position > 100 {
		return 100
	}

	return position
}

// widthPct maps the clipped inclusive span [clipStart, clipEnd] inside
// [from, to] onto a percentage width, clamped to [0, 100].
func widthPct(clipStart int, clipEnd int, from int, to int) float64 {
	if to < from {
		return 0
	}

	width := float64(clipEnd-clipStart+1) / float64(to-from+1) * 100
	if width < 0 {
		return 0
	}
	if width > 100 {
		return 100
	}

	return width
}

// formatRange renders an inclusive year range, collapsing a single year.
func formatRange(from int, to int) string {
	if from == to {
		return strconv.Itoa(from)
	}

	return strconv.Itoa(from) + "–" + strconv.Itoa(to)
}

// spanYears returns the number of inclusive years in [from, to].
func spanYears(from int, to int) int {
	if to < from {
		return 0
	}

	return to - from + 1
}

// formatDateLabel builds the human creation-date label for an artwork from its
// date-span fields. A qualifier ("after", "before", "around") takes precedence
// over the circa flag; a single year collapses; an unknown start yields "".
func formatDateLabel(start int, end int, circa bool, qualifier string) string {
	if start <= 0 {
		return ""
	}

	core := formatRange(start, effectiveEnd(start, end))

	qualifier = strings.TrimSpace(qualifier)
	if qualifier != "" {
		return qualifier + " " + core
	}
	if circa {
		return "circa " + core
	}

	return core
}
