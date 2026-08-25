package tours

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	wgaurl "github.com/blackfyre/wga/internal/utils/url"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
)

var ErrNotFound = errors.New("guided tour not found")

var filterKinds = map[string]string{
	"": "All", "survey": "Survey", "artist": "Artist", "site": "Site", "theme": "Theme",
}

type Service struct {
	app       core.App
	sanitizer *bluemonday.Policy
}

func NewService(app core.App) *Service {
	return &Service{app: app, sanitizer: bluemonday.UGCPolicy()}
}

func (s *Service) Index(kind string) (dto.TourIndex, error) {
	if _, ok := filterKinds[kind]; !ok {
		kind = ""
	}
	filter := "publication_status = 'published' && published_revision != ''"
	params := map[string]any{}
	if kind != "" {
		filter += " && kind = {:kind}"
		params["kind"] = kind
	}
	records, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTours, filter, "+series_position,+title,+id", 0, 0, params)
	if err != nil {
		return dto.TourIndex{}, err
	}

	view := dto.TourIndex{Filter: kind, Filters: buildFilters(kind)}
	for _, record := range records {
		editor, revision, ok := s.publishedContext(record)
		if !ok {
			continue
		}
		card := dto.TourCard{
			Slug: record.GetString("slug"), Title: record.GetString("title"), Blurb: record.GetString("blurb"),
			Kind: record.GetString("kind"), Editor: editor.GetString("name"), Revision: revisionLabel(revision),
			Number: record.GetString("tour_number"), Year: record.GetInt("revised_year"),
			PresentationStatus: record.GetString("presentation_status"),
		}
		if card.Year == 0 {
			card.Year = record.GetInt("published_year")
		}
		card.ImageURL = s.firstImage(record.Id, revision.Id, wgaurl.DeliveryProfileGuidedTourCard)
		if card.PresentationStatus == "original" {
			view.Original = append(view.Original, card)
			continue
		}
		pages, err := s.rebuiltPageCount(record.Id, revision.Id)
		if err != nil {
			continue
		}
		card.Pages = pages
		view.Rebuilt = append(view.Rebuilt, card)
	}
	return view, nil
}

// rebuiltPageCount returns the total addressed pages for a rebuilt tour and
// fails closed when the tour has no pages or any page lacks its required source
// provenance. This prevents a malformed rebuilt tour from appearing openable on
// the index while its addressed route would then return 404.
func (s *Service) rebuiltPageCount(tourID string, revisionID string) (int, error) {
	pageRecords, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourPages,
		"tour = {:tour} && revision = {:revision}", "+page_position,+id", 0, 0,
		map[string]any{"tour": tourID, "revision": revisionID})
	if err != nil {
		return 0, err
	}
	if len(pageRecords) == 0 {
		return 0, ErrNotFound
	}
	for _, page := range pageRecords {
		if page.GetString("source_path") == "" || page.GetString("source_hash") == "" {
			return 0, ErrNotFound
		}
	}
	total := 1 + len(pageRecords)
	sources, err := s.sources(tourID, revisionID)
	if err != nil {
		return 0, err
	}
	if len(sources) > 0 {
		total++
	}
	return total, nil
}

