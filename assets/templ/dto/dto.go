package dto

type Image struct {
	Thumb     string
	Image     string
	Title     string
	Technique string
	Comment   string
	Url       string
	Id        string
	Jsonld interface{}
	Artist
}

type ImageGrid []Image

type Artist struct {
	Id         string
	Name       string
	BornDied   string
	Schools    string
	Profession string
	Url        string
	BioExcerpt string
	Jsonld     string
	Bio        string
	Works      ImageGrid
}

type ArtistsView struct {
	Count      string
	Artists    []Artist
	Pagination string
	Jsonld     string
	QueryStr   string
}

type Artwork struct {
	Id        string
	Title     string
	Comment   string
	Technique string
	Jsonld    string
	Url       string
	Image
	Artist
}

type ArtworkSearchDTO struct {
	ArtFormOptions     map[string]string
	ArtTypeOptions     map[string]string
	ArtSchoolOptions   map[string]string
	ActiveFilterValues *ArtworkSearchFilterValues
	ArtistNameList     []string
	NewFilterValues    string
	Results            ArtworkSearchResultDTO
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
	Pagination      string
}