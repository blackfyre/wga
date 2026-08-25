package dto

// ItineraryStop is one ordered stop projection shared by the builder and the
// public slideshow. URLs are final delivery URLs supplied by the handler.
type ItineraryStop struct {
	ID          string
	ArtworkID   string
	Title       string
	Artist      string
	Date        string
	School      string
	Narration   string
	Position    int
	URL         string
	ImageURL    string
	Unavailable bool
}

// ItineraryTrayView is the persistent tray projection rendered by the shared
// layout after the first addition.
type ItineraryTrayView struct {
	Count      int
	BuilderURL string
	Thumbs     []string
}

// ItineraryPickerWork is one addable published artwork in the builder picker.
type ItineraryPickerWork struct {
	ArtworkID string
	Title     string
	Artist    string
	Date      string
	ImageURL  string
	Added     bool
}

// ItineraryBuilderView drives the server-rendered builder page and its
// HTMX-replaced draft block. Selected is the index of the active stop in Stops;
// PickerOpen holds the add-picker disclosure; Duration is the pre-formatted
// reading-time estimate.
type ItineraryBuilderView struct {
	ID         string
	Title      string
	Intro      string
	Creator    string
	Listed     bool
	Stops      []ItineraryStop
	Selected   int
	Count      int
	Max        int
	CSRF       string
	Query      string
	PickerOpen bool
	Picker     []ItineraryPickerWork
	AtLimit    bool
	Duration   string
}

// ItinerarySummary is one entry on the public itinerary index.
type ItinerarySummary struct {
	Title     string
	Creator   string
	URL       string
	Note      string
	Count     int
	Duration  string
	Published string
}

// ItineraryIndexView drives the public listed-itinerary index.
type ItineraryIndexView struct {
	Itineraries []ItinerarySummary
	Total       int
}

// ItineraryView drives the one-stop-at-a-time public slideshow. Plate is the
// deliberate artwork plate; DisplayURL uses the record/tour profile while
// ZoomURL uses the viewer profile with the no-upscale fallback. StopURLs holds
// one direct stop link per stop, in running order, for the progress strip.
type ItineraryView struct {
	Title      string
	Creator    string
	Total      int
	Index      int
	StopTitle  string
	StopArtist string
	StopDate   string
	StopSchool string
	Narration  string
	Plate      Plate
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	StopURLs   []string
	ExitURL    string
}

// ItineraryPublishedView drives the session-owned publication confirmation.
type ItineraryPublishedView struct {
	Title     string
	URL       string
	TokenURL  string
	Creator   string
	Expires   string
	Published string
	Duration  string
	StopCount int
	Listed    bool
}
