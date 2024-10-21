package dual

import (
	"reflect"
	"testing"

	"github.com/blackfyre/wga/assets/templ/dto"
)

func TestFormatArtistNameList(t *testing.T) {
	// Test case 1: Empty artist name list
	artistNameList := map[string]string{}
	expected := []dto.ArtistNameListEntry{}
	result := formatArtistNameList(artistNameList)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 2: Single entry in artist name list
	artistNameList = map[string]string{
		"/artist/1": "Artist One",
	}
	expected = []dto.ArtistNameListEntry{
		{Url: "/artist/1", Label: "Artist One"},
	}
	result = formatArtistNameList(artistNameList)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 3: Multiple entries in artist name list
	artistNameList = map[string]string{
		"/artist/1": "Artist One",
		"/artist/2": "Artist Two",
	}
	expected = []dto.ArtistNameListEntry{
		{Url: "/artist/1", Label: "Artist One"},
		{Url: "/artist/2", Label: "Artist Two"},
	}
	result = formatArtistNameList(artistNameList)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
func TestReverseSide(t *testing.T) {
	// Test case 1: Input is "left"
	input := "left"
	expected := "right"
	result := reverseSide(input)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 2: Input is "right"
	input = "right"
	expected = "left"
	result = reverseSide(input)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	// Test case 3: Input is neither "left" nor "right"
	input = "center"
	expected = ""
	result = reverseSide(input)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
