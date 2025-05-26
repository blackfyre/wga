package dto

type ComboboxOption struct {
	Value string
	Label string
}

type ComboboxProps struct {
	ID          string
	Name        string
	Placeholder string
	Options     []ComboboxOption
	Value       string
	Required    bool
	Disabled    bool
	Size        string // xs, sm, md, lg, xl
	Color       string // primary, secondary, accent, etc.
	HxGet       string
	HxTarget    string
	HxTrigger   string
	Class       string
}
