package dual

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/glossary"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	dualLookupPath              = "/dual-mode/lookup"
	dualLookupLimit             = 20
	dualLookupMinimumQueryRunes = 2

	dualIndexPageSize = 30

	sizeSmall  = "small"
	sizeMedium = "medium"
	sizeLarge  = "large"

	viewGrid = "grid"
	viewList = "list"

	sortAZ    = "az"
	sortZA    = "za"
	sortBirth = "birth"

	maxDualPathLength = 512
)

// dualArtistIdentityFilter is the PocketBase filter fragment requiring both
// authoritative identity fields to be present. Prior-bootstrap records carry
// blank filing/short fields and are denied rather than reconstructed.
const dualArtistIdentityFilter = "filing_name != '' && short_name != ''"

// panePathDto describes a parsed pane content path.
type panePathDto struct {
	Kind    string // "default" | "artist" | "artwork"
	Id      string
	RelPath string
}

// dualState is the parsed, canonical dual-mode route state.
type dualState struct {
	left  dualPaneState
	right dualPaneState
	wide  bool
}

// dualPaneState is the independent state of one window.
type dualPaneState struct {
	path     string         // canonical record path; "" = artist index
	renderTo string         // absolute pane name content links open in ("left" | "right")
	index    dualIndexState // remembered index/filter state for back navigation
	size     string         // study-image size for the work view
}

// dualIndexState is the artist-index filter/view/sort state of one window.
type dualIndexState struct {
	letter   string
	school   string
	period   string
	query    string
	bornFrom int
	bornTo   int
	view     string
	sort     string
}

// dualParam is one ordered query parameter of the canonical serialization.
type dualParam struct {
	name  string
	value string
}

// dualReference is the bounded reference data both windows resolve against.
type dualReference struct {
	schoolSlugs map[string]string // slug -> display name
	schoolByID  map[string]string // id -> display name
	periods     []dualPeriod      // sorted by start
	periodByID  map[string]dualPeriod
	bornMin     int
	bornMax     int
}

type dualSchool struct {
	slug string
	name string
}

type dualPeriod struct {
	id    string
	name  string
	start int
	end   int
}

