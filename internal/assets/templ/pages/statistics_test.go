package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStatisticsBlockRendersAccessibleSummaries(t *testing.T) {
	content := StatisticsPageDTO{
		ArtistCount:        "1",
		ArtworkCount:       "2",
		ArtFormData:        `[{"name":"Painting","count":2}]`,
		ArtworksPeriodData: `[{"period_start":1500,"school":"Italian","count":2}]`,
		ArtistsPeriodData:  `[{"period_start":1500,"school":"Italian","count":1}]`,
		ArtFormSummary: []StatisticsArtFormRow{
			{Name: "Painting", Count: 2},
		},
		ArtworksPeriodSummary: []StatisticsSchoolPeriodRow{
			{Period: "1500–1549", School: "Italian", Count: 2},
		},
		ArtistsPeriodSummary: []StatisticsSchoolPeriodRow{
			{Period: "1500–1549", School: "Italian", Count: 1},
		},
	}

	var output bytes.Buffer
	if err := StatisticsBlock(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render statistics block: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`aria-describedby="art-form-summary"`,
		`aria-describedby="artworks-period-summary"`,
		`aria-describedby="artists-period-summary"`,
		`id="art-form-summary"`,
		`id="artworks-period-summary"`,
		`id="artists-period-summary"`,
		"Painting",
		"1500–1549",
		"Italian",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected rendered statistics to contain %q\ngot: %s", expected, rendered)
		}
	}
}
