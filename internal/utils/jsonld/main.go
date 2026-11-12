// Package jsonld builds schema.org JSON-LD for public catalogue pages. Every
// value describes content the page visibly renders; missing values are omitted
// rather than invented.
package jsonld

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

var (
	fourDigitYearPattern = regexp.MustCompile(`^[0-9]{4}$`)
)

// ArtistURL returns the absolute canonical artist record URL.
func ArtistURL(r *core.Record) string {
	return utils.AssetUrl("/artists/" + utils.GenerateArtistSlug(r))
}

// ArtistJsonLd describes an artist as shown on the artist record page.
func ArtistJsonLd(r *core.Record) Person {
	person := ArtistSummaryJsonLd(r)
	person.Description = PlainText(r.GetString("bio"))
	if place := strings.TrimSpace(r.GetString("place_of_birth")); place != "" {
		person.BirthPlace = &Place{Type: "Place", Name: place}
	}
	if place := strings.TrimSpace(r.GetString("place_of_death")); place != "" {
		person.DeathPlace = &Place{Type: "Place", Name: place}
	}
	if occupation := strings.TrimSpace(r.GetString("profession")); occupation != "" {
		person.HasOccupation = &Occupation{Type: "Occupation", Name: occupation}
	}
	if portraitURL := url.GenerateArtistPortraitImageURL(r, url.DeliveryProfilePortraitRecordAndWorkFallback, ""); portraitURL != "" {
		person.Image = utils.AssetUrl(portraitURL)
	}
	return person
}

// ArtistSummaryJsonLd describes an artist as an index row shows it: name,
// record link and life dates.
func ArtistSummaryJsonLd(r *core.Record) Person {
	canonical := ArtistURL(r)
	person := Person{
		Type: "Person",
		ID:   canonical + PrimarySuffix,
		Name: strings.TrimSpace(r.GetString("filing_name")),
		Url:  canonical,
	}
	if year := r.GetInt("year_of_birth"); year > 0 {
		person.BirthDate = strconv.Itoa(year)
	}
	if year := r.GetInt("year_of_death"); year > 0 {
		person.DeathDate = strconv.Itoa(year)
	}
	return person
}

// ArtworkInput holds the visible artwork facts a record page renders.
type ArtworkInput struct {
	// CanonicalURL is the absolute canonical artwork record URL.
	CanonicalURL string
	Title        string
	// Commentary is the source comment HTML; it is reduced to plain text.
	Commentary string
	// ImageURL is the absolute URL of the rendered reproduction, or empty
	// when the record has no image.
	ImageURL  string
	Technique string
	ArtType   string
	// Year is the displayed creation year; only a four-digit year is emitted.
	Year   string
	Artist *core.Record
}

// ArtworkJsonLd describes an artwork and links its creator by @id to the
// creator's artist record.
func ArtworkJsonLd(in ArtworkInput) VisualArtwork {
	artwork := VisualArtwork{
		Type:        "VisualArtwork",
		ID:          in.CanonicalURL + PrimarySuffix,
		Name:        strings.TrimSpace(in.Title),
		Url:         in.CanonicalURL,
		Description: PlainText(in.Commentary),
		Image:       in.ImageURL,
		Artform:     strings.TrimSpace(in.ArtType),
		ArtMedium:   strings.TrimSpace(in.Technique),
	}
	if year := strings.TrimSpace(in.Year); fourDigitYearPattern.MatchString(year) {
		artwork.DateCreated = year
	}
	if in.Artist != nil {
		canonical := ArtistURL(in.Artist)
		artwork.Creator = &Person{
			Type: "Person",
			ID:   canonical + PrimarySuffix,
			Name: strings.TrimSpace(in.Artist.GetString("filing_name")),
			Url:  canonical,
		}
	}
	return artwork
}

// SelectionItem is one work a selection page shows.
type SelectionItem struct {
	Name string
	// URL is the absolute canonical artwork URL.
	URL string
}

// SelectionJsonLd describes a curated selection as an ordered list of the
// works the page shows. Works without a name or URL are skipped.
func SelectionJsonLd(canonical string, name string, items []SelectionItem) ItemList {
	list := ItemList{
		Type: "ItemList",
		ID:   canonical + PrimarySuffix,
		Name: strings.TrimSpace(name),
		Url:  canonical,
	}
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" || item.URL == "" {
			continue
		}
		list.ItemListElement = append(list.ItemListElement, ListItem{
			Type:     "ListItem",
			Position: len(list.ItemListElement) + 1,
			Name:     strings.TrimSpace(item.Name),
			Item:     item.URL,
		})
	}
	list.NumberOfItems = len(list.ItemListElement)
	return list
}

// Crumb is one visible, linked breadcrumb step.
type Crumb struct {
	Name string
	// URL is the absolute URL of the step.
	URL string
}

// Breadcrumbs describes a visible breadcrumb trail ending at the current page.
// It returns nil when the trail has fewer than two steps.
func Breadcrumbs(pageURL string, crumbs []Crumb) *BreadcrumbList {
	if len(crumbs) < 2 {
		return nil
	}
	list := &BreadcrumbList{Type: "BreadcrumbList", ID: pageURL + "#breadcrumb"}
	for index, crumb := range crumbs {
		list.ItemListElement = append(list.ItemListElement, ListItem{
			Type:     "ListItem",
			Position: index + 1,
			Name:     strings.TrimSpace(crumb.Name),
			Item:     crumb.URL,
		})
	}
	return list
}

// ScriptTag serialises nodes as one JSON-LD @graph inside a script element.
// Nil nodes are skipped, and an empty graph renders nothing. encoding/json
// escapes "<", ">" and "&", so record text cannot close the element.
func ScriptTag(nodes ...any) (string, error) {
	graph := make([]any, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if list, ok := node.(*BreadcrumbList); ok && list == nil {
			continue
		}
		graph = append(graph, node)
	}
	if len(graph) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(Document{Context: Context, Graph: graph})
	if err != nil {
		return "", err
	}
	return `<script type="application/ld+json">` + string(encoded) + `</script>`, nil
}

// PlainText reduces record HTML to single-spaced plain text.
func PlainText(value string) string {
	return tmplUtils.PlainText(value)
}
