package utils

import (
	"sort"

	"github.com/blackfyre/wga/assets/templ/dto"
)

// ConvertArtistMapToComboboxOptions converts a map[string]string of artist URLs and names
// to a sorted slice of ComboboxOption structs for use with the combobox component.
// The map keys are expected to be URLs and values are artist names.
func ConvertArtistMapToComboboxOptions(artistMap map[string]string) []dto.ComboboxOption {
	options := make([]dto.ComboboxOption, 0, len(artistMap))

	for url, name := range artistMap {
		options = append(options, dto.ComboboxOption{
			Value: url,  // The artist URL for programmatic use
			Label: name, // The artist name for display
		})
	}

	// Sort by label (artist name) for better UX
	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return options
}

// ConvertStringSliceToComboboxOptions converts a slice of strings to ComboboxOptions
// where both value and label are the same string (useful for simple dropdowns)
func ConvertStringSliceToComboboxOptions(items []string) []dto.ComboboxOption {
	options := make([]dto.ComboboxOption, len(items))

	for i, item := range items {
		options[i] = dto.ComboboxOption{
			Value: item,
			Label: item,
		}
	}

	return options
}

// ConvertMapToComboboxOptions converts any map[string]string to ComboboxOptions
// This is a generic helper for any key-value mapping
func ConvertMapToComboboxOptions(dataMap map[string]string) []dto.ComboboxOption {
	options := make([]dto.ComboboxOption, 0, len(dataMap))

	for key, value := range dataMap {
		options = append(options, dto.ComboboxOption{
			Value: key,
			Label: value,
		})
	}

	// Sort by label for consistency
	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return options
}