func (s *Service) Page(slug string, address int) (dto.TourPage, error) {
	if address < 1 {
		return dto.TourPage{}, ErrNotFound
	}
	record, err := s.app.FindFirstRecordByFilter(constants.CollectionGuidedTours,
		"slug = {:slug} && publication_status = 'published' && published_revision != ''", map[string]any{"slug": slug})
	if err != nil {
		return dto.TourPage{}, ErrNotFound
	}
	editor, revision, ok := s.publishedContext(record)
	if !ok {
		return dto.TourPage{}, ErrNotFound
	}

	view := dto.TourPage{
		Slug: record.GetString("slug"), TourTitle: record.GetString("title"), Blurb: record.GetString("blurb"),
		Kind: record.GetString("kind"), Editor: editor.GetString("name"), Revision: revisionLabel(revision),
		Number: record.GetString("tour_number"), PublishedYear: record.GetInt("published_year"),
		RevisedYear:        record.GetInt("revised_year"),
		PresentationStatus: record.GetString("presentation_status"), Address: address,
		RevisionSourceHash: revision.GetString("source_hash"),
	}
	if view.PresentationStatus == "original" {
		if address != 1 {
			return dto.TourPage{}, ErrNotFound
		}
		view.TotalPages = 1
		view.PageType = "title"
		view.PageTitle = view.TourTitle
		view.LegacyURL = safeLegacyURL(record.GetString("legacy_url"))
		view.Contents = []dto.TourContentsItem{{Number: 1, Title: view.TourTitle, Href: tourURL(view.Slug, 1), Current: true}}
		return view, nil
	}
	if view.PresentationStatus != "rebuilt" {
		return dto.TourPage{}, ErrNotFound
	}

	pageRecords, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourPages,
		"tour = {:tour} && revision = {:revision}", "+page_position,+id", 0, 0,
		map[string]any{"tour": record.Id, "revision": revision.Id})
	if err != nil {
		return dto.TourPage{}, err
	}
	for _, page := range pageRecords {
		if page.GetString("source_path") == "" || page.GetString("source_hash") == "" {
			return dto.TourPage{}, ErrNotFound
		}
	}
	sources, err := s.sources(record.Id, revision.Id)
	if err != nil {
		return dto.TourPage{}, err
	}
	view.TotalPages = 1 + len(pageRecords)
	if len(sources) > 0 {
		view.TotalPages++
	}
	if address > view.TotalPages {
		return dto.TourPage{}, ErrNotFound
	}

	sections, err := s.sectionNames(record.Id, revision.Id)
	if err != nil {
		return dto.TourPage{}, err
	}
	view.Contents = buildContents(view.Slug, view.TourTitle, pageRecords, sections, len(sources) > 0, address)
	if address > 1 {
		view.PreviousURL = tourURL(view.Slug, address-1)
		view.PreviousLabel = contentsTitle(view.Contents, address-1)
	}
	if address < view.TotalPages {
		view.NextURL = tourURL(view.Slug, address+1)
		view.NextLabel = contentsTitle(view.Contents, address+1)
	}

	if address == 1 {
		view.PageType = "title"
		view.PageTitle = view.TourTitle
		view.DisplayURL = s.firstImage(record.Id, revision.Id, wgaurl.DeliveryProfileTourTitlePlate)
		return view, nil
	}
	if address == view.TotalPages && len(sources) > 0 {
		view.PageType = "sources"
		view.PageTitle = "Sources"
		view.Sources = sources
		return view, nil
	}

	page := pageRecords[address-2]
	view.PageType = page.GetString("page_type")
	view.PageTitle = page.GetString("title")
	view.Dateline = page.GetString("dateline")
	view.Section = sections[page.GetString("section")]
	view.SourcePath = page.GetString("source_path")
	view.SourceHash = page.GetString("source_hash")
	view.Blocks, err = s.blocks(page.Id)
	if err != nil {
		return dto.TourPage{}, err
	}
	if view.PageType == "list" {
		view.IndexRows, err = s.indexRows(page.Id)
		if err != nil {
			return dto.TourPage{}, err
		}
	}
	if view.PageType == "picture" {
		artwork, ok := s.publishedArtwork(page.GetString("artwork"))
		if !ok {
			return dto.TourPage{}, ErrNotFound
		}
		view.DisplayURL = artworkImageURL(artwork, wgaurl.DeliveryProfileArtworkRecordTourPage)
		view.ZoomURL = artworkImageURL(artwork, wgaurl.DeliveryProfileViewer)
		view.ArtworkURL = safeInternalPath(page.GetString("work_target_path"))
		if view.ArtworkURL == "" {
			view.ArtworkURL = "/artworks/" + artwork.Id
		}
		view.ArtworkAlt = artwork.GetString("title")
		view.ArtworkCredit = page.GetString("credit")
	}
	return view, nil
}

func (s *Service) publishedContext(tour *core.Record) (*core.Record, *core.Record, bool) {
	editor, err := s.app.FindRecordById(constants.CollectionTourEditors, tour.GetString("editor"))
	if err != nil {
		return nil, nil, false
	}
	revision, err := s.app.FindRecordById(constants.CollectionGuidedTourRevisions, tour.GetString("published_revision"))
	if err != nil || revision.GetString("tour") != tour.Id || revision.GetString("source_hash") == "" {
		return nil, nil, false
	}
	return editor, revision, true
}

func (s *Service) firstImage(tourID string, revisionID string, profile wgaurl.DeliveryProfile) string {
	pages, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourPages,
		"tour = {:tour} && revision = {:revision} && page_type = 'picture' && artwork != ''",
		"+page_position,+id", 1, 0, map[string]any{"tour": tourID, "revision": revisionID})
	if err != nil || len(pages) == 0 {
		return ""
	}
	artwork, ok := s.publishedArtwork(pages[0].GetString("artwork"))
	if !ok {
		return ""
	}
	return artworkImageURL(artwork, profile)
}

func (s *Service) publishedArtwork(id string) (*core.Record, bool) {
	if id == "" {
		return nil, false
	}
	record, err := s.app.FindRecordById(constants.CollectionArtworks, id)
	if err != nil || !record.GetBool("published") || record.GetString("image") == "" {
		return nil, false
	}
	return record, true
}

func artworkImageURL(record *core.Record, profile wgaurl.DeliveryProfile) string {
	return wgaurl.GenerateDeliveryURL(constants.CollectionArtworks, record.Id, record.GetString("image"), record.GetInt("image_width"), profile, "")
}

func (s *Service) sectionNames(tourID string, revisionID string) (map[string]string, error) {
	records, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourSections,
		"tour = {:tour} && revision = {:revision}", "+section_order,+id", 0, 0,
		map[string]any{"tour": tourID, "revision": revisionID})
	if err != nil {
		return nil, err
	}
	sections := map[string]string{}
	for _, record := range records {
		sections[record.Id] = record.GetString("title")
	}
	return sections, nil
}

