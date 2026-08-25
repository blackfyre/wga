package artists

import (
	"encoding/json"
	"fmt"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils/jsonld"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	artistsPageSize = 30

	viewGrid = "grid"
	viewList = "list"

	sortAZ    = "az"
	sortZA    = "za"
	sortBirth = "birth"
)

// indexQuery is the parsed, allow-listed artist-index query state.
type indexQuery struct {
	query    string
	letter   string
	school   string
	period   string
	bornFrom int
	bornTo   int
	bornMin  int
	bornMax  int
	view     string
	sort     string
	page     int
}

// parseIndexQuery trims, allow-lists, clamps, and re-orders the raw query
// parameters into a safe, deterministic index state.
func parseIndexQuery(values neturl.Values, bornMin int, bornMax int) indexQuery {
	query := indexQuery{
		query:   strings.TrimSpace(values.Get("q")),
		school:  strings.TrimSpace(values.Get("school")),
		period:  strings.TrimSpace(values.Get("period")),
		view:    viewGrid,
		sort:    sortAZ,
		page:    1,
		bornMin: bornMin,
		bornMax: bornMax,
	}

	letter := strings.ToUpper(strings.TrimSpace(values.Get("letter")))
	if len(letter) == 1 && letter[0] >= 'A' && letter[0] <= 'Z' {
		query.letter = letter
	}

	if view := strings.TrimSpace(values.Get("view")); view == viewList {
		query.view = viewList
	}

	switch sort := strings.TrimSpace(values.Get("sort")); sort {
	case sortAZ, sortZA, sortBirth:
		query.sort = sort
	}

	query.bornFrom, _ = clampBornYear(values.Get("born_from"), bornMin, bornMax)
	query.bornTo, _ = clampBornYear(values.Get("born_to"), bornMin, bornMax)
	if query.bornFrom > 0 && query.bornTo > 0 && query.bornFrom > query.bornTo {
		query.bornFrom, query.bornTo = query.bornTo, query.bornFrom
	}

	query.page = parsePage(values.Get("page"))

	return query
}

// clampBornYear parses a birth-year bound and clamps it into the published
// known-year range. It returns false for empty, non-numeric, or unclampable
// input, which the caller treats as "unset" so an invalid value never hides
// results. When the published set has no known birth years (min/max are zero)
// there is no range to interpret the bound against, so the bound is discarded.
func clampBornYear(raw string, min int, max int) (int, bool) {
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	if min <= 0 || max <= 0 {
		return 0, false
	}
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}

	return value, true
}

// parsePage returns a positive page number, defaulting to 1 for invalid input.
func parsePage(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return 1
	}

	return value
}

// path returns the canonical /artists URL for the state, with parameters in a
// fixed order and defaults omitted.
func (q indexQuery) path() string {
	parts := []string{}
	add := func(key string, value string) {
		parts = append(parts, neturl.QueryEscape(key)+"="+neturl.QueryEscape(value))
	}

	if q.query != "" {
		add("q", q.query)
	}
	if q.letter != "" {
		add("letter", q.letter)
	}
	if q.school != "" {
		add("school", q.school)
	}
	if q.period != "" {
		add("period", q.period)
	}
	if q.bornFrom > 0 {
		add("born_from", strconv.Itoa(q.bornFrom))
	}
	if q.bornTo > 0 {
		add("born_to", strconv.Itoa(q.bornTo))
	}
	if q.view != "" && q.view != viewGrid {
		add("view", q.view)
	}
	if q.sort != "" && q.sort != sortAZ {
		add("sort", q.sort)
	}
	if q.page > 1 {
		add("page", strconv.Itoa(q.page))
	}

	if len(parts) == 0 {
		return "/artists"
	}

	return "/artists?" + strings.Join(parts, "&")
}

func (q indexQuery) withLetter(letter string) indexQuery {
	next := q
	next.letter = letter
	next.page = 1

	return next
}

