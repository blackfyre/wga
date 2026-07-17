package utils

import "testing"

func TestNormalizedBioExcerpt(t *testing.T) {
	tests := []struct {
		name string
		data BioExcerptDTO
		want string
	}{
		{
			name: "exact years and known places",
			data: BioExcerptDTO{
				YearOfBirth:       1901,
				ExactYearOfBirth:  "yes",
				PlaceOfBirth:      "Accra",
				KnownPlaceOfBirth: "yes",
				YearOfDeath:       1980,
				ExactYearOfDeath:  "yes",
				PlaceOfDeath:      "Lagos",
				KnownPlaceOfDeath: "yes",
			},
			want: "b. 1901 Accra, d. 1980 Lagos",
		},
		{
			name: "approximate years and uncertain places",
			data: BioExcerptDTO{
				YearOfBirth:       1901,
				ExactYearOfBirth:  "no",
				PlaceOfBirth:      "Accra",
				KnownPlaceOfBirth: "no",
				YearOfDeath:       1980,
				ExactYearOfDeath:  "no",
				PlaceOfDeath:      "Lagos",
				KnownPlaceOfDeath: "no",
			},
			want: "b. ~1901 Accra?, d. ~1980 Lagos?",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizedBioExcerpt(test.data); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