func renderDualModePage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	ref, err := loadDualReference(app)
	if err != nil {
		app.Logger().Error("Error loading dual mode reference data", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	state := parseDualState(c.Request.URL.Query()).normalize(ref)

	// Canonicalise each pane path to the generated public record URL before any
	// link or the push URL is built, so a shared URL never carries a mismatched
	// slug or artist segment. Missing, unpublished, or invalid records fall back
	// to that pane's index.
	leftPath, err := resolvePaneCanonicalPath(app, state.left)
	if err != nil {
		app.Logger().Error("Error resolving left pane path", "error", err.Error())
		return utils.ServerFaultError(c)
	}
	rightPath, err := resolvePaneCanonicalPath(app, state.right)
	if err != nil {
		app.Logger().Error("Error resolving right pane path", "error", err.Error())
		return utils.ServerFaultError(c)
	}
	state.left.path = leftPath
	state.right.path = rightPath

	leftWindow, err := buildWindow(app, "left", state.left, state, ref)
	if err != nil {
		app.Logger().Error("Error rendering left window", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	rightWindow, err := buildWindow(app, "right", state.right, state, ref)
	if err != nil {
		app.Logger().Error("Error rendering right window", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	view := pages.DualModeView{
		Windows:   [2]pages.DualWindow{leftWindow, rightWindow},
		SwapHref:  state.swapped().path(),
		ResetHref: state.reset().path(),
		ExitHref:  "/artists",
		WideHref:  state.withWide(true).path(),
		ForceWide: state.wide,
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Dual Mode")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Compare artists and artworks side by side.")
	c.Response.Header().Set("HX-Push-Url", state.path())

	var buff bytes.Buffer
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err = pages.DualModeBlock(view).Render(ctx, &buff)
	} else {
		err = pages.DualModePage(view).Render(ctx, &buff)
	}
	if err != nil {
		app.Logger().Error("Error rendering dual mode page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

func baseDualWindow(side string, pane dualPaneState, state dualState) pages.DualWindow {
	window := pages.DualWindow{
		Key:       side,
		Label:     strings.ToUpper(side),
		SelfSel:   "#dual-" + side,
		IndexHref: state.withPanePath(side, "").path(),
	}

	if side == "left" {
		window.Tag = "L"
		window.OtherLabel = "RIGHT WINDOW"
	} else {
		window.Tag = "R"
		window.OtherLabel = "LEFT WINDOW"
	}

	window.RoutesToSelf = pane.renderTo == side
	window.RouteHref = state.withPaneRenderTo(side, reverseSide(pane.renderTo)).path()
	window.TargetSel = "#dual-" + pane.renderTo

	return window
}

func buildWindow(app *pocketbase.PocketBase, side string, pane dualPaneState, state dualState, ref dualReference) (pages.DualWindow, error) {
	window := baseDualWindow(side, pane, state)

	if pane.path == "" {
		return buildIndexWindow(app, side, pane, state, ref, window)
	}

	parsed, err := parsePanePath(pane.path)
	if err != nil {
		return buildIndexWindow(app, side, pane, state, ref, window)
	}

	switch parsed.Kind {
	case "artist":
		record, buildErr := buildDualArtistRecord(app, side, pane, state, ref)
		if buildErr != nil {
			if errors.Is(buildErr, sql.ErrNoRows) {
				return buildIndexWindow(app, side, pane, state, ref, window)
			}
			return window, buildErr
		}
		window.View = "artist"
		window.Artist = record
		window.Crumb = []pages.DualCrumb{
			{Label: "ARTISTS", Href: window.IndexHref},
			{Label: record.ShortName},
		}
		window.BackHref = window.IndexHref
		window.Send = dualSendLink(side, pane, state, window.OtherLabel)
		return window, nil

	case "artwork":
		record, artistName, artistPath, buildErr := buildDualWorkRecord(app, side, pane, state, ref)
		if buildErr != nil {
			if errors.Is(buildErr, sql.ErrNoRows) {
				return buildIndexWindow(app, side, pane, state, ref, window)
			}
			return window, buildErr
		}
		backHref := state.withPanePath(side, artistPath).path()
		window.View = "work"
		window.Work = record
		window.Crumb = []pages.DualCrumb{
			{Label: "ARTISTS", Href: window.IndexHref},
			{Label: artistName, Href: backHref},
			{Label: record.Title},
		}
		window.BackHref = backHref
		window.Send = dualSendLink(side, pane, state, window.OtherLabel)
		return window, nil

	default:
		return buildIndexWindow(app, side, pane, state, ref, window)
	}
}

func buildIndexWindow(app *pocketbase.PocketBase, side string, pane dualPaneState, state dualState, ref dualReference, window pages.DualWindow) (pages.DualWindow, error) {
	window.View = "index"
	window.Crumb = []pages.DualCrumb{{Label: "ARTISTS"}}
	window.BackHref = ""

	indexView, err := buildDualIndexView(app, side, pane, state, ref)
	if err != nil {
		return window, err
	}
	window.Index = indexView

	return window, nil
}

func dualSendLink(side string, pane dualPaneState, state dualState, otherLabel string) pages.DualLink {
	other := reverseSide(side)
	return pages.DualLink{
		Label: "SEND TO " + otherLabel + " →",
		Href:  state.withPanePath(other, pane.path).path(),
	}
}

// ---------------------------------------------------------------------------
// Route-state parsing and serialization
// ---------------------------------------------------------------------------

func parseDualState(values neturl.Values) dualState {
	return dualState{
		left:  parseDualPane(values, "left", "l"),
		right: parseDualPane(values, "right", "r"),
		wide:  strings.TrimSpace(values.Get("wide")) == "1",
	}
}

func parseDualPane(values neturl.Values, side string, prefix string) dualPaneState {
	return dualPaneState{
		path:     parseDualPath(values.Get(side)),
		renderTo: resolvePaneTarget(side, values.Get(side+"_render_to")),
		index:    parseDualIndex(values, prefix),
		size:     parseDualSize(values.Get(prefix + "_size")),
	}
}

// parseDualPath canonicalizes a pane content path or returns "" (the index) for
// empty, the legacy "default" sentinel, overlong, or malformed input.
func parseDualPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "default" {
		return ""
	}
	if len(raw) > maxDualPathLength {
		return ""
	}

	parsed, err := parsePanePath(raw)
	if err != nil {
		return ""
	}

	return parsed.RelPath
}

func parseDualIndex(values neturl.Values, prefix string) dualIndexState {
	idx := dualIndexState{view: viewList, sort: sortAZ}

	letter := strings.ToUpper(strings.TrimSpace(values.Get(prefix + "_letter")))
	if len(letter) == 1 && letter[0] >= 'A' && letter[0] <= 'Z' {
		idx.letter = letter
	}

	idx.school = boundDualParam(values.Get(prefix+"_school"), 100)
	idx.period = boundDualParam(values.Get(prefix+"_period"), 100)
	idx.query = boundDualParam(values.Get(prefix+"_q"), 200)

	idx.bornFrom = parseDualBornYear(values.Get(prefix + "_born_from"))
	idx.bornTo = parseDualBornYear(values.Get(prefix + "_born_to"))
	if idx.bornFrom > 0 && idx.bornTo > 0 && idx.bornFrom > idx.bornTo {
		idx.bornFrom, idx.bornTo = idx.bornTo, idx.bornFrom
	}

	if view := strings.TrimSpace(values.Get(prefix + "_view")); view == viewGrid || view == viewList {
		idx.view = view
	}

	switch sort := strings.TrimSpace(values.Get(prefix + "_sort")); sort {
	case sortAZ, sortZA, sortBirth:
		idx.sort = sort
	}

	return idx
}

func boundDualParam(raw string, max int) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > max {
		return ""
	}

	return raw
}

func parseDualBornYear(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 || value > 9999 {
		return 0
	}

	return value
}

func parseDualSize(raw string) string {
	switch strings.TrimSpace(raw) {
	case sizeSmall, sizeLarge:
		return strings.TrimSpace(raw)
	default:
		return sizeMedium
	}
}

// normalize clamps born years to the published range and drops unknown
// school/period slugs so the canonical URL never carries unresolvable state.
func (s dualState) normalize(ref dualReference) dualState {
	next := s
	next.left.index = normalizeDualIndex(next.left.index, ref)
	next.right.index = normalizeDualIndex(next.right.index, ref)
	return next
}

func normalizeDualIndex(idx dualIndexState, ref dualReference) dualIndexState {
	next := idx

	if next.bornFrom > 0 || next.bornTo > 0 {
		if ref.bornMin <= 0 || ref.bornMax <= 0 {
			next.bornFrom = 0
			next.bornTo = 0
		} else {
			next.bornFrom = clampDualBornYear(next.bornFrom, ref.bornMin, ref.bornMax)
			next.bornTo = clampDualBornYear(next.bornTo, ref.bornMin, ref.bornMax)
			if next.bornFrom > 0 && next.bornTo > 0 && next.bornFrom > next.bornTo {
				next.bornFrom, next.bornTo = next.bornTo, next.bornFrom
			}
		}
	}

	if next.school != "" {
		if _, ok := ref.schoolSlugs[next.school]; !ok {
			next.school = ""
		}
	}
	if next.period != "" {
		if _, ok := ref.periodByID[next.period]; !ok {
			next.period = ""
		}
	}

	return next
}

func clampDualBornYear(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}

	return value
}

func (s dualState) queryParams() []dualParam {
	params := []dualParam{}
	add := func(key string, value string) {
		params = append(params, dualParam{name: key, value: value})
	}

	if s.wide {
		add("wide", "1")
	}
	s.left.addParams("left", "l", add)
	s.right.addParams("right", "r", add)

	return params
}

func (s dualState) path() string {
	params := s.queryParams()
	if len(params) == 0 {
		return "/dual-mode"
	}

	parts := make([]string, len(params))
	for i, param := range params {
		parts[i] = neturl.QueryEscape(param.name) + "=" + neturl.QueryEscape(param.value)
	}

	return "/dual-mode?" + strings.Join(parts, "&")
}

func (p dualPaneState) addParams(side string, prefix string, add func(string, string)) {
	if p.path != "" {
		add(side, p.path)
	}
	if p.renderTo != reverseSide(side) {
		add(side+"_render_to", p.renderTo)
	}
	p.index.addParams(prefix, add)
	if p.size != sizeMedium {
		add(prefix+"_size", p.size)
	}
}

func (i dualIndexState) addParams(prefix string, add func(string, string)) {
	if i.letter != "" {
		add(prefix+"_letter", i.letter)
	}
	if i.school != "" {
		add(prefix+"_school", i.school)
	}
	if i.period != "" {
		add(prefix+"_period", i.period)
	}
	if i.query != "" {
		add(prefix+"_q", i.query)
	}
	if i.bornFrom > 0 {
		add(prefix+"_born_from", strconv.Itoa(i.bornFrom))
	}
	if i.bornTo > 0 {
		add(prefix+"_born_to", strconv.Itoa(i.bornTo))
	}
	if i.view != viewList {
		add(prefix+"_view", i.view)
	}
	if i.sort != sortAZ {
		add(prefix+"_sort", i.sort)
	}
}

// hiddenFieldsFor returns the canonical state parameters except the visible
// filter inputs of the given window's index form. The form's visible controls
// (school, period, name, born range) supply those values; everything else is
// preserved as hidden inputs so a filter change never loses the other window.
func (s dualState) hiddenFieldsFor(side string) []pages.DualHiddenField {
	prefix := "l"
	if side == "right" {
		prefix = "r"
	}

	visible := map[string]bool{
		prefix + "_school":    true,
		prefix + "_period":    true,
		prefix + "_q":         true,
		prefix + "_born_from": true,
		prefix + "_born_to":   true,
	}

	hidden := []pages.DualHiddenField{}
	for _, param := range s.queryParams() {
		if visible[param.name] {
			continue
		}
		hidden = append(hidden, pages.DualHiddenField{Name: param.name, Value: param.value})
	}

	return hidden
}

func (s dualState) withPanePath(side string, path string) dualState {
	next := s
	if side == "left" {
		next.left.path = path
	} else {
		next.right.path = path
	}
	return next
}

func (s dualState) withPaneRenderTo(side string, renderTo string) dualState {
	next := s
	if side == "left" {
		next.left.renderTo = renderTo
	} else {
		next.right.renderTo = renderTo
	}
	return next
}

func (s dualState) withPaneIndex(side string, idx dualIndexState) dualState {
	next := s
	if side == "left" {
		next.left.index = idx
	} else {
		next.right.index = idx
	}
	return next
}

func (s dualState) withPaneSize(side string, size string) dualState {
	next := s
	if side == "left" {
		next.left.size = size
	} else {
		next.right.size = size
	}
	return next
}

func (s dualState) withWide(wide bool) dualState {
	next := s
	next.wide = wide
	return next
}

func (s dualState) swapped() dualState {
	return dualState{left: s.right, right: s.left, wide: s.wide}
}

func (s dualState) reset() dualState {
	return dualState{
		left:  dualPaneState{renderTo: "right", index: dualIndexState{view: viewList, sort: sortAZ}, size: sizeMedium},
		right: dualPaneState{renderTo: "left", index: dualIndexState{view: viewList, sort: sortAZ}, size: sizeMedium},
		wide:  s.wide,
	}
}

func (i dualIndexState) withLetter(letter string) dualIndexState {
	next := i
	next.letter = letter
	return next
}

func (i dualIndexState) withView(view string) dualIndexState {
	next := i
	next.view = view
	return next
}

func (i dualIndexState) withSort(sort string) dualIndexState {
	next := i
	next.sort = sort
	return next
}

func (i dualIndexState) repositoryFilter(periodStart int, periodEnd int) repositories.ArtistIndexFilter {
	filter := repositories.ArtistIndexFilter{
		Query:        i.query,
		Letter:       i.letter,
		School:       i.school,
		BornActive:   i.bornFrom > 0 || i.bornTo > 0,
		BornFrom:     i.bornFrom,
		BornTo:       i.bornTo,
		PeriodActive: periodStart > 0 || periodEnd > 0,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		Limit:        dualIndexPageSize,
	}

	switch i.sort {
	case sortZA:
		filter.Sort = repositories.ArtistSortNameDesc
	case sortBirth:
		filter.Sort = repositories.ArtistSortBirth
	default:
		filter.Sort = repositories.ArtistSortNameAsc
	}

	return filter
}

// ---------------------------------------------------------------------------
// Reference data
// ---------------------------------------------------------------------------

func loadDualReference(app core.App) (dualReference, error) {
	var ref dualReference

	schoolRecords, err := app.FindRecordsByFilter(constants.CollectionSchools, "", "+name", 0, 0)
	if err != nil {
		return ref, err
	}
	ref.schoolSlugs = map[string]string{}
	ref.schoolByID = map[string]string{}
	for _, record := range schoolRecords {
		slug := record.GetString("slug")
		name := record.GetString("name")
		if name == "" {
			continue
		}
		ref.schoolByID[record.Id] = name
		if slug != "" {
			ref.schoolSlugs[slug] = name
		}
	}

	periodRecords, err := app.FindRecordsByFilter("art_periods", "", "+start,+name", 0, 0)
	if err != nil {
		return ref, err
	}
	ref.periodByID = map[string]dualPeriod{}
	for _, record := range periodRecords {
		period := dualPeriod{
			id:    record.Id,
			name:  record.GetString("name"),
			start: record.GetInt("start"),
			end:   record.GetInt("end"),
		}
		ref.periods = append(ref.periods, period)
		ref.periodByID[record.Id] = period
	}

	repo := repositories.NewArtistIndexRepository(app)
	ref.bornMin, ref.bornMax, err = repo.BirthYearBounds()
	if err != nil {
		return ref, err
	}

	return ref, nil
}

// ---------------------------------------------------------------------------
// Window views
// ---------------------------------------------------------------------------

func buildDualIndexView(app *pocketbase.PocketBase, side string, pane dualPaneState, state dualState, ref dualReference) (pages.DualIndexView, error) {
	idx := pane.index
	prefix := "l"
	if side == "right" {
		prefix = "r"
	}

	periodStart := 0
	periodEnd := 0
	if idx.period != "" {
		if period, ok := ref.periodByID[idx.period]; ok {
			periodStart = period.start
			periodEnd = period.end
		}
	}

	repo := repositories.NewArtistIndexRepository(app)
	filter := idx.repositoryFilter(periodStart, periodEnd)

	total, err := repo.CountArtists(filter)
	if err != nil {
		return pages.DualIndexView{}, err
	}

	indexed, err := repo.ListArtists(filter)
	if err != nil {
		return pages.DualIndexView{}, err
	}

	availableLetters, err := repo.ListAvailableLetters(repositories.ArtistIndexFilter{})
	if err != nil {
		return pages.DualIndexView{}, err
	}

	view := pages.DualIndexView{
		View:        idx.view,
		Hidden:      state.hiddenFieldsFor(side),
		Letters:     buildDualLetters(side, idx, state, availableLetters),
		AllUrl:      state.withPaneIndex(side, idx.withLetter("")).path(),
		SchoolGroup: buildDualSchoolGroup(side, idx, ref),
		PeriodGroup: buildDualPeriodGroup(side, idx, ref),
		NameField:   buildDualNameField(side, idx),
		GridHref:    state.withPaneIndex(side, idx.withView(viewGrid)).path(),
		ListHref:    state.withPaneIndex(side, idx.withView(viewList)).path(),
		SortHref:    state.withPaneIndex(side, idx.withSort(nextDualSort(idx.sort))).path(),
		ResetHref:   state.withPaneIndex(side, dualIndexState{view: viewList, sort: sortAZ}).path(),
		SortLabel:   dualSortLabel(idx.sort),
		FilterNote:  dualFilterNote(idx, ref),
		Total:       total,
		Artists:     buildDualArtistRows(pane, state, ref, indexed),
	}

	if ref.bornMin > 0 && ref.bornMax > 0 {
		view.HasBornRange = true
		view.BornRange = dto.RangeField{
			Label:     "BORN BETWEEN",
			FromID:    prefix + "-born-from",
			FromName:  prefix + "_born_from",
			FromValue: effectiveDualBorn(idx.bornFrom, ref.bornMin),
			ToID:      prefix + "-born-to",
			ToName:    prefix + "_born_to",
			ToValue:   effectiveDualBorn(idx.bornTo, ref.bornMax),
			Min:       ref.bornMin,
			Max:       ref.bornMax,
			Step:      1,
		}
	}

	return view, nil
}

func buildDualLetters(side string, idx dualIndexState, state dualState, available []string) []pages.DualLetter {
	availableSet := map[string]bool{}
	for _, letter := range available {
		availableSet[letter] = true
	}

	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letters := make([]pages.DualLetter, 0, len(alphabet))
	for _, letter := range alphabet {
		value := string(letter)
		entry := pages.DualLetter{
			Label:    value,
			Enabled:  availableSet[value],
			Selected: idx.letter == value,
		}
		if entry.Enabled {
			entry.Href = state.withPaneIndex(side, idx.withLetter(value)).path()
		}
		letters = append(letters, entry)
	}

	return letters
}

func buildDualSchoolGroup(side string, idx dualIndexState, ref dualReference) dto.ChipGroup {
	options := []dto.ChipOption{{Label: "ALL", Value: "", Checked: idx.school == ""}}

	slugs := make([]string, 0, len(ref.schoolSlugs))
	for slug := range ref.schoolSlugs {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	for _, slug := range slugs {
		options = append(options, dto.ChipOption{
			Label:   ref.schoolSlugs[slug],
			Value:   slug,
			Checked: idx.school == slug,
		})
	}

	return dto.ChipGroup{Legend: "SCHOOL", Name: dualPrefix(side) + "_school", Inline: true, Options: options}
}

func buildDualPeriodGroup(side string, idx dualIndexState, ref dualReference) dto.ChipGroup {
	options := []dto.ChipOption{{Label: "ALL", Value: "", Checked: idx.period == ""}}

	ids := make([]string, 0, len(ref.periodByID))
	for id := range ref.periodByID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool {
		return ref.periodByID[ids[a]].start < ref.periodByID[ids[b]].start
	})

	for _, id := range ids {
		period := ref.periodByID[id]
		if period.name == "" {
			continue
		}
		options = append(options, dto.ChipOption{
			Label:   period.name,
			Value:   id,
			Checked: idx.period == id,
		})
	}

	return dto.ChipGroup{Legend: "PERIOD", Name: dualPrefix(side) + "_period", Inline: true, Options: options}
}

func buildDualNameField(side string, idx dualIndexState) dto.Field {
	return dto.Field{
		ID:          dualPrefix(side) + "-name",
		Name:        dualPrefix(side) + "_q",
		Label:       "NAME CONTAINS",
		Type:        "search",
		Value:       idx.query,
		Placeholder: "e.g. van",
	}
}

func buildDualArtistRows(pane dualPaneState, state dualState, ref dualReference, indexed []repositories.IndexedArtist) []pages.DualArtistRow {
	rows := make([]pages.DualArtistRow, 0, len(indexed))
	for _, entry := range indexed {
		record := entry.Record
		rows = append(rows, pages.DualArtistRow{
			Name:        record.GetString("filing_name"),
			Dates:       dualArtistDates(record),
			School:      dualSchoolNames(record.GetStringSlice("school"), ref),
			Period:      dualPeriodForBirth(ref.periods, record.GetInt("year_of_birth")),
			Form:        record.GetString("profession"),
			Thumb:       urlutils.GenerateArtistPortraitImageURL(record, urlutils.DeliveryProfileCardAndArtistIndex, ""),
			Unavailable: !entry.Available,
		})

		// The row href is a content link: it lands in the window the pane's
		// routing targets. Unavailable rows carry no href.
		if entry.Available {
			rows[len(rows)-1].Href = state.withPanePath(pane.renderTo, urlutils.GenerateArtistUrlFromRecord(record)).path()
		}
	}

	return rows
}

// findPublishedArtist returns a published artist with complete identity, or
// sql.ErrNoRows when the artist is missing, unpublished, or lacks either
// authoritative identity field. It delegates to the record repository so the
// denial predicate stays a single source of truth.
func findPublishedArtist(app core.App, id string) (*core.Record, error) {
	return repositories.NewArtistRecordRepository(app).FindPublishedArtist(id)
}

// findPublishedArtwork returns a published artwork record, or sql.ErrNoRows when
// the artwork is missing or unpublished.
func findPublishedArtwork(app core.App, id string) (*core.Record, error) {
	return app.FindRecordById(constants.CollectionArtworks, id, func(q *dbx.SelectQuery) error {
		q.AndWhere(dbx.NewExp("published = true"))
		return nil
	})
}

// resolvePaneCanonicalPath returns the canonical public record URL for a pane's
// content, or "" (the index) when the pane is at the index, its record is
// missing or unpublished, its author is missing or unpublished, or its path is
// malformed. Artwork resolution ignores the requested artist segment and always
// uses the work's published author, so a mismatched segment cannot leak into
// the shareable URL.
func resolvePaneCanonicalPath(app core.App, pane dualPaneState) (string, error) {
	if pane.path == "" {
		return "", nil
	}

	parsed, err := parsePanePath(pane.path)
	if err != nil {
		return "", nil
	}

	switch parsed.Kind {
	case "artist":
		artist, err := findPublishedArtist(app, parsed.Id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", nil
			}
			return "", err
		}

		return urlutils.GenerateArtistUrlFromRecord(artist), nil

	case "artwork":
		work, err := findPublishedArtwork(app, parsed.Id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", nil
			}
			return "", err
		}

		authorIDs := work.GetStringSlice("author")
		if len(authorIDs) == 0 {
			return "", nil
		}

		artist, err := findPublishedArtist(app, authorIDs[0])
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", nil
			}
			return "", err
		}

		return urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
			ArtistName:   artist.GetString("name"),
			ArtistId:     artist.Id,
			ArtworkTitle: work.GetString("title"),
			ArtworkId:    work.GetString("id"),
		}), nil

	default:
		return "", nil
	}
}

