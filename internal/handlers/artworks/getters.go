package artworks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
)

const artworkSearchOptionsTTL = 6 * time.Hour

// artworkPeriodOptionsLimit bounds the period facet read. Production art_periods
// is currently empty and the embedded synthetic seed holds 32 periods, so 200 is
// comfortably conservative.
const artworkPeriodOptionsLimit = 200

// artworkVenueOptionsLimit bounds the counted collection facet returned to the
// filter rail. Name search is applied before the limit so all eligible holdings
// remain reachable without returning the full producer taxonomy.
const artworkVenueOptionsLimit = 40

// artworkLocationOptionsLimit keeps the legacy taxonomy helper bounded at the
// same public collection-facet limit. Production search no longer uses it.
const artworkLocationOptionsLimit = artworkVenueOptionsLimit

const (
	artTypesCacheKey    = "artworks:search:art-types"
	artFormsCacheKey    = "artworks:search:art-forms"
	artSchoolsCacheKey  = "artworks:search:art-schools"
	artistNamesCacheKey = "artworks:search:artist-names"
	artPeriodsCacheKey  = "artworks:search:art-periods"
	locationsCacheKey   = "artworks:search:locations"
)

// getArtTypesOptions returns a map of art type slugs and their corresponding names.
// It retrieves the art types from the database using the provided PocketBase app instance.
func getArtTypesOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artTypesCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	options := map[string]string{
		"": "Any",
	}
	c, err := app.FindRecordsByFilter(constants.CollectionArtTypes, "", "+name", 0, 0)

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.GetString("slug")] = v.GetString("name")
	}

	utils.SetCachedValue(app, artTypesCacheKey, cloneStringMap(options), artworkSearchOptionsTTL)

	return options, nil
}

// getArtFormOptions returns a map of art form slugs to their corresponding names.
// It retrieves the art forms from the database using the provided PocketBase app instance.
func getArtFormOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artFormsCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	options := map[string]string{
		"": "Any",
	}
	c, err := app.FindRecordsByFilter(constants.CollectionArtForms, "", "+name", 0, 0)

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.GetString("slug")] = v.GetString("name")
	}

	utils.SetCachedValue(app, artFormsCacheKey, cloneStringMap(options), artworkSearchOptionsTTL)

	return options, nil
}

// getArtSchoolOptions returns a map of art school options where the key is the slug and the value is the name.
func getArtSchoolOptions(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artSchoolsCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	options := map[string]string{
		"": "Any",
	}
	c, err := app.FindRecordsByFilter(constants.CollectionSchools, "", "+name", 0, 0)

	if err != nil {
		return options, err
	}

	for _, v := range c {
		options[v.GetString("slug")] = v.GetString("name")
	}

	utils.SetCachedValue(app, artSchoolsCacheKey, cloneStringMap(options), artworkSearchOptionsTTL)

	return options, nil
}

func GetArtistNameList(app *pocketbase.PocketBase) (map[string]string, error) {
	if cached, ok := utils.GetCachedValue[map[string]string](app, artistNamesCacheKey); ok {
		return cloneStringMap(cached), nil
	}

	names := make(map[string]string) // Initialize the names map
	c, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		"published = true",
		"+filing_name",
		0,
		0,
	)

	if err != nil {
		return names, err
	}

	for _, v := range c {
		names[url.GenerateArtistUrl(url.ArtistUrlDTO{
			ArtistId:   v.GetString("id"),
			ArtistName: v.GetString("name"),
		})] = v.GetString("filing_name")
	}

	utils.SetCachedValue(app, artistNamesCacheKey, cloneStringMap(names), artworkSearchOptionsTTL)

	return names, nil
}

func cloneStringMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))

	for key, value := range source {
		clone[key] = value
	}

	return clone
}

// buildChipGroup assembles a labelled filter choice group with an "ALL" default
// and the options sorted by label.
func buildChipGroup(legend string, name string, options map[string]string, selected string) dto.ChipGroup {
	group := dto.ChipGroup{
		Legend:  legend,
		Name:    name,
		Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: selected == ""}},
	}

	for _, option := range sortedChipOptions(options) {
		option.Checked = selected == option.Value
		group.Options = append(group.Options, option)
	}

	return group
}

// buildUnavailableChipGroup renders an honest empty filter group for a field
// WGA does not yet store. It carries a note rather than fabricated options.
func buildUnavailableChipGroup(legend string, name string) dto.ChipGroup {
	return dto.ChipGroup{
		Legend: legend,
		Name:   name,
		Note:   "No values are recorded for this filter yet.",
	}
}

func sortedChipOptions(options map[string]string) []dto.ChipOption {
	sorted := make([]dto.ChipOption, 0, len(options))
	for value, label := range options {
		if value == "" {
			continue
		}
		sorted = append(sorted, dto.ChipOption{Label: label, Value: value})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Label < sorted[j].Label
	})

	return sorted
}

// optionEntry is one ordered filter choice. Period options are ordered by the
// producer chronology rather than alphabetically, so they cannot use the
// label-sorted map helper.
type optionEntry struct {
	value string
	label string
}

