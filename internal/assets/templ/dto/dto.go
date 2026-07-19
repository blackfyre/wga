package dto

type Artist struct {
	Id              string
	Name            string
	BornDied        string
	Schools         string
	Profession      string
	Url             string
	BioExcerpt      string
	Jsonld          string
	Bio             string
	Works           ImageGrid
	HxTarget        string
	ShowBreadcrumbs bool
}

type ArtistsView struct {
	Count      string
	Artists    []Artist
	Pagination string
	Jsonld     string
	QueryStr   string
	HxTarget   string
}

type Artwork struct {
	Id              string
	Title           string
	Comment         string
	Technique       string
	Jsonld          string
	Url             string
	HxTarget        string
	ShowBreadcrumbs bool
	Image
	Artist
}

type ArtworkSearchDTO struct {
	ArtFormOptions     map[string]string
	ArtTypeOptions     map[string]string
	ArtSchoolOptions   map[string]string
	ActiveFilterValues *ArtworkSearchFilterValues
	ArtistNameList     map[string]string
	NewFilterValues    string
	Results            ArtworkSearchResultDTO
	HxTarget           string
}

type ArtworkSearchFilterValues struct {
	ArtFormString string
	ArtTypeString string
	SchoolString  string
	Title         string
	ArtistString  string
}

type ArtworkSearchResultDTO struct {
	ActiveFiltering bool
	Artworks        ImageGrid
	ResultCount     int
	ResultSummary   string
	Pagination      string
	HxTarget        string
}

type ArtistNameListEntry struct {
	Url   string `json:"url"`
	Label string `json:"label"`
}

type DualViewDto struct {
	Left                      string
	Right                     string
	LeftLinksOpenInOtherPane  bool
	RightLinksOpenInOtherPane bool
	ArtistNameList            []ArtistNameListEntry
	CopyLeftToRightUrl        string
	CopyRightToLeftUrl        string
	ReverseUrl                string
	ClearLeftUrl              string
	ClearRightUrl             string
}