func (q indexQuery) withView(view string) indexQuery {
	next := q
	next.view = view
	next.page = 1

	return next
}

func (q indexQuery) withSort(sort string) indexQuery {
	next := q
	next.sort = sort
	next.page = 1

	return next
}

func (q indexQuery) withPage(page int) indexQuery {
	next := q
	next.page = page

	return next
}

// repositoryFilter maps the parsed state onto the bounded repository query.
func (q indexQuery) repositoryFilter(periodStart int, periodEnd int) repositories.ArtistIndexFilter {
	filter := repositories.ArtistIndexFilter{
		Query:        q.query,
		Letter:       q.letter,
		School:       q.school,
		BornActive:   q.bornFrom > 0 || q.bornTo > 0,
		BornFrom:     q.bornFrom,
		BornTo:       q.bornTo,
		Limit:        artistsPageSize,
		Offset:       (q.page - 1) * artistsPageSize,
		PeriodActive: periodStart > 0 || periodEnd > 0,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
	}

	switch q.sort {
	case sortZA:
		filter.Sort = repositories.ArtistSortNameDesc
	case sortBirth:
		filter.Sort = repositories.ArtistSortBirth
	default:
		filter.Sort = repositories.ArtistSortNameAsc
	}

	return filter
}

// schoolRef is one loaded school and its slug for filter resolution.
type schoolRef struct {
	id   string
	slug string
	name string
}

// artPeriod is one loaded art period and its birth-year range.
type artPeriod struct {
	id    string
	name  string
	start int
	end   int
}

func loadSchools(app *pocketbase.PocketBase) ([]schoolRef, error) {
	records, err := app.FindRecordsByFilter("schools", "", "+name", 0, 0)
	if err != nil {
		return nil, err
	}

	schools := make([]schoolRef, 0, len(records))
	for _, record := range records {
		schools = append(schools, schoolRef{
			id:   record.Id,
			slug: record.GetString("slug"),
			name: record.GetString("name"),
		})
	}

	return schools, nil
}

func loadArtPeriods(app *pocketbase.PocketBase) ([]artPeriod, error) {
	records, err := app.FindRecordsByFilter("art_periods", "", "+start,+name", 0, 0)
	if err != nil {
		return nil, err
	}

	periods := make([]artPeriod, 0, len(records))
	for _, record := range records {
		periods = append(periods, artPeriod{
			id:    record.Id,
			name:  record.GetString("name"),
			start: record.GetInt("start"),
			end:   record.GetInt("end"),
		})
	}

	return periods, nil
}

// periodForBirth returns the art-period label whose range unambiguously contains
// the birth year. It returns "" when the year is unknown, unmatched, or matches
// more than one period, so callers can use an honest unavailable marker instead
// of inventing a label.
func periodForBirth(periods []artPeriod, birth int) string {
	if birth <= 0 {
		return ""
	}

	match := ""
	found := false
	ambiguous := false
	for _, period := range periods {
		if period.start <= 0 || period.end <= 0 || period.start > period.end {
			continue
		}
		if period.start <= birth && birth <= period.end {
			if found {
				ambiguous = true
			}
			match = period.name
			found = true
		}
	}

	if !found || ambiguous {
		return ""
	}

	return match
}

func formatArtistDates(birth int, death int) string {
	if birth > 0 && death > 0 {
		return strconv.Itoa(birth) + "–" + strconv.Itoa(death)
	}
	if birth > 0 {
		return "b. " + strconv.Itoa(birth)
	}
	if death > 0 {
		return "d. " + strconv.Itoa(death)
	}

	return ""
}

func resolveSchoolNames(ids []string, byID map[string]string) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := byID[id]; ok {
			names = append(names, name)
		}
	}

	return strings.Join(names, ", ")
}

func sortLabel(sort string) string {
	switch sort {
	case sortZA:
		return "Z–A"
	case sortBirth:
		return "BIRTH YEAR"
	default:
		return "A–Z"
	}
}