// facetOptions is a bounded, ordered facet option set plus a truncation flag.
// The flag is true only when a limit+1 read surfaced an extra row, proving the
// facet has more values than the limit.
type facetOptions struct {
	entries   []optionEntry
	truncated bool
}

// truncateFacet caps the entries at limit and sets truncated only when an extra
// entry beyond the limit was actually present.
func truncateFacet(entries []optionEntry, limit int) facetOptions {
	if len(entries) > limit {
		return facetOptions{entries: entries[:limit], truncated: true}
	}

	return facetOptions{entries: entries, truncated: false}
}

// getArtPeriodOptions returns the approved art periods in chronological order
// (start year, then name, then id for a deterministic tie-break) so the facet
// reads as history rather than a word list.
func getArtPeriodOptions(app *pocketbase.PocketBase) (facetOptions, error) {
	if cached, ok := utils.GetCachedValue[facetOptions](app, artPeriodsCacheKey); ok {
		return cloneFacetOptions(cached), nil
	}

	records, err := app.FindRecordsByFilter("art_periods", "", "+start,+name,+id", artworkPeriodOptionsLimit+1, 0)
	if err != nil {
		return facetOptions{}, err
	}

	entries := make([]optionEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, optionEntry{value: record.Id, label: record.GetString("name")})
	}

	result := truncateFacet(entries, artworkPeriodOptionsLimit)
	utils.SetCachedValue(app, artPeriodsCacheKey, cloneFacetOptions(result), artworkSearchOptionsTTL)

	return result, nil
}

// getLocationOptions returns the producer locations taxonomy ordered by name,
// then id for a deterministic tie-break between duplicate names.
func getLocationOptions(app *pocketbase.PocketBase) (facetOptions, error) {
	if cached, ok := utils.GetCachedValue[facetOptions](app, locationsCacheKey); ok {
		return cloneFacetOptions(cached), nil
	}

	records, err := app.FindRecordsByFilter(constants.CollectionLocations, "", "+name,+id", artworkLocationOptionsLimit+1, 0)
	if err != nil {
		return facetOptions{}, err
	}

	entries := make([]optionEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, optionEntry{value: record.Id, label: record.GetString("name")})
	}

	result := truncateFacet(entries, artworkLocationOptionsLimit)
	utils.SetCachedValue(app, locationsCacheKey, cloneFacetOptions(result), artworkSearchOptionsTTL)

	return result, nil
}

func cloneOptionEntries(source []optionEntry) []optionEntry {
	return append([]optionEntry(nil), source...)
}

func cloneFacetOptions(source facetOptions) facetOptions {
	return facetOptions{
		entries:   cloneOptionEntries(source.entries),
		truncated: source.truncated,
	}
}

// buildChipGroupFromEntries assembles a labelled filter choice group with an
// "ALL" default and preserves the caller-provided option order.
func buildChipGroupFromEntries(legend string, name string, entries []optionEntry, selected string) dto.ChipGroup {
	group := dto.ChipGroup{
		Legend:  legend,
		Name:    name,
		Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: selected == ""}},
	}

	for _, entry := range entries {
		group.Options = append(group.Options, dto.ChipOption{
			Label:   entry.label,
			Value:   entry.value,
			Checked: selected == entry.value,
		})
	}

	return group
}

// buildFilterGroup renders a populated filter group, an honest empty group when
// upstream data has no values, or a truncation note when the bounded read may
// have left further options unlisted.
func buildFilterGroup(legend string, name string, opts facetOptions, selected string) dto.ChipGroup {
	if len(opts.entries) == 0 {
		return buildUnavailableChipGroup(legend, name)
	}

	group := buildChipGroupFromEntries(legend, name, opts.entries, selected)

	if opts.truncated {
		group.Note = fmt.Sprintf("Showing the first %d options. Refine your search to narrow the list.", len(opts.entries))
	}

	return group
}

// venueOption is one collection holding with its eligible published-work count.
type venueOption struct {
	value string
	label string
	count int
}

// venueFacetOptions is the bounded result derived from the complete collection
// projection. retained carries an existing selected location that fell outside
// the forty-option cap or was excluded by venue_q; unknownSelected is set when
// the selected venue has no matching location record at all.
type venueFacetOptions struct {
	entries         []venueOption
	totalOptions    int
	totalHoldings   int
	omittedOptions  int
	omittedHoldings int
	retained        venueOption
	retainedSet     bool
	unknownSelected bool
}

