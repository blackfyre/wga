package timeline

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/constants"
	wgaurl "github.com/blackfyre/wga/internal/utils/url"
)

const (
	worksPageSize = 8
	markCap       = 48
)

// timelineStore is the persistence boundary the timeline workflow depends on.
// It is implemented by repository and by test doubles so the workflow can prove
// which reads it performs without reaching the database.
type timelineStore interface {
	listPeriods() ([]artPeriod, error)
	artworkBounds() (int, int, error)
	countWorks(from int, to int) (int, error)
	dateSpans(from int, to int) ([]dateSpan, error)
	listWorks(from int, to int, limit int, offset int) ([]artworkRow, error)
}

// buildTimelineView loads the approved chronology and assembles the page-owned
// view plus its canonical URL. It is the timeline capability's workflow: all
// parsing, clamping, and projection decisions live here rather than in the
// request handler (ADR 0007).
func buildTimelineView(store timelineStore, values url.Values) (pages.TimelineView, string, error) {
	periods, err := store.listPeriods()
	if err != nil {
		return pages.TimelineView{}, "", err
	}

	artMin, artMax, err := store.artworkBounds()
	if err != nil {
		return pages.TimelineView{}, "", err
	}

	periodMin, periodMax := periodBounds(periods)
	min, max := unionBounds(artMin, artMax, periodMin, periodMax)

	win := parseWindow(values, min, max)
	win.page = parsePage(values)

	view := pages.TimelineView{
		From: win.from,
		To:   win.to,
	}

	if min <= 0 || max <= 0 {
		// No chronology at all: there is no page to carry, so the canonical URL
		// must never include one.
		win.page = 1
		view.HasRange = false
		view.Page = win.page
		return view, win.path(min, max), nil
	}

	view.HasRange = true

	total, err := store.countWorks(win.from, win.to)
	if err != nil {
		return pages.TimelineView{}, "", err
	}

	spans, err := store.dateSpans(win.from, win.to)
	if err != nil {
		return pages.TimelineView{}, "", err
	}

	markRows, err := store.listWorks(win.from, win.to, markCap, 0)
	if err != nil {
		return pages.TimelineView{}, "", err
	}

	pageCount := 0
	if total > 0 {
		pageCount = (total + worksPageSize - 1) / worksPageSize
		if win.page < 1 {
			win.page = 1
		}
		if win.page > pageCount {
			win.page = pageCount
		}
	} else {
		// A valid window with no matching works has no pages: normalise the page
		// before the canonical URL is built and skip the paged query entirely.
		win.page = 1
	}

	var workRows []artworkRow
	if total > 0 {
		workRows, err = store.listWorks(win.from, win.to, worksPageSize, (win.page-1)*worksPageSize)
		if err != nil {
			return pages.TimelineView{}, "", err
		}
	}

	view.RangeLabel = formatRange(win.from, win.to)
	view.SpanLabel = fmt.Sprintf("%d YEARS", spanYears(win.from, win.to))
	view.HitLabel = fmt.Sprintf("%d WORKS", total)

	view.Presets = buildPresets(periods, win, min, max)
	view.WindowRange = dto.RangeField{
		Label:     "SET A WINDOW",
		FromID:    "timeline-from",
		FromName:  "from",
		FromValue: win.from,
		ToID:      "timeline-to",
		ToName:    "to",
		ToValue:   win.to,
		Min:       min,
		Max:       max,
		Step:      1,
		Brush:     true,
	}

	view.Density = densityBins(spans, win.from, win.to)
	view.Bands = buildBands(periods, win.from, win.to)
	view.Marks = buildMarks(markRows, win.from, win.to)
	view.Works = buildWorks(workRows, win.from, win.to)

	view.TotalWorks = total
	view.Page = win.page
	view.PageCount = pageCount
	view.WorksRangeLabel = worksRangeLabel(total, win.page, worksPageSize)
	view.PrevUrl = win.withPage(win.page-1).path(min, max)
	view.NextUrl = win.withPage(win.page+1).path(min, max)

	view.HasWorks = total > 0
	view.MarksNote = marksNote(total)

	return view, win.path(min, max), nil
}

// marksNote discloses the bounded mark overview truthfully: when more works
// overlap the window than the plot can hold, it states the exact cap and the
// deterministic order, otherwise it is empty.
func marksNote(total int) string {
	if total <= markCap {
		return ""
	}

	return fmt.Sprintf("MARKS SHOW THE FIRST %d OF %d WORKS, ORDERED BY DATE", markCap, total)
}

// periodBounds returns the inclusive [min, max] over valid art-period spans, or
// (0, 0) when there are no valid periods.
func periodBounds(periods []artPeriod) (int, int) {
	min, max := 0, 0
	for _, period := range periods {
		if period.start <= 0 || period.end <= 0 || period.start > period.end {
			continue
		}
		if min == 0 || period.start < min {
			min = period.start
		}
		if period.end > max {
			max = period.end
		}
	}

	return min, max
}

// unionBounds merges the artwork and art-period outer bounds so period presets
// stay reachable even when no artwork is dated inside them. Either source may be
// empty (zero); the result is (0, 0) only when both are.
func unionBounds(artMin int, artMax int, periodMin int, periodMax int) (int, int) {
	min := 0
	for _, value := range []int{artMin, periodMin} {
		if value > 0 && (min == 0 || value < min) {
			min = value
		}
	}

	max := artMax
	if periodMax > max {
		max = periodMax
	}

	return min, max
}