func (s *Service) blocks(pageID string) ([]dto.TourBlock, error) {
	records, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourBlocks,
		"page = {:page}", "+block_order,+id", 0, 0, map[string]any{"page": pageID})
	if err != nil {
		return nil, err
	}
	blocks := make([]dto.TourBlock, 0, len(records))
	for _, record := range records {
		blocks = append(blocks, dto.TourBlock{Kind: record.GetString("block_kind"), HTML: s.sanitizer.Sanitize(record.GetString("content_html"))})
	}
	return blocks, nil
}

func (s *Service) indexRows(pageID string) ([]dto.TourIndexEntry, error) {
	records, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourIndexRows,
		"page = {:page}", "+row_order,+id", 0, 0, map[string]any{"page": pageID})
	if err != nil {
		return nil, err
	}
	rows := make([]dto.TourIndexEntry, 0, len(records))
	for _, record := range records {
		rows = append(rows, dto.TourIndexEntry{Name: record.GetString("name"), Dates: record.GetString("dates"),
			Note: record.GetString("note"), TargetPath: safeInternalPath(record.GetString("target_path"))})
	}
	return rows, nil
}

func (s *Service) sources(tourID string, revisionID string) ([]dto.TourSource, error) {
	records, err := s.app.FindRecordsByFilter(constants.CollectionGuidedTourBibliography,
		"tour = {:tour} && revision = {:revision}", "+item_order,+id", 0, 0,
		map[string]any{"tour": tourID, "revision": revisionID})
	if err != nil {
		return nil, err
	}
	sources := make([]dto.TourSource, 0, len(records))
	for _, record := range records {
		sources = append(sources, dto.TourSource{Citation: record.GetString("citation")})
	}
	return sources, nil
}

func buildFilters(active string) []dto.TourFilter {
	keys := []string{"", "survey", "artist", "site", "theme"}
	filters := make([]dto.TourFilter, 0, len(keys))
	for _, key := range keys {
		href := "/tours"
		if key != "" {
			href += "?kind=" + key
		}
		filters = append(filters, dto.TourFilter{Label: filterKinds[key], Href: href, Active: key == active})
	}
	return filters
}

func buildContents(slug string, title string, pages []*core.Record, sections map[string]string, hasSources bool, current int) []dto.TourContentsItem {
	items := []dto.TourContentsItem{{Number: 1, Title: title, Href: tourURL(slug, 1), Current: current == 1}}
	for index, page := range pages {
		number := index + 2
		items = append(items, dto.TourContentsItem{Number: number, Title: page.GetString("title"),
			Section: sections[page.GetString("section")], SectionID: page.GetString("section"),
			Href: tourURL(slug, number), Current: current == number})
	}
	if hasSources {
		number := len(items) + 1
		items = append(items, dto.TourContentsItem{Number: number, Title: "Sources", Href: tourURL(slug, number), Current: current == number})
	}
	return items
}

func contentsTitle(items []dto.TourContentsItem, address int) string {
	for _, item := range items {
		if item.Number == address {
			return item.Title
		}
	}
	return ""
}

func tourURL(slug string, page int) string {
	if page == 1 {
		return "/tours/" + url.PathEscape(slug)
	}
	return "/tours/" + url.PathEscape(slug) + "/" + strconv.Itoa(page)
}

func revisionLabel(record *core.Record) string {
	if label := record.GetString("label"); label != "" {
		return label
	}
	return "Revision " + strconv.Itoa(record.GetInt("revision_number"))
}

func safeLegacyURL(raw string) string {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return parsed.String()
}

func safeInternalPath(raw string) string {
	if raw == "" {
		return ""
	}
	// Browsers resolve backslashes as path separators for http(s) URLs, so any
	// backslash can turn a safe-looking path (e.g. `/\evil.example`) into a
	// protocol-relative cross-origin destination. Reject them outright.
	if strings.ContainsRune(raw, '\\') {
		return ""
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return ""
	}
	return raw
}

// LegacyRoute resolves a validated source path to its canonical tour address.
// It never echoes the requested path into the destination: the canonical URL is
// derived only from the resolved tour slug and page position, so there is no
// open-redirect surface.
func (s *Service) LegacyRoute(rawPath string) (string, error) {
	path := safeInternalPath(rawPath)
	if path == "" {
		return "", ErrNotFound
	}
	record, err := s.app.FindFirstRecordByFilter(constants.CollectionGuidedTourLegacyRoutes,
		"legacy_path = {:path}", map[string]any{"path": path})
	if err != nil {
		return "", ErrNotFound
	}
	page, err := s.app.FindRecordById(constants.CollectionGuidedTourPages, record.GetString("tour_page"))
	if err != nil {
		return "", ErrNotFound
	}
	tour, err := s.app.FindRecordById(constants.CollectionGuidedTours, page.GetString("tour"))
	if err != nil || tour.GetString("publication_status") != "published" || tour.GetString("published_revision") == "" {
		return "", ErrNotFound
	}
	address := page.GetInt("page_position") + 1
	return tourURL(tour.GetString("slug"), address), nil
}
