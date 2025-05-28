package utils

import (
	"testing"
)

func TestConvertArtistMapToComboboxOptions(t *testing.T) {
	// Test data similar to what GetArtistNameList returns
	artistMap := map[string]string{
		"/artists/leonardo-da-vinci-123": "Leonardo da Vinci",
		"/artists/michelangelo-456":      "Michelangelo",
		"/artists/raphael-789":           "Raphael",
		"/artists/albrecht-durer-101":    "Albrecht Dürer",
	}

	options := ConvertArtistMapToComboboxOptions(artistMap)

	// Test that we get the right number of options
	if len(options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(options))
	}

	// Test that options are sorted by label
	expectedOrder := []string{"Albrecht Dürer", "Leonardo da Vinci", "Michelangelo", "Raphael"}
	for i, expected := range expectedOrder {
		if options[i].Label != expected {
			t.Errorf("Expected label %s at position %d, got %s", expected, i, options[i].Label)
		}
	}

	// Test that values are preserved correctly
	for _, option := range options {
		if originalName, exists := artistMap[option.Value]; !exists || originalName != option.Label {
			t.Errorf("Value %s should map to label %s", option.Value, option.Label)
		}
	}
}

func TestConvertStringSliceToComboboxOptions(t *testing.T) {
	items := []string{"Option A", "Option B", "Option C"}
	options := ConvertStringSliceToComboboxOptions(items)

	if len(options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(options))
	}

	for i, item := range items {
		if options[i].Value != item || options[i].Label != item {
			t.Errorf("Expected both value and label to be %s, got value=%s, label=%s",
				item, options[i].Value, options[i].Label)
		}
	}
}

func TestConvertMapToComboboxOptions(t *testing.T) {
	dataMap := map[string]string{
		"key1": "Label 1",
		"key2": "Label 2",
		"key3": "Label 3",
	}

	options := ConvertMapToComboboxOptions(dataMap)

	if len(options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(options))
	}

	// Test that all keys and values are preserved
	for _, option := range options {
		if originalLabel, exists := dataMap[option.Value]; !exists || originalLabel != option.Label {
			t.Errorf("Value %s should map to label %s", option.Value, option.Label)
		}
	}

	// Test that options are sorted by label
	for i := 1; i < len(options); i++ {
		if options[i-1].Label > options[i].Label {
			t.Errorf("Options not sorted by label: %s comes before %s",
				options[i-1].Label, options[i].Label)
		}
	}
}

func TestEmptyInputs(t *testing.T) {
	// Test empty map
	emptyMap := map[string]string{}
	options := ConvertMapToComboboxOptions(emptyMap)
	if len(options) != 0 {
		t.Errorf("Expected 0 options for empty map, got %d", len(options))
	}

	// Test empty slice
	emptySlice := []string{}
	options2 := ConvertStringSliceToComboboxOptions(emptySlice)
	if len(options2) != 0 {
		t.Errorf("Expected 0 options for empty slice, got %d", len(options2))
	}
}
