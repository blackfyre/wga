package utils

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// TitleSuffix is appended by GetTitle to every non-empty page title.
const TitleSuffix = " - WGA"

// TitleMaxLength is the maximum document-title length, in code points,
// including TitleSuffix, that FitTitle produces.
const TitleMaxLength = 80

const titleSeparator = " · "

// TitlePartRole decides how FitTitle may reduce a title part.
type TitlePartRole uint8

const (
	// TitleLead is the distinguishing free-text part. It is only ever
	// truncated, never removed.
	TitleLead TitlePartRole = iota
	// TitlePageName is the fixed page name. It is never reduced.
	TitlePageName
	// TitleFilter is a filter value. Removed filters are replaced by one
	// "+N" count part.
	TitleFilter
	// TitlePosition is the page position. It is removed without a count.
	TitlePosition
)

// TitlePart is one segment of a state-derived page title. Key identifies the
// part for reduction steps. Short may be empty when the part has no shorter
// form. Quoted parts render inside typographic quotes, which truncation
// preserves.
type TitlePart struct {
	Key    string
	Long   string
	Short  string
	Role   TitlePartRole
	Quoted bool
}

// TitleStepKind is the kind of one title reduction.
type TitleStepKind uint8

const (
	// TitleShorten replaces the part with its short form.
	TitleShorten TitleStepKind = iota
	// TitleRemove removes a filter or position part.
	TitleRemove
	// TitleTruncate truncates the part to Limit code points plus "…".
	TitleTruncate
)

// TitleStep is one reduction applied to the part named by Key. Steps naming
// an absent part are skipped.
type TitleStep struct {
	Kind  TitleStepKind
	Key   string
	Limit int
}

type titleSegment struct {
	part    TitlePart
	text    string
	removed bool
}

// FitTitle joins parts into a title without TitleSuffix and, when the
// suffixed title would exceed TitleMaxLength code points, applies steps in
// order until it fits. If the steps are not enough, it removes the remaining
// position and filter parts from the last and finally truncates the leads,
// from the last, to fit. Page names are never reduced, so only page names
// longer than the budget can exceed it.
func FitTitle(parts []TitlePart, steps []TitleStep) string {
	segments := make([]titleSegment, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part.Long) == "" {
			continue
		}
		segments = append(segments, titleSegment{part: part, text: part.Long})
	}

	title := renderTitle(segments)
	for _, step := range steps {
		if fitsTitle(title) {
			return title
		}
		index := titleSegmentIndex(segments, step.Key)
		if index < 0 {
			continue
		}
		applyTitleStep(&segments[index], step)
		title = renderTitle(segments)
	}

	for index := len(segments) - 1; index >= 0 && !fitsTitle(title); index-- {
		role := segments[index].part.Role
		if role == TitleFilter || role == TitlePosition {
			segments[index].removed = true
			title = renderTitle(segments)
		}
	}

	if fitsTitle(title) {
		return title
	}
	// Truncate leads from the last, so the first (most distinguishing) lead
	// keeps as much text as possible.
	for index := len(segments) - 1; index >= 0 && !fitsTitle(title); index-- {
		if segments[index].part.Role != TitleLead {
			continue
		}
		excess := utf8.RuneCountInString(title+TitleSuffix) - TitleMaxLength
		limit := max(utf8.RuneCountInString(segments[index].text)-excess-1, 1)
		segments[index].text = truncateTitleText(segments[index].text, limit)
		title = renderTitle(segments)
	}

	return title
}

func applyTitleStep(segment *titleSegment, step TitleStep) {
	switch step.Kind {
	case TitleShorten:
		if strings.TrimSpace(segment.part.Short) != "" {
			segment.text = segment.part.Short
		}
	case TitleRemove:
		if segment.part.Role == TitleFilter || segment.part.Role == TitlePosition {
			segment.removed = true
		}
	case TitleTruncate:
		if segment.part.Role != TitlePageName && step.Limit > 0 {
			segment.text = truncateTitleText(segment.text, step.Limit)
		}
	}
}

func truncateTitleText(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	runes := []rune(text)
	return strings.TrimRight(string(runes[:limit]), " ") + "…"
}

func titleSegmentIndex(segments []titleSegment, key string) int {
	for index, segment := range segments {
		if segment.part.Key == key && !segment.removed {
			return index
		}
	}
	return -1
}

func fitsTitle(title string) bool {
	return utf8.RuneCountInString(title+TitleSuffix) <= TitleMaxLength
}

// renderTitle joins the visible segments. Removed filters collapse into one
// "+N" part placed after the last visible filter, before the page position.
func renderTitle(segments []titleSegment) string {
	removedFilters := 0
	for _, segment := range segments {
		if segment.removed && segment.part.Role == TitleFilter {
			removedFilters++
		}
	}

	rendered := make([]string, 0, len(segments)+1)
	countPlaced := removedFilters == 0
	for _, segment := range segments {
		if !countPlaced && segment.part.Role == TitlePosition {
			rendered = append(rendered, "+"+strconv.Itoa(removedFilters))
			countPlaced = true
		}
		if segment.removed {
			continue
		}
		text := segment.text
		if segment.part.Quoted {
			text = "“" + text + "”"
		}
		rendered = append(rendered, text)
	}
	if !countPlaced {
		rendered = append(rendered, "+"+strconv.Itoa(removedFilters))
	}

	return strings.Join(rendered, titleSeparator)
}

// TitlePagePosition returns the "p. N/M" page-position text, or an empty
// string on the first page, where the position is omitted.
func TitlePagePosition(page int, pageCount int) string {
	if page <= 1 || pageCount <= 1 {
		return ""
	}

	return "p. " + strconv.Itoa(page) + "/" + strconv.Itoa(pageCount)
}