// buildPresets returns a FULL chip plus one chip per approved art period, each
// an ordinary link that opens that named span without JavaScript.
func buildPresets(periods []artPeriod, win window, min int, max int) []dto.NavChip {
	chips := make([]dto.NavChip, 0, len(periods)+1)

	fullActive := win.from == min && win.to == max
	chips = append(chips, dto.NavChip{
		Label:  "FULL RANGE",
		Href:   "/timeline",
		Active: fullActive,
	})

	for _, period := range periods {
		if period.start <= 0 || period.end <= 0 || period.start > period.end {
			continue
		}
		chips = append(chips, dto.NavChip{
			Label:  period.name,
			Href:   periodPath(period.start, period.end, min, max),
			Active: win.from == period.start && win.to == period.end,
		})
	}

	return chips
}

// periodPath returns the canonical /timeline URL for an art-period span.
func periodPath(start int, end int, min int, max int) string {
	return (window{from: start, to: end}).path(min, max)
}

// densityBins buckets the overlapping published spans into decade bins clipped
// to the selected window, using the same inclusive overlap predicate as marks
// and counts, and normalises each bar height against the densest decade.
func densityBins(spans []dateSpan, from int, to int) []pages.TimelineDensityBar {
	bins := decadeBins(from, to)
	if len(bins) == 0 {
		return nil
	}

	counts := make([]int, len(bins))
	for i, decade := range bins {
		binStart := decade
		if binStart < from {
			binStart = from
		}
		binEnd := decade + 9
		if binEnd > to {
			binEnd = to
		}

		for _, span := range spans {
			if span.Start > binEnd {
				// spans are ordered by start year, so later spans cannot overlap.
				break
			}
			if overlaps(span.Start, span.End, binStart, binEnd) {
				counts[i]++
			}
		}
	}

	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	bars := make([]pages.TimelineDensityBar, 0, len(bins))
	for i, decade := range bins {
		bars = append(bars, pages.TimelineDensityBar{
			Decade:    strconv.Itoa(decade),
			Count:     counts[i],
			HeightPct: counts[i] * 100 / maxCount,
		})
	}

	return bars
}

// buildBands clips each approved art period to the window, in order. Periods
// with missing or reversed spans are not meaningful and are skipped.
func buildBands(periods []artPeriod, from int, to int) []pages.TimelineBand {
	bands := make([]pages.TimelineBand, 0, len(periods))
	for _, period := range periods {
		if period.start <= 0 || period.end <= 0 || period.start > period.end {
			continue
		}
		clipStart, clipEnd, ok := clipSpan(period.start, period.end, from, to)
		if !ok {
			continue
		}
		bands = append(bands, pages.TimelineBand{
			Name:        period.name,
			SpanLabel:   formatRange(clipStart, clipEnd),
			LeftPct:     leftPct(clipStart, from, to),
			WidthPct:    widthPct(clipStart, clipEnd, from, to),
			Description: period.description,
		})
	}

	return bands
}

// buildMarks positions each bounded work row on the window lane.
func buildMarks(rows []artworkRow, from int, to int) []pages.TimelineMark {
	marks := make([]pages.TimelineMark, 0, len(rows))
	for _, row := range rows {
		marks = append(marks, pages.TimelineMark{
			Href:    artworkURL(row),
			Label:   markLabel(row),
			LeftPct: leftPct(row.DateStart, from, to),
		})
	}

	return marks
}

// buildWorks projects one already-bounded works page into the card panel.
func buildWorks(rows []artworkRow, from int, to int) []dto.Work {
	works := make([]dto.Work, 0, len(rows))
	for _, row := range rows {
		works = append(works, dto.Work{
			URL:       artworkURL(row),
			ImageURL:  artworkImageURL(row),
			ArtworkID: row.ID,
			Title:     row.Title,
			Artist:    row.ArtistName,
			Metadata:  formatDateLabel(row.DateStart, row.DateEnd, row.IsCirca != 0, row.Qualifier),
		})
	}

	return works
}

// worksRangeLabel renders an explicit "WORKS 1–8 OF 20" style readout, or the
// plain total when everything fits on one page.
func worksRangeLabel(total int, page int, pageSize int) string {
	if total == 0 {
		return "0 WORKS"
	}
	if total <= pageSize {
		return fmt.Sprintf("%d WORKS", total)
	}

	first := (page-1)*pageSize + 1
	last := first + pageSize - 1
	if last > total {
		last = total
	}

	return fmt.Sprintf("WORKS %d–%d OF %d", first, last, total)
}

func artworkURL(row artworkRow) string {
	return wgaurl.GenerateFullArtworkUrl(wgaurl.ArtworkUrlDTO{
		ArtistName:   row.ArtistName,
		ArtistId:     row.ArtistID,
		ArtworkTitle: row.Title,
		ArtworkId:    row.ID,
	})
}

func artworkImageURL(row artworkRow) string {
	if row.Image == "" {
		return ""
	}

	return wgaurl.GenerateDeliveryURL(
		constants.CollectionArtworks,
		row.ID,
		row.Image,
		row.ImageWidth,
		wgaurl.DeliveryProfileRelatedTimelineCard,
		"",
	)
}

func markLabel(row artworkRow) string {
	date := formatDateLabel(row.DateStart, row.DateEnd, row.IsCirca != 0, row.Qualifier)
	if date == "" {
		return row.Title
	}

	return row.Title + " — " + date
}
