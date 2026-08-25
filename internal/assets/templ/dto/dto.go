package dto

type Artist struct {
	Id              string
	Name            string
	Schools         string
	Profession      string
	Url             string
	BioExcerpt      string
	Jsonld          string
	Bio             string
	Portrait        string
	Works           ImageGrid
	HxTarget        string
	ShowBreadcrumbs bool
}

type Artwork struct {
	Id              string
	Title           string
	Comment         string
	Technique       string
	Year            string
	Dimensions      string
	ArtType         string
	Location        string
	CitationKey     string
	CitationTitle   string
	CitationURL     string
	RelatedWorks    ImageGrid
	Jsonld          string
	Url             string
	HxTarget        string
	ShowBreadcrumbs bool
	// ReproFile is the truthful reproduction file summary (dimensions and
	// format derived from the record's source dimensions and image filename).
	// It is empty when no reproduction dimensions are recorded, so the artwork
	// template never fabricates a file caption.
	ReproFile string
	// SourceURL is the original artwork file URL (no thumbnail query), used only
	// for the deliberate source-file download. Empty when the record has no
	// image filename.
	SourceURL string
	// ReproductionSourceURL is the approved canonical WGA record source URL. It
	// is empty when the producer value is absent or fails the allow-list policy.
	ReproductionSourceURL string
	// Palette is the compact image-derived colour palette (hex and quantised
	// weight). Empty when no palette profile is recorded.
	Palette []ColourSwatch
	// HasCommentary reports whether source-backed commentary exists. AI-generated
	// or metadata-only text is never surfaced as commentary.
	HasCommentary bool
	// SourceComment is the raw source commentary, or "" when absent.
	SourceComment string
	// Related carries the active related-work basis, connection heading, sparse
	// state, and alternative link.
	Related RelatedWorkState
	// Music is the deterministic period-music card for the artwork's creation
	// date, or an unavailable card when no published match exists.
	Music MusicPeriod
	Image
	Artist
}

// ColourSwatch is one compact image-derived palette entry: a hex colour and its
// quantised share weight.
type ColourSwatch struct {
	Hex    string
	Weight int
}

// RelatedWorkBasis is one of the four related-work basis controls.
type RelatedWorkBasis struct {
	Value  string // "artist", "collection", "palette", "period"
	Label  string // "BY ARTIST", "SAME COLLECTION", "SIMILAR PALETTE", "SAME PERIOD"
	URL    string // canonical URL; the default basis omits the query parameter
	Active bool
}

// RelatedWorkState carries the active related-work basis, its connection
// heading, and the sparse-result explanation and alternative link.
type RelatedWorkState struct {
	ActiveBasis    string
	Connection     string
	Sparse         bool
	SparseNote     string
	Alternative    string
	AlternativeURL string
	Bases          []RelatedWorkBasis
}

// MusicPeriod is the deterministic period-music card for one artwork. Available
// is false when no published matching song exists.
type MusicPeriod struct {
	Available bool
	SongID    string
	Piece     string
	Composer  string
	PlayerURL string
}

type ArtworkSearchDTO struct {
	ArtFormOptions     map[string]string
	ArtTypeOptions     map[string]string
	ArtSchoolOptions   map[string]string
	ActiveFilterValues *ArtworkSearchFilterValues
	ArtistNameList     map[string]string
	NewFilterValues    string
	ClearUrl           string
	DualModeContext    *ArtworkSearchDualModeDto
	Results            ArtworkSearchResultDTO
	HxTarget           string
}

type ArtworkSearchFilterValues struct {
	Query         string
	ArtFormString string
	ArtTypeString string
	SchoolString  string
	Title         string
	ArtistString  string
	YearFrom      string
	YearTo        string
	View          string
}

type ArtworkSearchResultDTO struct {
	ActiveFiltering bool
	Artworks        ImageGrid
	ResultCount     int
	ResultSummary   string
	Pagination      string
	HxTarget        string
	DualModeUrls    map[string]string
	DualModeTarget  string
	View            string
	GridUrl         string
	ListUrl         string
}

type ArtworkSearchDualModeDto struct {
	LeftPath      string
	RightPath     string
	LeftRenderTo  string
	RightRenderTo string
	Target        string
}

type DualViewDto struct {
	Left                      string
	Right                     string
	LeftLinksOpenInOtherPane  bool
	RightLinksOpenInOtherPane bool
	ArtworkSearchLeftUrl      string
	ArtworkSearchRightUrl     string
	CopyLeftToRightUrl        string
	CopyRightToLeftUrl        string
	ReverseUrl                string
	ClearLeftUrl              string
	ClearRightUrl             string
}

type DualLookupResultDto struct {
	Url     string
	Label   string
	Context string
}

type DualLookupResultsDto struct {
	Kind          string
	Query         string
	QueryTooShort bool
	Results       []DualLookupResultDto
}

type DualPaneLoadFormDto struct {
	Path          string
	OtherPath     string
	LeftRenderTo  string
	RightRenderTo string
}

type DualPaneLoadFormsDto struct {
	Left  DualPaneLoadFormDto
	Right DualPaneLoadFormDto
}

type DualPaneTargetUrlsDto struct {
	LeftSamePaneUrl   string
	LeftOtherPaneUrl  string
	RightSamePaneUrl  string
	RightOtherPaneUrl string
}
