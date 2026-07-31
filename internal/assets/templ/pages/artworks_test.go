package pages

import (
	"reflect"
	"testing"
)

func TestSortedArtworkFilterOptions(t *testing.T) {
	options := map[string]string{
		"":             "Any",
		"sculpture":    "Sculpture",
		"painting":     "Painting",
		"architecture": "Architecture",
	}

	got := sortedArtworkFilterOptions(options)
	want := []ArtworkFilterOption{
		{Value: "architecture", Label: "Architecture"},
		{Value: "painting", Label: "Painting"},
		{Value: "sculpture", Label: "Sculpture"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedArtworkFilterOptions() = %#v, want %#v", got, want)
	}
}