// getVenueOptions filters, orders, and bounds the complete counted holdings
// projection in memory. VenueQuery changes only the collection choices; it never
// enters the artwork result predicate or a holding's own count.
func getVenueOptions(app *pocketbase.PocketBase, venueQuery string, selectedVenue string) (venueFacetOptions, error) {
	holdings, err := repositories.LoadCollectionHoldings(app)
	if err != nil {
		return venueFacetOptions{}, err
	}

	options := venueFacetOptions{entries: make([]venueOption, 0, min(len(holdings), artworkVenueOptionsLimit))}
	for _, holding := range holdings {
		option := venueOption{value: holding.Value, label: holding.Label, count: holding.Count}
		if selectedVenue != "" && option.value == selectedVenue {
			options.retained = option
			options.retainedSet = true
		}
		if option.count <= 0 || !venueNameMatchesQuery(option.label, venueQuery) {
			continue
		}
		options.entries = append(options.entries, option)
		options.totalOptions++
		options.totalHoldings += option.count
	}

	sort.Slice(options.entries, func(i, j int) bool {
		left, right := options.entries[i], options.entries[j]
		if left.count != right.count {
			return left.count > right.count
		}
		leftFolded, rightFolded := sqliteNoCaseKey(left.label), sqliteNoCaseKey(right.label)
		if leftFolded != rightFolded {
			return leftFolded < rightFolded
		}
		if left.label != right.label {
			return left.label < right.label
		}
		return left.value < right.value
	})
	if len(options.entries) > artworkVenueOptionsLimit {
		options.entries = options.entries[:artworkVenueOptionsLimit]
	}

	selectedShown := false
	for _, entry := range options.entries {
		if entry.value == selectedVenue {
			selectedShown = true
			break
		}
	}
	if selectedVenue != "" && selectedShown {
		options.retainedSet = false
	} else if selectedVenue != "" && !options.retainedSet {
		options.unknownSelected = true
	}

	finalizeVenueOptions(venueQuery, selectedVenue, &options)
	return options, nil
}

func sqliteNoCaseKey(value string) string {
	folded := []byte(value)
	for i, char := range folded {
		if char >= 'A' && char <= 'Z' {
			folded[i] = char + ('a' - 'A')
		}
	}
	return string(folded)
}

// unknownVenueLabel is the honest display label for a selected venue value that
// has no matching location record. It reads as unavailable rather than an
// opaque identifier while the underlying value still round-trips.
const unknownVenueLabel = "Unknown collection"

// finalizeVenueOptions enforces the forty-option cap and recomputes the
// omission counts over the candidate universe actually displayed. A selected
// venue that fell outside the capped list or the name query is appended,
// displacing the final visible unselected option when the list is already full,
// so the rendered choice set never exceeds the limit. Unknown selections hold
// no eligible works and are not part of the candidate universe, so they never
// affect the hidden-holdings accounting.
func finalizeVenueOptions(venueQuery string, selectedVenue string, options *venueFacetOptions) {
	hasRetained := options.retainedSet || options.unknownSelected
	if hasRetained {
		if len(options.entries) >= artworkVenueOptionsLimit {
			options.entries = options.entries[:len(options.entries)-1]
		}

		if options.retainedSet {
			options.entries = append(options.entries, options.retained)
		} else {
			options.entries = append(options.entries, venueOption{
				value: selectedVenue,
				label: unknownVenueLabel,
				count: 0,
			})
		}
	}

	shownOptions := len(options.entries)
	shownHoldings := 0
	for _, entry := range options.entries {
		shownHoldings += entry.count
	}

	if hasRetained {
		lastIsCandidate := options.retainedSet &&
			options.retained.count > 0 &&
			venueNameMatchesQuery(options.retained.label, venueQuery)
		if !lastIsCandidate {
			shownOptions--
			shownHoldings -= options.entries[len(options.entries)-1].count
		}
	}

	options.omittedOptions = options.totalOptions - shownOptions
	options.omittedHoldings = options.totalHoldings - shownHoldings
	if options.omittedOptions < 0 {
		options.omittedOptions = 0
	}
	if options.omittedHoldings < 0 {
		options.omittedHoldings = 0
	}
}

// venueNameMatchesQuery reports whether a collection name satisfies the
// collection-name search, mirroring the SQL instr predicate in getVenueOptions.
func venueNameMatchesQuery(label string, query string) bool {
	if query == "" {
		return true
	}

	return strings.Contains(sqliteNoCaseKey(label), sqliteNoCaseKey(query))
}

func venueFacetNote(options venueFacetOptions) string {
	if options.omittedOptions == 0 {
		return ""
	}

	return fmt.Sprintf(
		"Showing %d of %d collections; omitted collections hold %d works. Keep typing to narrow.",
		options.totalOptions-options.omittedOptions,
		options.totalOptions,
		options.omittedHoldings,
	)
}

func buildVenueChipGroup(options venueFacetOptions, selected string) dto.ChipGroup {
	if len(options.entries) == 0 {
		return dto.ChipGroup{
			Legend: "COLLECTION",
			Name:   "venue",
			Note:   "No eligible LOCATION holdings are recorded for this filter yet.",
		}
	}

	group := dto.ChipGroup{
		Legend:  "COLLECTION",
		Name:    "venue",
		Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: selected == ""}},
		Note:    venueFacetNote(options),
	}

	for _, option := range options.entries {
		group.Options = append(group.Options, dto.ChipOption{
			Label:   option.label,
			Value:   option.value,
			Checked: selected == option.value,
		})
	}

	return group
}