func buildDualArtistRecord(app *pocketbase.PocketBase, side string, pane dualPaneState, state dualState, ref dualReference) (pages.DualArtistRecord, error) {
	parsed, _ := parsePanePath(pane.path)

	repo := repositories.NewArtistRecordRepository(app)
	artist, err := repo.FindPublishedArtist(parsed.Id)
	if err != nil {
		return pages.DualArtistRecord{}, err
	}

	expectedSlug := utils.GenerateArtistSlug(artist)

	workCount, err := repo.CountPublishedWorks(artist.Id)
	if err != nil {
		return pages.DualArtistRecord{}, err
	}
	works, err := repo.ListPublishedWorks(artist.Id, 0)
	if err != nil {
		return pages.DualArtistRecord{}, err
	}

	schoolNames, err := repo.ListSchoolNames(artist.GetStringSlice("school"))
	if err != nil {
		return pages.DualArtistRecord{}, err
	}

	periodRecords, err := repo.ListMatchingArtPeriods(artist.GetInt("year_of_birth"))
	if err != nil {
		return pages.DualArtistRecord{}, err
	}
	period := dualUnambiguousPeriodName(periodRecords)

	glossaryEntries, glossaryErr := glossary.GetGlossaryEntries(app)
	if glossaryErr != nil {
		app.Logger().Warn("Failed to load glossary entries", "error", glossaryErr)
	}
	bio := dualAnnotatedHTML(artist.GetString("bio"), glossaryEntries)

	periodSong, err := repo.MatchPeriodSong(artist.GetInt("year_of_birth"))
	if err != nil {
		return pages.DualArtistRecord{}, err
	}

	aliases := dualResolveAliases(app, artist.GetStringSlice("also_known_as"))

	record := pages.DualArtistRecord{
		FilingName: artist.GetString("filing_name"),
		ShortName:  artist.GetString("short_name"),
		Dates:      dualArtistDates(artist),
		Portrait:   urlutils.GenerateArtistPortraitImageURL(artist, urlutils.DeliveryProfilePortraitRecordAndWorkFallback, ""),
		Meta:       dualArtistMeta(schoolNames, period, artist.GetString("profession"), aliases),
		Bio:        bio,
		Heading:    dualWorksHeading(workCount, len(works)),
		Works:      buildDualWorkCards(side, pane, state, artist, works),
		Music:      buildDualMusic(periodSong),
		Citation: components.Citation{
			Key:   "wga-" + artist.GetString("slug"),
			Title: artist.GetString("filing_name"),
			URL:   utils.AssetUrl("/artists/" + expectedSlug),
		},
	}

	return record, nil
}