func nextSort(sort string) string {
	switch sort {
	case sortAZ:
		return sortZA
	case sortZA:
		return sortBirth
	default:
		return sortAZ
	}
}

func filterNote(query indexQuery, schoolName string, periodName string) string {
	parts := []string{}
	if query.letter != "" {
		parts = append(parts, "· "+query.letter)
	}
	if query.school != "" {
		parts = append(parts, "· "+strings.ToUpper(schoolName))
	}
	if query.period != "" {
		parts = append(parts, "· "+strings.ToUpper(periodName))
	}
	if query.bornFrom > 0 || query.bornTo > 0 {
		from := query.bornFrom
		if from == 0 {
			from = query.bornMin
		}
		to := query.bornTo
		if to == 0 {
			to = query.bornMax
		}
		parts = append(parts, "· BORN "+strconv.Itoa(from)+"–"+strconv.Itoa(to))
	}

	return strings.Join(parts, " ")
}

func buildArtistsJsonLd(records []*core.Record) string {
	collector := make([]jsonld.Person, 0, len(records))
	for _, record := range records {
		collector = append(collector, jsonld.ArtistJsonLd(record))
	}
	marshalled, err := json.Marshal(collector)
	if err != nil {
		return ""
	}

	return fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)
}

// buildArtistIndexView parses the request, loads the filtered page, and
// assembles the page-owned view plus the canonical URL for the response.
func buildArtistIndexView(app *pocketbase.PocketBase, values neturl.Values) (pages.ArtistsView, string, error) {
	schools, err := loadSchools(app)
	if err != nil {
		return pages.ArtistsView{}, "", err
	}
	periods, err := loadArtPeriods(app)
	if err != nil {
		return pages.ArtistsView{}, "", err
	}

	schoolByID := make(map[string]string, len(schools))
	schoolSlugToName := make(map[string]string, len(schools))
	for _, school := range schools {
		schoolByID[school.id] = school.name
		schoolSlugToName[school.slug] = school.name
	}

	periodByName := make(map[string]artPeriod, len(periods))
	for _, period := range periods {
		periodByName[period.id] = period
	}

	repo := repositories.NewArtistIndexRepository(app)
	bornMin, bornMax, err := repo.BirthYearBounds()
	if err != nil {
		return pages.ArtistsView{}, "", err
	}

	query := parseIndexQuery(values, bornMin, bornMax)

	if query.school != "" {
		if _, ok := schoolSlugToName[query.school]; !ok {
			query.school = ""
		}
	}

	periodStart := 0
	periodEnd := 0
	if query.period != "" {
		if period, ok := periodByName[query.period]; ok {
			periodStart = period.start
			periodEnd = period.end
		} else {
			query.period = ""
		}
	}

	filter := query.repositoryFilter(periodStart, periodEnd)
	total, err := repo.CountArtists(filter)
	if err != nil {
		return pages.ArtistsView{}, "", err
	}

	pageCount := (total + artistsPageSize - 1) / artistsPageSize
	if pageCount == 0 {
		query.page = 1
	} else if query.page > pageCount {
		query.page = pageCount
	}

	// Rebuild the offset after the page has been clamped to the result set.
	filter.Offset = (query.page - 1) * artistsPageSize

	indexed, err := repo.ListArtists(filter)
	if err != nil {
		return pages.ArtistsView{}, "", err
	}

	availableLetters, err := repo.ListAvailableLetters(repositories.ArtistIndexFilter{})
	if err != nil {
		return pages.ArtistsView{}, "", err
	}

	letters := buildIndexLetters(query.letter, availableLetters, query)

	artists := make([]pages.ArtistRow, 0, len(indexed))
	records := make([]*core.Record, 0, len(indexed))
	for _, entry := range indexed {
		artists = append(artists, pages.ArtistRow{
			URL:       urlutils.GenerateArtistUrlFromRecord(entry.Record),
			Name:      entry.Record.GetString("name"),
			Dates:     formatArtistDates(entry.Record.GetInt("year_of_birth"), entry.Record.GetInt("year_of_death")),
			School:    resolveSchoolNames(entry.Record.GetStringSlice("school"), schoolByID),
			Period:    periodForBirth(periods, entry.Record.GetInt("year_of_birth")),
			Form:      entry.Record.GetString("profession"),
			Thumb:     urlutils.GenerateArtistPortraitImageURL(entry.Record, urlutils.DeliveryProfileCardAndArtistIndex, ""),
			Available: entry.Available,
		})
		records = append(records, entry.Record)
	}

	view := pages.ArtistsView{
		Letters:        letters,
		Schools:        buildSchoolOptions(schools, query.school),
		Periods:        buildPeriodOptions(periods, query.period),
		NameField:      buildNameField(query.query),
		NameQuery:      query.query,
		SelectedLetter: query.letter,
		View:           query.view,
		Sort:           query.sort,
		SortLabel:      sortLabel(query.sort),
		GridUrl:        query.withView(viewGrid).path(),
		ListUrl:        query.withView(viewList).path(),
		SortUrl:        query.withSort(nextSort(query.sort)).path(),
		AllUrl:         query.withLetter("").path(),
		ResetUrl:       "/artists",
		PrevUrl:        query.withPage(query.page - 1).path(),
		NextUrl:        query.withPage(query.page + 1).path(),
		FilterNote:     filterNote(query, schoolSlugToName[query.school], periodName(periodByName, query.period)),
		Total:          total,
		Page:           query.page,
		PageCount:      pageCount,
		Artists:        artists,
		Jsonld:         buildArtistsJsonLd(records),
	}

	if bornMin > 0 && bornMax > 0 {
		view.HasBirthBounds = true
		view.BornRange = dto.RangeField{
			Label:     "BORN BETWEEN",
			FromID:    "born-from",
			FromName:  "born_from",
			FromValue: effectiveBorn(query.bornFrom, bornMin),
			ToID:      "born-to",
			ToName:    "born_to",
			ToValue:   effectiveBorn(query.bornTo, bornMax),
			Min:       bornMin,
			Max:       bornMax,
			Step:      1,
		}
	}

	return view, query.path(), nil
}

