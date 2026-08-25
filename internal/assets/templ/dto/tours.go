package dto

// TourFilter is an ordinary-link editorial filter.
type TourFilter struct {
	Label  string
	Href   string
	Active bool
}

// TourCard is one published tour on the index.
type TourCard struct {
	Slug               string
	Title              string
	Blurb              string
	Kind               string
	Editor             string
	Revision           string
	Number             string
	Year               int
	ImageURL           string
	PresentationStatus string
	// Pages counts the whole walk including the synthesised title and sources
	// pages. 0 = not yet rebuilt page by page.
	Pages int
	// Scope is the pre-formatted editorial scope (for example "1,910 works").
	// It is empty until an approved scope is curated, in which case the index
	// shows an honest unavailable state rather than a fabricated figure.
	Scope string
}

// TourIndex is the Guided Tours index projection. Rebuilt and Original mirror
// the reference's two editorial groups: tours rebuilt page by page versus the
// rest of the series still in its original layout.
type TourIndex struct {
	Filter   string
	Filters  []TourFilter
	Rebuilt  []TourCard
	Original []TourCard
}

// TourContentsItem is one stable address in a tour.
type TourContentsItem struct {
	Number    int
	Title     string
	Section   string
	SectionID string
	Href      string
	Current   bool
}

// TourBlock is sanitised editorial HTML.
type TourBlock struct {
	Kind string
	HTML string
}

// TourIndexEntry is one entry on an editorial index page.
type TourIndexEntry struct {
	Name       string
	Dates      string
	Note       string
	TargetPath string
}

// TourSource is one approved bibliography entry.
type TourSource struct {
	Citation string
}

// TourPage is one addressed title, text, picture, list, or sources page.
type TourPage struct {
	Slug               string
	TourTitle          string
	Blurb              string
	Kind               string
	Editor             string
	Revision           string
	Number             string
	PublishedYear      int
	RevisedYear        int
	PresentationStatus string
	LegacyURL          string

	Address            int
	TotalPages         int
	PageType           string
	PageTitle          string
	Dateline           string
	Section            string
	Contents           []TourContentsItem
	PreviousURL        string
	PreviousLabel      string
	NextURL            string
	NextLabel          string
	RevisionSourceHash string
	SourcePath         string
	SourceHash         string

	Blocks        []TourBlock
	IndexRows     []TourIndexEntry
	Sources       []TourSource
	DisplayURL    string
	ZoomURL       string
	ArtworkURL    string
	ArtworkAlt    string
	ArtworkCredit string
}
