package artworks

import "github.com/blackfyre/wga/internal/assets/templ/pages"

// Artwork sort state. Each criterion carries the explicit direction labels a
// scholar sees (rather than an ambiguous "ascending") and the deterministic
// database order for each direction. "id" is always the final tiebreak so
// pagination is stable across pages.
const (
	sortDate   = "date"
	sortArtist = "artist"
	sortTitle  = "title"

	sortAsc  = "asc"
	sortDesc = "desc"
)

type artworkSortCriterion struct {
	key       string
	label     string
	ascLabel  string
	descLabel string
	field     string
}

var artworkSortCriteria = []artworkSortCriterion{
	{key: sortTitle, label: "TITLE", ascLabel: "A–Z", descLabel: "Z–A", field: "title"},
	{key: sortArtist, label: "ARTIST", ascLabel: "A–Z", descLabel: "Z–A", field: "author.filing_name"},
	{key: sortDate, label: "DATE", ascLabel: "EARLIEST", descLabel: "LATEST", field: "date_start"},
}

func artworkSortCriterionFor(key string) (artworkSortCriterion, bool) {
	for _, criterion := range artworkSortCriteria {
		if criterion.key == key {
			return criterion, true
		}
	}

	return artworkSortCriterion{}, false
}

// sortString resolves the current sort state into a deterministic PocketBase
// sort string. Unknown criteria fall back to title ascending. "id" is always
// the final tiebreak so pagination is stable across pages.
func (f *filters) sortString() string {
	criterion, ok := artworkSortCriterionFor(f.Sort)
	if !ok {
		criterion, _ = artworkSortCriterionFor(sortTitle)
	}

	direction := "+"
	if f.SortDir == sortDesc {
		direction = "-"
	}

	return direction + criterion.field + ",+id"
}

// sortPrefixOrderBy returns the raw order-by prefix that keeps records with an
// unknown or missing value (0) after all authoritative values, regardless of
// direction. The date criterion uses date_start. It is applied before the
// resolver sort string because the PocketBase sort string cannot express a
// computed "unknown last" expression.
func (f *filters) sortPrefixOrderBy() string {
	switch f.Sort {
	case sortDate:
		return "(date_start = 0) ASC"
	default:
		return ""
	}
}

// sortDirLabel names the order the scholar will receive, not the abstract
// direction: "LATEST" reads where "DESCENDING" has to be decoded.
func (f *filters) sortDirLabel() string {
	criterion, ok := artworkSortCriterionFor(f.Sort)
	if !ok {
		criterion, _ = artworkSortCriterionFor(sortTitle)
	}

	if f.SortDir == sortDesc {
		return criterion.descLabel
	}

	return criterion.ascLabel
}

// forSort selects an inactive criterion in ascending order or reverses the active one.
func (f *filters) forSort(key string) *filters {
	next := f.clone()
	if key == f.Sort {
		next.SortDir = flipSortDir(f.SortDir)
	} else {
		next.Sort = key
		next.SortDir = sortAsc
	}
	next.Page = ""

	return next
}

// forView selects a result view without changing the sort or filters.
func (f *filters) forView(view string) *filters {
	next := f.clone()
	next.View = view
	next.Page = ""

	return next
}

func (f *filters) clone() *filters {
	next := *f

	return &next
}

// flipSortDir returns the opposite direction of the supplied one.
func flipSortDir(dir string) string {
	if dir == sortDesc {
		return sortAsc
	}

	return sortDesc
}

// buildSortOptions renders one criterion link per sort key, each carrying the
// rest of the active query state. Clicking the active criterion reverses it;
// clicking an inactive criterion selects its ascending direction.
func buildSortOptions(f *filters, dualModeContext *pages.ArtworkSearchDualMode) []pages.ArtworkSortOption {
	options := make([]pages.ArtworkSortOption, 0, len(artworkSortCriteria))
	for _, criterion := range artworkSortCriteria {
		options = append(options, pages.ArtworkSortOption{
			Key:   criterion.key,
			Label: criterion.label,
			Direction: func() string {
				if f.Sort == criterion.key {
					if f.SortDir == sortDesc {
						return criterion.descLabel
					}
					return criterion.ascLabel
				}
				return ""
			}(),
			Href:   buildArtworkSearchPath("/artworks", f.forSort(criterion.key), dualModeContext),
			Active: f.Sort == criterion.key,
		})
	}

	return options
}
