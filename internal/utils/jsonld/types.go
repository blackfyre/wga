package jsonld

// Context is the schema.org vocabulary every top-level JSON-LD document uses.
// Nested nodes inherit it and carry no @context of their own.
const Context = "https://schema.org"

// PrimarySuffix identifies a page's main entity: {canonical URL}#primary.
const PrimarySuffix = "#primary"

// Document is one JSON-LD script body holding a page's linked nodes.
type Document struct {
	Context string `json:"@context"`
	Graph   []any  `json:"@graph"`
}

// Person represents a schema.org Person. Pointer fields are omitted when nil,
// so a missing value never serialises as an empty object.
type Person struct {
	Type          string      `json:"@type"`
	ID            string      `json:"@id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Url           string      `json:"url,omitempty"`
	BirthDate     string      `json:"birthDate,omitempty"`
	DeathDate     string      `json:"deathDate,omitempty"`
	BirthPlace    *Place      `json:"birthPlace,omitempty"`
	DeathPlace    *Place      `json:"deathPlace,omitempty"`
	Description   string      `json:"description,omitempty"`
	Image         string      `json:"image,omitempty"`
	HasOccupation *Occupation `json:"hasOccupation,omitempty"`
}

// Place represents a schema.org Place.
type Place struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

// Occupation represents a schema.org Occupation.
type Occupation struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

// VisualArtwork represents a schema.org VisualArtwork.
type VisualArtwork struct {
	Type        string  `json:"@type"`
	ID          string  `json:"@id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Url         string  `json:"url,omitempty"`
	Description string  `json:"description,omitempty"`
	Image       string  `json:"image,omitempty"`
	Artform     string  `json:"artform,omitempty"`
	ArtMedium   string  `json:"artMedium,omitempty"`
	DateCreated string  `json:"dateCreated,omitempty"`
	Creator     *Person `json:"creator,omitempty"`
}

// BreadcrumbList represents a schema.org BreadcrumbList.
type BreadcrumbList struct {
	Type            string     `json:"@type"`
	ID              string     `json:"@id,omitempty"`
	ItemListElement []ListItem `json:"itemListElement"`
}

// ListItem is one position in a BreadcrumbList.
type ListItem struct {
	Type     string `json:"@type"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	Item     string `json:"item,omitempty"`
}

// ItemList represents a schema.org ItemList, such as a curated selection.
type ItemList struct {
	Type            string     `json:"@type"`
	ID              string     `json:"@id,omitempty"`
	Name            string     `json:"name,omitempty"`
	Url             string     `json:"url,omitempty"`
	NumberOfItems   int        `json:"numberOfItems"`
	ItemListElement []ListItem `json:"itemListElement"`
}