func buildDualWorkRecord(app *pocketbase.PocketBase, side string, pane dualPaneState, state dualState, ref dualReference) (pages.DualWorkRecord, string, string, error) {
	parsed, _ := parsePanePath(pane.path)

	work, err := findPublishedArtwork(app, parsed.Id)
	if err != nil {
		return pages.DualWorkRecord{}, "", "", err
	}

	artist, artistPath, err := dualWorkAuthor(app, work)
	if err != nil {
		return pages.DualWorkRecord{}, "", "", err
	}

	location, dimensions := dualArtworkLocationAndDimensions(work.GetString("comment"))
	technique := dualTrimDimensions(work.GetString("technique"), dimensions)

	year := work.GetInt("year")

	glossaryEntries, glossaryErr := glossary.GetGlossaryEntries(app)
	if glossaryErr != nil {
		app.Logger().Warn("Failed to load glossary entries", "error", glossaryErr)
	}
	comment := dualAnnotatedHTML(work.GetString("comment"), glossaryEntries)

	image := urlutils.GenerateArtworkImageURL(work, dualSizeProfile(pane.size), "")
	zoom := urlutils.GenerateArtworkImageURL(work, urlutils.DeliveryProfileViewer, "")

	byline := artist.GetString("filing_name")
	if year > 0 {
		byline = byline + " · " + strconv.Itoa(year)
	}
	byline = byline + " →"

	sizes := []pages.DualLink{}
	for _, size := range []string{sizeSmall, sizeMedium, sizeLarge} {
		sizes = append(sizes, pages.DualLink{
			Label:    strconv.Itoa(dualSizeWidth(size)),
			Href:     state.withPaneSize(side, size).path(),
			Selected: pane.size == size,
		})
	}

	artType := dualWorkArtType(app, work)

	record := pages.DualWorkRecord{
		Title:       work.GetString("title"),
		Byline:      byline,
		ArtistHref:  state.withPanePath(pane.renderTo, artistPath).path(),
		Image:       image,
		Zoom:        zoom,
		PlateClass:  dualSizePlateClass(pane.size),
		Sizes:       sizes,
		SizeCaption: fmt.Sprintf("REPRODUCTION AT %dPX WIDE", dualSizeWidth(pane.size)),
		Meta:        dualWorkMeta(technique, dimensions, artType, location),
		Comment:     comment,
		ArtworkID:   work.Id,
		Citation: components.Citation{
			Key:   "wga-" + utils.Slugify(work.GetString("title")),
			Title: work.GetString("title") + " by " + artist.GetString("filing_name"),
			URL:   utils.AssetUrl(artistPath + "/" + utils.Slugify(work.GetString("title")) + "-" + work.Id),
		},
	}

	return record, artist.GetString("short_name"), artistPath, nil
}

