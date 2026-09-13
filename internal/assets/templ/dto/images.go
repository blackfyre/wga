package dto

type Image struct {
	Thumb     string
	Image     string
	Zoom      string
	Title     string
	Technique string
	Date      string
	School    string
	Form      string
	Type      string
	Metadata  string
	Comment   string
	Url       string
	Id        string
	Jsonld    interface{}
	HxTarget  string
	Artist
}

type ImageGrid []Image
