package handlers

import (
	"testing"
)

func TestGenerateBioSection(t *testing.T) {
	// Test case 1: All parameters are provided
	result := generateBioSection("b.", 1990, "yes", "New York", "yes")
	expected := "b. 1990 New York"
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}

	// Test case 2: Exact year is not applicable
	result = generateBioSection("b.", 1990, "no", "New York", "yes")
	expected = "b. ~1990 New York"
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}

	// Test case 3: Place of birth is not known
	result = generateBioSection("b.", 1990, "yes", "Unknown", "no")
	expected = "b. 1990 Unknown?"
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}
}
func TestNormalizedBioExcerpt(t *testing.T) {
	d := BioExcerptDTO{
		YearOfBirth:       1990,
		ExactYearOfBirth:  "yes",
		PlaceOfBirth:      "New York",
		KnownPlaceOfBirth: "yes",
		YearOfDeath:       2020,
		ExactYearOfDeath:  "yes",
		PlaceOfDeath:      "Los Angeles",
		KnownPlaceOfDeath: "yes",
	}

	result := normalizedBioExcerpt(d)
	expected := "b. 1990 New York, d. 2020 Los Angeles"
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}
}