func dualWorkAuthor(app core.App, work *core.Record) (*core.Record, string, error) {
	authorIDs := work.GetStringSlice("author")
	if len(authorIDs) == 0 {
		return nil, "", sql.ErrNoRows
	}

	artist, err := findPublishedArtist(app, authorIDs[0])
	if err != nil {
		return nil, "", err
	}

	artistPath := urlutils.GenerateArtistUrlFromRecord(artist)

	return artist, artistPath, nil
}

func buildDualWorkCards(side string, pane dualPaneState, state dualState, artist *core.Record, works []*core.Record) []pages.DualCard {
	cards := make([]pages.DualCard, 0, len(works))
	for _, work := range works {
		card := pages.DualCard{
			Title:     work.GetString("title"),
			Thumb:     urlutils.GenerateArtworkImageURL(work, urlutils.DeliveryProfileCardAndArtistIndex, ""),
			ArtworkID: work.Id,
			Href: state.withPanePath(pane.renderTo, urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
				ArtistName:   artist.GetString("name"),
				ArtistId:     artist.Id,
				ArtworkTitle: work.GetString("title"),
				ArtworkId:    work.GetString("id"),
			})).path(),
		}
		if year := work.GetInt("year"); year > 0 {
			card.Meta = strconv.Itoa(year)
		}
		cards = append(cards, card)
	}

	return cards
}

