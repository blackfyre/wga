package artists

import (
	"regexp"
	"strings"
)

// largeHoldingSelectionThreshold is the number of published selections from
// which an artist record replaces its selection previews with the form-grouped
// selection index. Records with two up to this value minus one keep previews.
const largeHoldingSelectionThreshold = 10

// selectionTitle is one published selection, in the artist's published
// selection order, as input to groupSelectionsByForm.
type selectionTitle struct {
	Title     string
	WorkCount int
}

// selectionFormEntry is one selection inside a form. Index is the selection's
// position in the input, so callers can map it back to its record.
type selectionFormEntry struct {
	Index     int
	Label     string
	WorkCount int
}

// selectionForm is one form of a large holding: its display name (the first
// occurrence's spelling), its ordered selections, and its work total.
type selectionForm struct {
	Name       string
	Selections []selectionFormEntry
	WorkTotal  int
}

var (
	// trailingSelectionCount matches a work count appended to a title after a
	// middle dot, a spaced dash, or a comma. A count is either up to three
	// digits or digits grouped with thousands separators, so an ungrouped year
	// such as "Saint Ursula Shrine, 1489" and an unspaced range such as
	// "1435-40" are not mistaken for counts.
	trailingSelectionCount = regexp.MustCompile(`(?:\s*·|\s+[-–—]|\s*,)\s*\d{1,3}(?:[,.\x{00A0}\x{202F} ]\d{3})*\s*(?:works?)?\s*$`)
	// hyphenatedYearRange matches a year range written with a hyphen.
	hyphenatedYearRange = regexp.MustCompile(`\b(\d{3,4})\s*-\s*(\d{1,4})\b`)
	whitespaceRun       = regexp.MustCompile(`\s+`)
)

// parseSelectionTitle derives a selection's form and qualifier from its
// display title. The order is fixed: strip a trailing count, then try the
// final bracketed group, then the first colon, and otherwise use the full
// title as the form with no qualifier.
func parseSelectionTitle(title string) (form string, qualifier string) {
	title = strings.TrimSpace(whitespaceRun.ReplaceAllString(title, " "))
	if stripped := strings.TrimSpace(trailingSelectionCount.ReplaceAllString(title, "")); stripped != "" {
		title = stripped
	}

	if form, qualifier, ok := splitFinalBracket(title); ok {
		return form, enDashYearRanges(qualifier)
	}

	if before, after, found := strings.Cut(title, ":"); found {
		before, after = strings.TrimSpace(before), strings.TrimSpace(after)
		if before != "" && after != "" {
			return before, enDashYearRanges(after)
		}
	}

	return title, ""
}

// splitFinalBracket splits "Form (qualifier)" at the balanced bracket group
// that closes the title. It reports false when the title does not end in a
// bracket, the brackets do not balance, or either side would be empty.
func splitFinalBracket(title string) (string, string, bool) {
	if !strings.HasSuffix(title, ")") {
		return "", "", false
	}

	depth := 0
	for i := len(title) - 1; i >= 0; i-- {
		switch title[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth == 0 {
				form := strings.TrimSpace(title[:i])
				qualifier := strings.TrimSpace(title[i+1 : len(title)-1])
				if form == "" || qualifier == "" {
					return "", "", false
				}
				return form, qualifier, true
			}
		}
	}

	return "", "", false
}

func enDashYearRanges(qualifier string) string {
	return hyphenatedYearRange.ReplaceAllString(qualifier, "$1–$2")
}

// groupSelectionsByForm groups ordered selections by form. Forms are compared
// case-insensitively and appear in the order each first occurs; selections
// keep their input order within each form. A selection's label is its
// qualifier, or its full title when the title has none.
func groupSelectionsByForm(selections []selectionTitle) []selectionForm {
	forms := []selectionForm{}
	positions := map[string]int{}

	for index, selection := range selections {
		form, qualifier := parseSelectionTitle(selection.Title)
		label := qualifier
		if label == "" {
			label = form
		}

		key := strings.ToLower(form)
		position, seen := positions[key]
		if !seen {
			position = len(forms)
			positions[key] = position
			forms = append(forms, selectionForm{Name: form})
		}

		forms[position].Selections = append(forms[position].Selections, selectionFormEntry{
			Index:     index,
			Label:     label,
			WorkCount: selection.WorkCount,
		})
		forms[position].WorkTotal += selection.WorkCount
	}

	return forms
}
