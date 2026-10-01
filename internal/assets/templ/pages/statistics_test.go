package pages

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestStatisticsBlockRendersAccessibleSummaries(t *testing.T) {
	content := StatisticsPageDTO{
		ArtistCount:  "1",
		ArtworkCount: "2",
		ArtFormData:  `[{"name":"Painting","count":6},{"name":"Sculpture","count":2}]`,
		ArtworksPeriodData: `[{"period_start":1500,"school":"Italian","count":3},` +
			`{"period_start":1500,"school":"French","count":1},` +
			`{"period_start":1550,"school":"Dutch","count":2}]`,
		ArtistsPeriodData: `[{"period_start":1500,"school":"Italian","count":1}]`,
		ArtFormSummary: []StatisticsArtFormRow{
			{Name: "Painting", Count: 6},
			{Name: "Sculpture", Count: 2},
		},
		ArtworksPeriodSummary: []StatisticsSchoolPeriodRow{
			{Period: "1500–1549", School: "Italian", Count: 3},
			{Period: "1500–1549", School: "French", Count: 1},
			{Period: "1550–1599", School: "Dutch", Count: 2},
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
		`class="caption-top`,
		"Art form distribution — data",
		"STACKED BY SCHOOL · BIRTH PERIOD OF ARTIST",
		`data-series-token="--wga-series-0"`,
		`data-school="Other"`,
		`data-series-token="--wga-faint" data-series-fill="hatch"`,
		`title="Italian"`,
		`title="Other"`,
		"background:var(--wga-series-0)",
		"Painting",
		"1500–1549",
		"IT",
		"TOTAL",
		"SHARE",
		"wga-enter",
		"text-(length:--t-32)",
		"md:text-(length:--t-44)",
		"RECOMPUTED HOURLY FROM PUBLISHED RECORDS.",
		"ARTISTS WITHOUT A RECORDED BIRTH YEAR ARE EXCLUDED FROM THE PERIOD CHARTS.",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected rendered statistics to contain %q\ngot: %s", expected, rendered)
		}
	}

	if strings.Contains(rendered, `class="sr-only"`) {
		t.Errorf("expected no sr-only caption on the statistics tables\ngot: %s", rendered)
	}
}

func TestStatisticsBlockRendersNoJavascriptVisualSummaries(t *testing.T) {
	content := StatisticsPageDTO{
		ArtFormSummary: []StatisticsArtFormRow{
			{Name: "Painting", Count: 6},
			{Name: "Sculpture", Count: 2},
		},
		ArtworksPeriodSummary: []StatisticsSchoolPeriodRow{
			{Period: "1500–1549", School: "Italian", Count: 3},
			{Period: "1500–1549", School: "French", Count: 1},
			{Period: "1500–1549", School: "Other", Count: 2},
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
		// The art-form donut has a server-rendered bar summary with proportional
		// widths derived from the same figures as the table.
		"width:75.000%",
		"width:25.000%",
		// Each stacked-bar chart carries a server-rendered CSS column summary.
		"height:50.000%",
		// "Other" wears a hatch, never a school tone.
		"repeating-linear-gradient(135deg, var(--wga-faint) 0 2px, transparent 2px 5px)",
		// The shared legend names every school.
		"Dutch",
		"Spanish",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected no-JS statistics summary to contain %q\ngot: %s", expected, rendered)
		}
	}

	// Three no-JS visual summaries: one art-form bar list and two period charts.
	if got := strings.Count(rendered, "<noscript>"); got != 3 {
		t.Errorf("expected three noscript summaries, got %d\ngot: %s", got, rendered)
	}
}

