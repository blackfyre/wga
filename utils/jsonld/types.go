package jsonld

// Person represents a person entity in JSON-LD format.
type Person struct {
	Context       string     `json:"@context,omitempty"`
	Type          string     `json:"@type,omitempty"`
	Name          string     `json:"name,omitempty"`
	Url           string     `json:"url,omitempty"`
	BirthDate     string     `json:"birthDate,omitempty"`
	DeathDate     string     `json:"deathDate,omitempty"`
	PlaceOfBirth  Place      `json:"birthPlace,omitempty"`
	PlaceOfDeath  Place      `json:"deathPlace,omitempty"`
	Description   string     `json:"description,omitempty"`
	HasOccupation Occupation `json:"hasOccupation,omitempty"`
}

// Place represents a place entity in JSON-LD format.
type Place struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"@type,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Occupation represents an occupation entity in JSON-LD format.
type Occupation struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"@type,omitempty"`
	Name    string `json:"name,omitempty"`
}

// VisualArtwork represents a visual artwork entity in JSON-LD format.
type VisualArtwork struct {
	Context     string      `json:"@context,omitempty"`
	Type        string      `json:"@type,omitempty"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Url         string      `json:"url,omitempty"`
	Artform     string      `json:"artform,omitempty"`
	Artist      Person      `json:"artist,omitempty"`
	ArtMedium   string      `json:"artMedium,omitempty"`
	Image       ImageObject `json:"image,omitempty"`
}

// ImageObject represents an image object entity in JSON-LD format.
type ImageObject struct {
	Context      string `json:"@context,omitempty"`
	Type         string `json:"@type,omitempty"`
	Name         string `json:"name,omitempty"`
	Caption      string `json:"caption,omitempty"`
	Image        string `json:"image,omitempty"`
	ThumbnailUrl string `json:"thumbnailUrl,omitempty"`
}

// newPerson returns a new Person object with the given Person object.
func newPerson(p Person) Person {
	p.Context = "https://schema.org"
	p.Type = "Person"
	return p
}

// newPlace returns a new Place object with the given Place object.
func newPlace(p Place) Place {
	p.Context = "https://schema.org"
	p.Type = "Place"
	return p
}

// newOccupation returns a new Occupation object with the given Occupation object.
func newOccupation(o Occupation) Occupation {
	o.Context = "https://schema.org"
	o.Type = "Occupation"
	return o
}

// newVisualArtwork returns a new VisualArtwork object with the given VisualArtwork object.
func newVisualArtwork(v VisualArtwork) VisualArtwork {
	v.Context = "https://schema.org"
	v.Type = "VisualArtwork"
	return v
}
