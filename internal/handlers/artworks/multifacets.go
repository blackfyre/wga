package artworks

import (
	"sort"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbsearch "github.com/pocketbase/pocketbase/tools/search"
)

const artworkMultiFacetInitialLimit = 8

type artworkMultiFacetKind uint8

const (
	schoolMultiFacet artworkMultiFacetKind = iota
	formMultiFacet
)

func buildCountedMultiFacet(app *pocketbase.PocketBase, f *filters, dual *pages.ArtworkSearchDualMode, kind artworkMultiFacetKind, options map[string]string) (pages.ArtworkSearchMultiFacet, error) {
	label, name, selected, expanded, openByDefault := multiFacetState(f, kind)
	selectedSet := make(map[string]bool, len(selected))
	for _, value := range selected {
		selectedSet[value] = true
		if _, known := options[value]; !known {
			options[value] = value
		}
	}
	counts, err := loadMultiFacetCounts(app, f, kind)
	if err != nil {
		return pages.ArtworkSearchMultiFacet{}, err
	}

	counted := make([]pages.ArtworkSearchMultiOption, 0, len(options))
	for value, optionLabel := range options {
		if value == "" {
			continue
		}
		count := counts[value]
		counted = append(counted, pages.ArtworkSearchMultiOption{
			Label:    optionLabel,
			Value:    value,
			Count:    count,
			Selected: selectedSet[value],
			Disabled: count == 0 && !selectedSet[value],
		})
	}
	sort.Slice(counted, func(i, j int) bool {
		if counted[i].Count != counted[j].Count {
			return counted[i].Count > counted[j].Count
		}
		if counted[i].Label != counted[j].Label {
			return counted[i].Label < counted[j].Label
		}
		return counted[i].Value < counted[j].Value
	})

	visible := counted
	if !expanded && len(counted) > artworkMultiFacetInitialLimit {
		visible = append([]pages.ArtworkSearchMultiOption(nil), counted[:artworkMultiFacetInitialLimit]...)
		visibleValues := map[string]bool{}
		for _, option := range visible {
			visibleValues[option.Value] = true
		}
		for _, option := range counted[artworkMultiFacetInitialLimit:] {
			if option.Selected && !visibleValues[option.Value] {
				visible = append(visible, option)
			}
		}
	}

	clear := f.clearMultiFacet(kind)
	toggle := f.toggleMultiFacet(kind)
	facet := pages.ArtworkSearchMultiFacet{
		ArtworkSearchFacet: pages.ArtworkSearchFacet{
			Label:   label,
			Summary: multiFacetSummary(selected, options),
			Active:  len(selected) > 0,
			Open:    openByDefault || len(selected) > 0,
		},
		Name:     name,
		Options:  visible,
		Total:    len(counted),
		ClearURL: buildArtworkSearchPath("/artworks", clear, dual),
	}
	if len(counted) > artworkMultiFacetInitialLimit {
		facet.ToggleURL = buildArtworkSearchPath("/artworks", toggle, dual)
		facet.ToggleLabel = "SHOW ALL " + strconv.Itoa(len(counted))
		if expanded {
			facet.ToggleLabel = "SHOW FEWER"
		}
	}
	return facet, nil
}

type artworkMultiFacetCount struct {
	Value *string `db:"facet_value"`
	Count int     `db:"facet_count"`
}

// loadMultiFacetCounts applies every result filter except the facet being
// counted, then groups the matching artwork IDs by that facet in one query.
func loadMultiFacetCounts(app *pocketbase.PocketBase, f *filters, kind artworkMultiFacetKind) (map[string]int, error) {
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return nil, err
	}
	withoutOwnFacet := f.withoutMultiFacet(kind)
	filter, params := withoutOwnFacet.BuildFilter()
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)
	field := "form.slug"
	if kind == schoolMultiFacet {
		field = "school.slug"
	}
	resolved, err := resolver.Resolve(field)
	if err != nil {
		return nil, err
	}
	expression, err := pbsearch.FilterData(filter).BuildExpr(resolver, params)
	if err != nil {
		return nil, err
	}
	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	query := app.RecordQuery(collection).
		Select(resolved.Identifier+" AS facet_value", "COUNT(DISTINCT "+baseID+") AS facet_count").
		AndWhere(expression).
		GroupBy("facet_value")
	if err := resolver.UpdateQuery(query); err != nil {
		return nil, err
	}
	rows := []artworkMultiFacetCount{}
	if err := query.All(&rows); err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		if row.Value != nil && *row.Value != "" {
			counts[*row.Value] = row.Count
		}
	}
	return counts, nil
}

func multiFacetState(f *filters, kind artworkMultiFacetKind) (label string, name string, selected []string, expanded bool, openByDefault bool) {
	if kind == schoolMultiFacet {
		return "SCHOOL", "art_school", f.schoolValues(), f.SchoolExpanded, true
	}
	return "FORM", "art_form", f.formValues(), f.FormExpanded, false
}

func multiFacetSummary(selected []string, options map[string]string) string {
	if len(selected) == 0 {
		return "ANY"
	}
	if len(selected) > 1 {
		return strconv.Itoa(len(selected)) + " SELECTED"
	}
	return strings.ToUpper(options[selected[0]])
}

func (f *filters) withoutMultiFacet(kind artworkMultiFacetKind) *filters {
	next := f.clone()
	if kind == schoolMultiFacet {
		next.SchoolString = ""
		next.SchoolValues = nil
	} else {
		next.ArtFormString = ""
		next.ArtFormValues = nil
	}
	return next
}

func (f *filters) clearMultiFacet(kind artworkMultiFacetKind) *filters {
	next := f.withoutMultiFacet(kind)
	next.Page = ""
	return next
}

func (f *filters) toggleMultiFacet(kind artworkMultiFacetKind) *filters {
	next := f.clone()
	if kind == schoolMultiFacet {
		next.SchoolExpanded = !next.SchoolExpanded
	} else {
		next.FormExpanded = !next.FormExpanded
	}
	return next
}