func TestStatisticsPeriodBarsRenderSingleChronologicalAxis(t *testing.T) {
	// Thirteen fifty-year periods exercise the release data shape: the columns
	// must stay on one chronological axis without wrapping.
	periods := []int{1300, 1350, 1400, 1450, 1500, 1550, 1600, 1650, 1700, 1750, 1800, 1850, 1900}
	rows := make([]StatisticsSchoolPeriodRow, 0, len(periods))
	for _, start := range periods {
		rows = append(rows, StatisticsSchoolPeriodRow{
			Period: fiftyYearLabel(start),
			School: "Italian",
			Count:  1,
		})
	}

	var output bytes.Buffer
	if err := StatisticsPeriodBars(rows).Render(context.Background(), &output); err != nil {
		t.Fatalf("render period bars: %v", err)
	}

	rendered := output.String()

	// The columns live in a single horizontally scrollable track, not a wrapping
	// fixed grid.
	if !strings.Contains(rendered, "overflow-x-auto") {
		t.Errorf("expected a scroll container\ngot: %s", rendered)
	}
	if !strings.Contains(rendered, "min-w-max") {
		t.Errorf("expected a single-row track that fits every period\ngot: %s", rendered)
	}
	if strings.Contains(rendered, "grid-cols") {
		t.Errorf("expected no fixed grid-cols layout\ngot: %s", rendered)
	}

	// Every period renders exactly once, in chronological order.
	last := -1
	for _, start := range periods {
		label := fiftyYearLabel(start)
		idx := strings.Index(rendered, label)
		if idx == -1 {
			t.Errorf("missing period label %q\ngot: %s", label, rendered)
			continue
		}
		if idx <= last {
			t.Errorf("period %q is out of chronological order (index %d <= %d)", label, idx, last)
		}
		last = idx
	}

	// Each period contributes one column.
	if got := strings.Count(rendered, `class="group flex h-full w-16 shrink-0 flex-col`); got != len(periods) {
		t.Errorf("expected %d bar columns, got %d\ngot: %s", len(periods), got, rendered)
	}
}

