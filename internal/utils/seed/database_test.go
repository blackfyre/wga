package seed

import "testing"

func TestCenturyForPeriod(t *testing.T) {
	tests := []struct {
		period string
		want   string
	}{
		{period: "12th century", want: "12"},
		{period: "Baroque", want: "18"},
	}

	for _, test := range tests {
		t.Run(test.period, func(t *testing.T) {
			got, err := centuryForPeriod(test.period)
			if err != nil {
				t.Fatalf("centuryForPeriod(%q): %v", test.period, err)
			}
			if got != test.want {
				t.Fatalf("centuryForPeriod(%q) = %q, want %q", test.period, got, test.want)
			}
		})
	}
}

func TestUniqueArtistSlug(t *testing.T) {
	used := map[string]struct{}{}
	if got, want := uniqueArtistSlug(used, "Marco d' Oggiono", "first"), "marco-d-oggiono"; got != want {
		t.Fatalf("first slug = %q, want %q", got, want)
	}
	if got, want := uniqueArtistSlug(used, "Marco d' Oggiono", "second"), "marco-d-oggiono-second"; got != want {
		t.Fatalf("second slug = %q, want %q", got, want)
	}
}
