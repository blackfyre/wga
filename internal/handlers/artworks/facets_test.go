package artworks

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
)

func TestTask71ChipFacetKnownUnknownAndEmpty(t *testing.T) {
	known := chipFacet("SCHOOL", dto.ChipGroup{
		Options: []dto.ChipOption{
			{Label: "ALL", Value: "", Checked: false},
			{Label: "Dutch", Value: "dutch", Checked: true},
		},
	}, "dutch", false)
	if known.Summary != "DUTCH" || !known.Active || !known.Open {
		t.Fatalf("known chip facet = %#v, want active open DUTCH", known)
	}

	unknown := chipFacet("SCHOOL", dto.ChipGroup{
		Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: false}},
	}, "mystery-school", false)
	if unknown.Summary != "MYSTERY-SCHOOL" || !unknown.Active || !unknown.Open {
		t.Fatalf("unknown chip facet = %#v, want active open MYSTERY-SCHOOL", unknown)
	}

	empty := chipFacet("SCHOOL", dto.ChipGroup{
		Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}},
	}, "", true)
	if empty.Summary != "ANY" || empty.Active || !empty.Open {
		t.Fatalf("empty chip facet = %#v, want inactive ANY kept open by default", empty)
	}
}

func TestTask71UnknownEnumeratedFacetHonestSummaryAndPredicate(t *testing.T) {
	app := newArtworkSearchApp(t)
	view, canonical, err := buildArtworkSearchView(app, url.Values{
		"art_school": {"unknown-school"},
		"art_form":   {"unknown-form"},
		"art_type":   {"unknown-type"},
		"period":     {"unknown-period"},
	}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	for _, facet := range []struct {
		name string
		got  pages.ArtworkSearchFacet
		want string
	}{
		{"school", view.Facets.School, "UNKNOWN-SCHOOL"},
		{"form", view.Facets.Form, "UNKNOWN-FORM"},
		{"type", view.Facets.Type, "UNKNOWN-TYPE"},
		{"period", view.Facets.Period, "UNKNOWN-PERIOD"},
	} {
		if facet.got.Summary != facet.want || !facet.got.Active || !facet.got.Open {
			t.Errorf("%s facet = %#v, want active/open summary %q", facet.name, facet.got, facet.want)
		}
	}
	if view.Facets.ActiveCount != 4 {
		t.Fatalf("active count = %d, want 4", view.Facets.ActiveCount)
	}
	if view.Results.ResultCount != 0 {
		t.Fatalf("unknown values yielded %d results, want an honest empty result", view.Results.ResultCount)
	}
	for _, part := range []string{"art_school=unknown-school", "art_form=unknown-form", "art_type=unknown-type", "period=unknown-period"} {
		if !strings.Contains(canonical, part) {
			t.Errorf("canonical %q missing %q", canonical, part)
		}
	}
}

func TestTask71UnknownFacetSummaryEscapedAtTemplateBoundary(t *testing.T) {
	app := newArtworkSearchApp(t)
	view, _, err := buildArtworkSearchView(app, url.Values{"art_school": {"<script>alert(1)</script>"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if view.Facets.School.Summary != "<SCRIPT>ALERT(1)</SCRIPT>" {
		t.Fatalf("summary = %q, want the raw uppercase selected value", view.Facets.School.Summary)
	}

	var output bytes.Buffer
	if err := pages.ArtworkSeachFilterBlock(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render filter block: %v", err)
	}
	rendered := output.String()
	if strings.Contains(rendered, "<SCRIPT>") {
		t.Fatal("unknown facet summary was interpolated as raw HTML")
	}
	if !strings.Contains(rendered, "&lt;SCRIPT&gt;ALERT(1)&lt;/SCRIPT&gt;") {
		t.Fatalf("unknown facet summary was not escaped at the template boundary: %s", rendered)
	}
}

func TestTask71YearRangeInlinePresentation(t *testing.T) {
	app := newArtworkSearchApp(t)
	view, _, err := buildArtworkSearchView(app, url.Values{}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if !view.Facets.YearRange.Inline {
		t.Fatal("artwork year range must set the inline presentation flag")
	}

	var output bytes.Buffer
	if err := pages.ArtworkSeachFilterBlock(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render filter block: %v", err)
	}
	if strings.Contains(output.String(), "YEAR RANGE</legend>") {
		t.Fatal("inline year range must not duplicate its inner legend inside the facet")
	}
}
