package dto

// Work is the card and row projection used by public artwork lists.
type Work struct {
	URL       string
	ImageURL  string
	ArtworkID string
	Title     string
	Artist    string
	Metadata  string
}

// MetaEntry is one labelled value in a public record's metadata list.
type MetaEntry struct {
	Label string
	Value string
}

// Plate describes a deliberately presented reproduction. URLs are final
// delivery URLs supplied by the owning workflow.
type Plate struct {
	DisplayURL  string
	ZoomURL     string
	Alt         string
	Label       string
	Placeholder string
	Aspect      string
	Contain     bool
}

// Field describes one ordinary labelled form control.
type Field struct {
	ID          string
	Name        string
	Label       string
	Type        string
	Value       string
	Placeholder string
	Hint        string
	Error       string
	Required    bool
	MaxLength   int
	Rows        int
}

// ChipOption is one value in a no-JavaScript filter choice.
type ChipOption struct {
	Label   string
	Value   string
	Checked bool
}

// ChipGroup is a labelled group of mutually exclusive filter choices.
type ChipGroup struct {
	Legend  string
	Name    string
	Note    string
	Inline  bool
	Options []ChipOption
}

// NavChip is a navigational filter link. HxTarget is an optional enhancement.
type NavChip struct {
	Label    string
	Href     string
	Active   bool
	HxTarget string
}

// RangeField describes a dual-handle server-rendered range control.
type RangeField struct {
	Label     string
	FromID    string
	FromName  string
	FromValue int
	ToID      string
	ToName    string
	ToValue   int
	Min       int
	Max       int
	Step      int
	Brush     bool
}

// EmptyState describes a visible absence with an optional recovery link.
type EmptyState struct {
	Title         string
	Message       string
	RecoveryLabel string
	RecoveryHref  string
}
