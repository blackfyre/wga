package dto

type Image struct {
	Thumb     string
	Image     string
	Title     string
	Technique string
	Comment   string
	Url       string
	Id        string
	Jsonld    interface{}
	HxTarget  string
	Artist
}

type ImageGrid []Image