func buildIndexLetters(selected string, available []string, query indexQuery) []pages.ArtistLetter {
	availableSet := make(map[string]bool, len(available))
	for _, letter := range available {
		availableSet[letter] = true
	}

	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letters := make([]pages.ArtistLetter, 0, len(alphabet))
	for _, letter := range alphabet {
		value := string(letter)
		entry := pages.ArtistLetter{
			Label:    value,
			Enabled:  availableSet[value],
			Selected: selected == value,
		}
		if entry.Enabled {
			entry.Href = query.withLetter(value).path()
		}
		letters = append(letters, entry)
	}

	return letters
}

func buildSchoolOptions(schools []schoolRef, selected string) []dto.ChipOption {
	options := []dto.ChipOption{{Label: "ALL", Value: "", Checked: selected == ""}}
	for _, school := range schools {
		options = append(options, dto.ChipOption{
			Label:   school.name,
			Value:   school.slug,
			Checked: selected == school.slug,
		})
	}

	return options
}

func buildPeriodOptions(periods []artPeriod, selected string) []dto.ChipOption {
	options := []dto.ChipOption{{Label: "ALL", Value: "", Checked: selected == ""}}
	for _, period := range periods {
		options = append(options, dto.ChipOption{
			Label:   period.name,
			Value:   period.id,
			Checked: selected == period.id,
		})
	}

	return options
}

func buildNameField(value string) dto.Field {
	return dto.Field{
		ID:          "artist-name",
		Name:        "q",
		Label:       "NAME CONTAINS",
		Type:        "search",
		Value:       value,
		Placeholder: "e.g. van",
	}
}

func effectiveBorn(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}

func periodName(periods map[string]artPeriod, id string) string {
	if period, ok := periods[id]; ok {
		return period.name
	}

	return ""
}
