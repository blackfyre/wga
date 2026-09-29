package artists

import (
	"strconv"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

const artistIndexPageName = "Artists"

// artistIndexTitleSteps is the reduction order for artist-index titles.
var artistIndexTitleSteps = []tmplUtils.TitleStep{
	{Kind: tmplUtils.TitleRemove, Key: "position"},
	{Kind: tmplUtils.TitleShorten, Key: "born"},
	{Kind: tmplUtils.TitleShorten, Key: "letter"},
	{Kind: tmplUtils.TitleRemove, Key: "born"},
	{Kind: tmplUtils.TitleRemove, Key: "period"},
	{Kind: tmplUtils.TitleRemove, Key: "school"},
	{Kind: tmplUtils.TitleTruncate, Key: "q", Limit: 20},
}

// artistIndexTitle derives the document title from the rendered index state.
// Parts follow the order of their controls on the page; view and sort are
// presentation only and never contribute.
func artistIndexTitle(view pages.ArtistsView) string {
	parts := []tmplUtils.TitlePart{
		{Key: "q", Long: view.NameQuery, Role: tmplUtils.TitleLead, Quoted: true},
		{Key: "page", Long: artistIndexPageName, Role: tmplUtils.TitlePageName},
	}
	if view.SelectedLetter != "" {
		parts = append(parts, tmplUtils.TitlePart{Key: "letter", Long: "Letter " + view.SelectedLetter, Short: view.SelectedLetter, Role: tmplUtils.TitleFilter})
	}
	if school := selectedOptionLabel(view.Schools); school != "" {
		parts = append(parts, tmplUtils.TitlePart{Key: "school", Long: school, Role: tmplUtils.TitleFilter})
	}
	if period := selectedOptionLabel(view.Periods); period != "" {
		parts = append(parts, tmplUtils.TitlePart{Key: "period", Long: period, Role: tmplUtils.TitleFilter})
	}
	if born := view.BornRange; view.HasBirthBounds && (born.FromValue != born.Min || born.ToValue != born.Max) {
		bounds := strconv.Itoa(born.FromValue) + "–" + strconv.Itoa(born.ToValue)
		parts = append(parts, tmplUtils.TitlePart{Key: "born", Long: "born " + bounds, Short: "b. " + bounds, Role: tmplUtils.TitleFilter})
	}
	parts = append(parts, tmplUtils.TitlePart{Key: "position", Long: tmplUtils.TitlePagePosition(view.Page, view.PageCount), Role: tmplUtils.TitlePosition})

	return tmplUtils.FitTitle(parts, artistIndexTitleSteps)
}

// selectedOptionLabel returns the label of the selected non-empty option.
func selectedOptionLabel(options []dto.SelectOption) string {
	for _, option := range options {
		if option.Selected && option.Value != "" {
			return option.Label
		}
	}
	return ""
}