// ---------------------------------------------------------------------------
// Small record-shaping helpers
// ---------------------------------------------------------------------------

func dualArtistDates(artist *core.Record) string {
	birth := artist.GetInt("year_of_birth")
	death := artist.GetInt("year_of_death")

	switch {
	case birth > 0 && death > 0:
		return strconv.Itoa(birth) + "–" + strconv.Itoa(death)
	case birth > 0:
		return "b. " + strconv.Itoa(birth)
	case death > 0:
		return "d. " + strconv.Itoa(death)
	default:
		return ""
	}
}

func dualSchoolNames(ids []string, ref dualReference) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := ref.schoolByID[id]; ok {
			names = append(names, name)
		}
	}

	return strings.Join(names, ", ")
}

// dualPeriodForBirth returns the art-period label whose range unambiguously
// contains the birth year, or "" when unknown, unmatched, or ambiguous.
func dualPeriodForBirth(periods []dualPeriod, birth int) string {
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

func dualUnambiguousPeriodName(records []*core.Record) string {
	if len(records) != 1 {
		return ""
	}

	return records[0].GetString("name")
}

func dualArtistMeta(schools []string, period string, profession string, aliases string) []components.MetaEntry {
	entries := []components.MetaEntry{}
	if len(schools) > 0 {
		entries = append(entries, components.MetaEntry{Label: "SCHOOLS", Value: strings.Join(schools, ", ")})
	}
	if period != "" {
		entries = append(entries, components.MetaEntry{Label: "PERIOD", Value: period})
	}
	if profession != "" {
		entries = append(entries, components.MetaEntry{Label: "PROFESSION", Value: profession})
	}
	if aliases != "" {
		entries = append(entries, components.MetaEntry{Label: "ALIASES", Value: aliases})
	}

	return entries
}

func dualWorksHeading(workCount int, shown int) string {
	if workCount > shown {
		return fmt.Sprintf("WORKS IN ARCHIVE (%d OF %d)", shown, workCount)
	}

	return fmt.Sprintf("WORKS IN ARCHIVE (%d)", workCount)
}

// dualAnnotatedHTML sanitises persisted HTML against the shared WGA sanitisation
// policy before glossary annotation, and returns the sanitised HTML even when no
// glossary entries are available (including glossary-load failure). It is the
// single boundary through which all dual-pane prose reaches templ.Raw.
func dualAnnotatedHTML(html string, entries []glossary.GlossaryEntry) string {
	sanitized := glossary.SanitizeDefinition(html)
	if len(entries) == 0 {
		return sanitized
	}

	return glossary.AnnotateHTML(sanitized, entries)
}

func dualResolveAliases(app core.App, ids []string) string {
	if len(ids) == 0 {
		return ""
	}

	records, err := app.FindRecordsByIds(constants.CollectionArtists, ids)
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(records))
	for _, record := range records {
		if name := record.GetString("filing_name"); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	return strings.Join(names, ", ")
}

func buildDualMusic(song *repositories.PeriodSong) components.MusicPeriodCard {
	if song == nil || song.Record == nil {
		return components.MusicPeriodCard{}
	}

	piece := strings.TrimSpace(song.Record.GetString("title"))
	if piece == "" || song.Record.GetString("source") == "" {
		return components.MusicPeriodCard{}
	}

	return components.MusicPeriodCard{
		SongID:    song.Record.Id,
		Piece:     piece,
		PlayerURL: "/player?song=" + song.Record.Id,
	}
}

func dualArtworkLocationAndDimensions(comment string) (string, string) {
	parts := strings.Split(tmplUtils.StripHtmlTags(comment), " · ")
	if len(parts) < 3 {
		return "", ""
	}

	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
}

func dualTrimDimensions(technique string, dimensions string) string {
	if dimensions == "" {
		return technique
	}

	return strings.TrimSpace(strings.TrimSuffix(technique, ", "+dimensions))
}

func dualWorkArtType(app core.App, work *core.Record) string {
	for _, typeID := range work.GetStringSlice("type") {
		artType, err := app.FindRecordById(constants.CollectionArtTypes, typeID)
		if err == nil {
			return artType.GetString("name")
		}
	}

	return ""
}

func dualWorkMeta(technique string, dimensions string, artType string, location string) []components.MetaEntry {
	entries := []components.MetaEntry{}
	for _, entry := range []components.MetaEntry{
		{Label: "MEDIUM", Value: technique},
		{Label: "DIMENSIONS", Value: dimensions},
		{Label: "TYPE", Value: artType},
		{Label: "LOCATION", Value: location},
	} {
		if entry.Value != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

func dualSizeProfile(size string) urlutils.DeliveryProfile {
	switch size {
	case sizeSmall:
		return urlutils.DeliveryProfilePostcardSmallDualPlate
	case sizeLarge:
		return urlutils.DeliveryProfileDualLargePlate
	default:
		return urlutils.DeliveryProfileDualMediumPlate
	}
}

func dualSizeWidth(size string) int {
	switch size {
	case sizeSmall:
		return 700
	case sizeLarge:
		return 1600
	default:
		return 1100
	}
}

func dualSizePlateClass(size string) string {
	switch size {
	case sizeSmall:
		return "h-[300px]"
	case sizeLarge:
		return "h-[680px]"
	default:
		return "h-[460px]"
	}
}

func dualPrefix(side string) string {
	if side == "right" {
		return "r"
	}

	return "l"
}

func dualSortLabel(sort string) string {
	switch sort {
	case sortZA:
		return "Z–A"
	case sortBirth:
		return "BIRTH YEAR"
	default:
		return "A–Z"
	}
}

func nextDualSort(sort string) string {
	switch sort {
	case sortAZ:
		return sortZA
	case sortZA:
		return sortBirth
	default:
		return sortAZ
	}
}

func dualFilterNote(idx dualIndexState, ref dualReference) string {
	parts := []string{}
	if idx.letter != "" {
		parts = append(parts, "· "+idx.letter)
	}
	if idx.school != "" {
		parts = append(parts, "· "+strings.ToUpper(ref.schoolSlugs[idx.school]))
	}
	if idx.period != "" {
		parts = append(parts, "· "+strings.ToUpper(ref.periodByID[idx.period].name))
	}
	if idx.bornFrom > 0 || idx.bornTo > 0 {
		from := idx.bornFrom
		if from == 0 {
			from = ref.bornMin
		}
		to := idx.bornTo
		if to == 0 {
			to = ref.bornMax
		}
		parts = append(parts, "· BORN "+strconv.Itoa(from)+"–"+strconv.Itoa(to))
	}

	return strings.Join(parts, " ")
}

func effectiveDualBorn(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}

// ---------------------------------------------------------------------------
// Pane path and routing helpers
// ---------------------------------------------------------------------------

func parsePanePath(raw string) (panePathDto, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "default" || normalized == "" {
		return panePathDto{Kind: "default", RelPath: "default"}, nil
	}

	normalized = "/" + strings.Trim(normalized, "/")
	parts := strings.Split(strings.Trim(normalized, "/"), "/")

	switch {
	case len(parts) == 2 && parts[0] == "artists":
		return panePathDto{
			Kind:    "artist",
			Id:      utils.ExtractIdFromString(parts[1]),
			RelPath: normalized,
		}, nil
	case len(parts) == 3 && parts[0] == "artists":
		return panePathDto{
			Kind:    "artwork",
			Id:      utils.ExtractIdFromString(parts[2]),
			RelPath: normalized,
		}, nil
	case len(parts) == 4 && parts[0] == "artists" && parts[2] == "artworks":
		return panePathDto{
			Kind:    "artwork",
			Id:      utils.ExtractIdFromString(parts[3]),
			RelPath: normalized,
		}, nil
	case len(parts) == 2 && parts[0] == "artworks":
		return panePathDto{
			Kind:    "artwork",
			Id:      utils.ExtractIdFromString(parts[1]),
			RelPath: normalized,
		}, nil
	default:
		return panePathDto{}, errs.ErrUnknownDualPane
	}
}

func reverseSide(side string) string {
	switch side {
	case "left":
		return "right"
	case "right":
		return "left"
	default:
		return ""
	}
}

func resolvePaneTarget(side string, requestedTarget string) string {
	requestedTarget = strings.TrimSpace(requestedTarget)
	if requestedTarget == side || requestedTarget == reverseSide(side) {
		return requestedTarget
	}

	return reverseSide(side)
}

// ---------------------------------------------------------------------------
// Lookup
// ---------------------------------------------------------------------------

func renderDualLookupResults(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	queryValues := c.Request.URL.Query()
	content, err := getDualLookupResults(app, queryValues.Get("kind"), queryValues.Get("q"))
	if err != nil {
		app.Logger().Error("Error getting dual lookup results", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	var buff bytes.Buffer
	if err := pages.DualLookupResultContent(content).Render(context.Background(), &buff); err != nil {
		app.Logger().Error("Error rendering dual lookup results", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

func getDualLookupResults(app core.App, kind string, query string) (pages.DualLookupResults, error) {
	content := pages.DualLookupResults{
		Kind:    resolveDualLookupKind(kind),
		Query:   strings.TrimSpace(query),
		Results: []pages.DualLookupResult{},
	}

	if content.Query == "" {
		return content, nil
	}

	if utf8.RuneCountInString(content.Query) < dualLookupMinimumQueryRunes {
		content.QueryTooShort = true
		return content, nil
	}

	var err error

	switch content.Kind {
	case "artist":
		content.Results, err = getArtistLookupResults(app, content.Query)
	case "artwork":
		content.Results, err = getArtworkLookupResults(app, content.Query)
	}

	return content, err
}

func resolveDualLookupKind(kind string) string {
	if strings.TrimSpace(kind) == "artwork" {
		return "artwork"
	}

	return "artist"
}

func getArtistLookupResults(app core.App, query string) ([]pages.DualLookupResult, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		"published = true && "+dualArtistIdentityFilter+" && filing_name ~ {:query}",
		"+filing_name,+id",
		dualLookupLimit,
		0,
		dbx.Params{"query": query},
	)
	if err != nil {
		return nil, err
	}

	results := make([]pages.DualLookupResult, 0, len(records))
	for _, record := range records {
		results = append(results, pages.DualLookupResult{
			Url: urlutils.GenerateArtistUrl(urlutils.ArtistUrlDTO{
				ArtistId:   record.Id,
				ArtistName: record.GetString("name"),
			}),
			Label: record.GetString("filing_name"),
		})
	}

	return results, nil
}

func getArtworkLookupResults(app core.App, query string) ([]pages.DualLookupResult, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollectionArtworks,
		"published = true && author:length > 0 && title ~ {:query}",
		"+title,+id",
		dualLookupLimit,
		0,
		dbx.Params{"query": query},
	)
	if err != nil {
		return nil, err
	}

	artistsByID, err := getLookupArtistsByID(app, uniqueLookupArtistIDs(records))
	if err != nil {
		return nil, err
	}

	results := make([]pages.DualLookupResult, 0, len(records))
	for _, record := range records {
		authorIDs := record.GetStringSlice("author")
		if len(authorIDs) == 0 {
			continue
		}

		artist, ok := artistsByID[authorIDs[0]]
		if !ok {
			continue
		}

		results = append(results, pages.DualLookupResult{
			Url: urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
				ArtistId:     artist.Id,
				ArtistName:   artist.GetString("name"),
				ArtworkId:    record.Id,
				ArtworkTitle: record.GetString("title"),
			}),
			Label:   record.GetString("title"),
			Context: artist.GetString("filing_name"),
		})
	}

	return results, nil
}

func uniqueLookupArtistIDs(records []*core.Record) []string {
	seen := map[string]struct{}{}
	artistIDs := make([]string, 0, len(records))

	for _, record := range records {
		for _, artistID := range record.GetStringSlice("author") {
			if artistID == "" {
				continue
			}
			if _, exists := seen[artistID]; exists {
				continue
			}
			seen[artistID] = struct{}{}
			artistIDs = append(artistIDs, artistID)
		}
	}

	return artistIDs
}

func getLookupArtistsByID(app core.App, artistIDs []string) (map[string]*core.Record, error) {
	if len(artistIDs) == 0 {
		return map[string]*core.Record{}, nil
	}

	params := dbx.Params{}
	conditions := make([]string, 0, len(artistIDs))
	for index, artistID := range artistIDs {
		key := fmt.Sprintf("artist_id_%d", index)
		conditions = append(conditions, fmt.Sprintf("id = {:%s}", key))
		params[key] = artistID
	}

	records, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		"("+strings.Join(conditions, " || ")+") && "+dualArtistIdentityFilter,
		"+filing_name,+id",
		len(artistIDs),
		0,
		params,
	)
	if err != nil {
		return nil, err
	}

	artistsByID := make(map[string]*core.Record, len(records))
	for _, record := range records {
		artistsByID[record.Id] = record
	}

	return artistsByID, nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/dual-mode", func(c *core.RequestEvent) error {
			return renderDualModePage(app, c)
		})
		se.Router.GET(dualLookupPath, func(c *core.RequestEvent) error {
			return renderDualLookupResults(app, c)
		})
		return se.Next()
	})
}