func TestStatisticsBlockRendersEmptyStates(t *testing.T) {
	content := StatisticsPageDTO{
		ArtFormSummary:        nil,
		ArtworksPeriodSummary: nil,
		ArtistsPeriodSummary:  nil,
	}

	var output bytes.Buffer
	if err := StatisticsBlock(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render statistics block: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		"No published artworks are grouped by art form.",
		"No published artworks are grouped by school and birth period.",
		"No published artists are grouped by school and birth period.",
		`role="status"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected empty statistics to contain %q\ngot: %s", expected, rendered)
		}
	}

	// No chart-equivalent visuals, legends, or tables should render when empty.
	for _, unexpected := range []string{
		"<noscript>",
		"<table",
		"TOTAL</p>",
		"title=\"Italian\"",
	} {
		if strings.Contains(rendered, unexpected) {
			t.Errorf("expected empty statistics to omit %q\ngot: %s", unexpected, rendered)
		}
	}

	// The empty messages must be announced, not merely visible.
	if got := strings.Count(rendered, `role="status"`); got != 3 {
		t.Errorf("expected three announced empty states, got %d\ngot: %s", got, rendered)
	}
}

func TestArtFormShareDerivesFromTotal(t *testing.T) {
	rows := []StatisticsArtFormRow{
		{Name: "Painting", Count: 6},
		{Name: "Sculpture", Count: 2},
	}

	if got := artFormTotal(rows); got != 8 {
		t.Errorf("art form total = %d, want 8", got)
	}
	if got := artFormSharePct(rows, 6); got != 75.0 {
		t.Errorf("share pct for 6 of 8 = %.3f, want 75.0", got)
	}
	if got := artFormShare(rows, 6); got != "75.0%" {
		t.Errorf("share for 6 of 8 = %q, want 75.0%%", got)
	}
	if got := artFormShare(rows, 2); got != "25.0%" {
		t.Errorf("share for 2 of 8 = %q, want 25.0%%", got)
	}
	if got := artFormShare(nil, 0); got != "—" {
		t.Errorf("share for empty rows = %q, want dash", got)
	}
	if got := artFormSharePct(nil, 0); got != 0 {
		t.Errorf("share pct for empty rows = %.3f, want 0", got)
	}
}

func TestBarShareScalesAgainstLargestPeriodTotal(t *testing.T) {
	rows := []StatisticsSchoolPeriodRow{
		{Period: "1500–1549", School: "Italian", Count: 3},
		{Period: "1500–1549", School: "French", Count: 1},
		{Period: "1550–1599", School: "Dutch", Count: 2},
	}

	// Largest period total is 4 (1500–1549).
	if got := barShare(rows, "1500–1549", "Italian"); got != 75.0 {
		t.Errorf("Italian bar share = %.3f, want 75.0", got)
	}
	if got := barShare(rows, "1500–1549", "French"); got != 25.0 {
		t.Errorf("French bar share = %.3f, want 25.0", got)
	}
	if got := barShare(rows, "1550–1599", "Dutch"); got != 50.0 {
		t.Errorf("Dutch bar share = %.3f, want 50.0", got)
	}
	if got := barShare(rows, "1550–1599", "Italian"); got != 0 {
		t.Errorf("missing Italian share = %.3f, want 0", got)
	}
	if got := barShare(nil, "1500–1549", "Italian"); got != 0 {
		t.Errorf("empty bar share = %.3f, want 0", got)
	}
}

func TestSchoolPeriodHelpersDeriveFromSameRows(t *testing.T) {
	rows := []StatisticsSchoolPeriodRow{
		{Period: "1500–1549", School: "Italian", Count: 3},
		{Period: "1500–1549", School: "French", Count: 1},
		{Period: "1500–1549", School: "Other", Count: 2},
		{Period: "1550–1599", School: "Dutch", Count: 4},
	}

	if got := periodsOf(rows); len(got) != 2 || got[0] != "1500–1549" || got[1] != "1550–1599" {
		t.Errorf("periodsOf = %v, want [1500–1549 1550–1599]", got)
	}
	if got := countFor(rows, "1500–1549", "Italian"); got != 3 {
		t.Errorf("countFor Italian = %d, want 3", got)
	}
	if got := countFor(rows, "1500–1549", "Dutch"); got != 0 {
		t.Errorf("countFor missing Dutch = %d, want 0", got)
	}
	if got := periodTotal(rows, "1500–1549"); got != 6 {
		t.Errorf("periodTotal 1500 = %d, want 6", got)
	}
	if got := periodTotal(rows, "1550–1599"); got != 4 {
		t.Errorf("periodTotal 1550 = %d, want 4", got)
	}
}

// TestStatisticsKeysDrawOneSwatchPerRow guards design-audit ST-1: each key row
// draws exactly one swatch, and that swatch names the series token the chart
// script reads back, so the key and the chart share one colour source.
func TestStatisticsKeysDrawOneSwatchPerRow(t *testing.T) {
	forms := make([]StatisticsArtFormRow, 9)
	for i := range forms {
		forms[i] = StatisticsArtFormRow{Name: fmt.Sprintf("Form %d", i), Count: i + 1}
	}

	var summary bytes.Buffer
	if err := artFormSummary(forms).Render(context.Background(), &summary); err != nil {
		t.Fatalf("render art form summary: %v", err)
	}
	rows := strings.Split(summary.String(), `<tr class="border-b`)[1:]
	if len(rows) != len(forms) {
		t.Fatalf("expected %d key rows, got %d", len(forms), len(rows))
	}
	for i, row := range rows {
		if got := strings.Count(row, "data-series-token="); got != 1 {
			t.Errorf("art form row %d draws %d swatches, want 1", i, got)
		}
		want := fmt.Sprintf(`data-series-token="--wga-series-%d" data-series-fill="solid" style="background:var(--wga-series-%d);"`, i%7, i%7)
		if !strings.Contains(row, want) {
			t.Errorf("art form row %d swatch: want %s\ngot: %s", i, want, row)
		}
	}

	var legend bytes.Buffer
	if err := schoolLegend().Render(context.Background(), &legend); err != nil {
		t.Fatalf("render school legend: %v", err)
	}
	items := strings.Split(legend.String(), "<li ")[1:]
	if len(items) != len(schoolOrder) {
		t.Fatalf("expected %d school key items, got %d", len(schoolOrder), len(items))
	}
	for i, item := range items {
		if got := strings.Count(item, "data-series-token="); got != 1 {
			t.Errorf("school key item %d draws %d swatches, want 1", i, got)
		}
	}
	if !strings.Contains(items[len(items)-1], `data-series-fill="hatch"`) {
		t.Errorf("expected Other to be a hatch, got: %s", items[len(items)-1])
	}
}

// fiftyYearLabel mirrors the handler's period formatting so tests can build the
// same fifty-year labels without importing the handler package.
func fiftyYearLabel(start int) string {
	return fmt.Sprintf("%d–%d", start, start+49)
}
