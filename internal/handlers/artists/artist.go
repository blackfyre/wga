package artists

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/glossary"
	"github.com/blackfyre/wga/internal/utils/jsonld"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var bioSanitizer = biographySanitizer()

// biographySanitizer returns the record-owned biography sanitizer. It retains
// legitimate editorial structure while removing scripts, event handlers, and
// unsafe URL schemes before the biography reaches templ.Raw.
func biographySanitizer() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements("a", "b", "blockquote", "br", "em", "i", "li", "ol", "p", "strong", "sub", "sup", "u", "ul")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowStandardURLs()

	return policy
}

// annotateBiography renders the persisted biography editor HTML as safe,
// glossary-annotated prose. Glossary terms use the task-4.4 keyboard/focus
// contract (a .wga-term dfn with a complete accessible name and an inline,
// aria-hidden tooltip). When no glossary entries are available the sanitized
// biography is returned unchanged.
func annotateBiography(bio string, entries []glossary.GlossaryEntry) string {
	sanitized := bioSanitizer.Sanitize(bio)
	if len(entries) == 0 {
		return sanitized
	}

	rewritten, err := rewriteGlossaryTerms(glossary.AnnotateHTML(sanitized, entries))
	if err != nil {
		return sanitized
	}

	return rewritten
}

// rewriteGlossaryTerms converts the legacy glossary-term annotation produced by
// glossary.AnnotateHTML into the task-4.4 .wga-term contract.
func rewriteGlossaryTerms(annotated string) (string, error) {
	fragmentContext := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(annotated), fragmentContext)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		rewriteGlossaryNode(node)
	}

	var buff bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&buff, node); err != nil {
			return "", err
		}
	}

	return buff.String(), nil
}

func rewriteGlossaryNode(n *html.Node) {
	children := make([]*html.Node, 0)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}

	for i := 0; i < len(children); i++ {
		child := children[i]
		if isGlossaryTermSpan(child) {
			definition := ""
			var definitionNode *html.Node
			if i+1 < len(children) && isGlossaryDefinitionTemplate(children[i+1]) {
				definitionNode = children[i+1]
				definition = nodeText(definitionNode)
			}

			n.InsertBefore(buildWgaTermNode(nodeText(child), definition), child)
			n.RemoveChild(child)
			if definitionNode != nil {
				n.RemoveChild(definitionNode)
			}
			continue
		}
		rewriteGlossaryNode(child)
	}
}

func isGlossaryTermSpan(n *html.Node) bool {
	return n.Type == html.ElementNode && n.DataAtom == atom.Span && hasNodeClass(n, "glossary-term")
}

func isGlossaryDefinitionTemplate(n *html.Node) bool {
	return n.Type == html.ElementNode && n.DataAtom == atom.Template && hasNodeClass(n, "glossary-definition")
}

func hasNodeClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, value := range strings.Fields(attr.Val) {
				if value == class {
					return true
				}
			}
		}
	}

	return false
}

// nodeText returns the plain text content of a node, excluding markup.
func nodeText(n *html.Node) string {
	var buff bytes.Buffer
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			buff.WriteString(node.Data)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return strings.TrimSpace(buff.String())
}

// buildWgaTermNode builds the task-4.4 glossary-term dfn with an inline
// aria-hidden tooltip and a complete accessible name.
func buildWgaTermNode(term string, definition string) *html.Node {
	dfn := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Dfn,
		Data:     "dfn",
		Attr: []html.Attribute{
			{Key: "class", Val: "wga-term"},
			{Key: "role", Val: "note"},
			{Key: "tabindex", Val: "0"},
			{Key: "aria-label", Val: term + ": " + definition},
			{Key: "data-bionic", Val: "off"},
		},
	}
	dfn.AppendChild(&html.Node{Type: html.TextNode, Data: term})

	tooltip := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Span,
		Data:     "span",
		Attr: []html.Attribute{
			{Key: "class", Val: "wga-term__tooltip"},
			{Key: "aria-hidden", Val: "true"},
		},
	}
	meta := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Span,
		Data:     "span",
		Attr:     []html.Attribute{{Key: "class", Val: "wga-tooltip__meta"}},
	}
	meta.AppendChild(&html.Node{Type: html.TextNode, Data: "GLOSSARY"})
	tooltip.AppendChild(meta)

	if definition != "" {
		body := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Span,
			Data:     "span",
			Attr:     []html.Attribute{{Key: "class", Val: "wga-tooltip__body"}},
		}
		body.AppendChild(&html.Node{Type: html.TextNode, Data: definition})
		tooltip.AppendChild(body)
	}

	dfn.AppendChild(tooltip)

	return dfn
}

// RenderArtistContent renders the content of an artist by generating a DTO (Data Transfer Object) that contains
// information about the artist, their works, and JSON-LD metadata. It takes the PocketBase application instance,
// the Echo context, and the artist record as input parameters. It returns the DTO representing the artist content
// and an error if any occurred during the process.
func RenderArtistContent(app *pocketbase.PocketBase, c *core.RequestEvent, artist *core.Record, hxTarget string, showBreadcrumbs bool) (dto.Artist, error) {
	id := artist.GetString("id")
	expectedSlug := utils.GenerateArtistSlug(artist)

	works, err := utils.FindArtworksByAuthorID(app, id)

	if err != nil {
		app.Logger().Error("Error finding artworks: ", "error", err.Error())
		return dto.Artist{}, errs.ErrArtistNotFound
	}

	schools := utils.RenderSchoolNames(app, artist.GetStringSlice("school"))

	content := dto.Artist{
		FilingName: artist.GetString("filing_name"),
		ShortName:  artist.GetString("short_name"),
		Name:       artist.GetString("filing_name"),
		Bio:        artist.GetString("bio"),
		BioExcerpt: utils.NormalizedBioExcerpt(utils.BioExcerptDTO{
			YearOfBirth:       artist.GetInt("year_of_birth"),
			ExactYearOfBirth:  artist.GetString("exact_year_of_birth"),
			PlaceOfBirth:      artist.GetString("place_of_birth"),
			KnownPlaceOfBirth: artist.GetString("known_place_of_birth"),
			YearOfDeath:       artist.GetInt("year_of_death"),
			ExactYearOfDeath:  artist.GetString("exact_year_of_death"),
			PlaceOfDeath:      artist.GetString("place_of_death"),
			KnownPlaceOfDeath: artist.GetString("known_place_of_death"),
		}),
		Schools:         schools,
		Profession:      artist.GetString("profession"),
		Portrait:        url.GenerateArtistPortraitImageURL(artist, url.DeliveryProfilePortraitRecordAndWorkFallback, ""),
		Works:           dto.ImageGrid{},
		Url:             "/artists/" + expectedSlug,
		HxTarget:        hxTarget,
		ShowBreadcrumbs: showBreadcrumbs,
	}

	JsonLd := jsonld.ArtistJsonLd(artist)

	marshalled, err := json.Marshal(JsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artist jsonld for"+id, "error", err.Error())
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	app.Logger().Debug("Rendering artist content", "artistId", id, "artistName", artist.GetString("name"), "worksCount", len(works))

	for _, w := range works {

		artJsonLd := jsonld.ArtworkJsonLd(w, artist)

		marshalled, err := json.Marshal(artJsonLd)

		if err != nil {
			app.Logger().Error("Error marshalling artwork jsonld for"+w.GetString("id"), "error", err.Error())
		}

		var img dto.Image

		img.Id = w.GetString("id")
		img.Title = w.GetString("title")
		img.Comment = w.GetString("comment")
		img.Technique = w.GetString("technique")

		if w.GetString("image") != "" {
			img.Image = url.GenerateArtworkImageURL(w, url.DeliveryProfileCardAndArtistIndex, "")
			img.Thumb = img.Image
		} else {
			img.Image = utils.AssetUrl("/assets/images/no-image.png")
			img.Thumb = utils.AssetUrl("/assets/images/no-image.png")
		}

		img.Url = "/artists/" + expectedSlug + "/" + utils.Slugify(w.GetString("title")) + "-" + w.GetString("id")
		img.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)
		img.HxTarget = hxTarget

		content.Works = append(content.Works, img)
	}

	// Annotate bio with glossary terms (after content is fully built,
	// so callers can use the raw Bio for meta descriptions first). The
	// biography is sanitised regardless of glossary availability so unsafe
	// persisted markup never reaches templ.Raw.
	glossaryEntries, glossaryErr := glossary.GetGlossaryEntries(app)
	if glossaryErr != nil {
		app.Logger().Warn("Failed to load glossary entries", "error", glossaryErr)
	}
	content.Bio = annotateBiography(content.Bio, glossaryEntries)

	return content, nil
}

// processArtist is the public artist record adapter. It resolves the record
// through the bounded read-model, renders the full page or the HTMX block, and
// pushes the canonical record URL.
func processArtist(c *core.RequestEvent, app *pocketbase.PocketBase) error {
	slug := c.Request.PathValue("name")

	id := utils.ExtractIdFromString(slug)
	artist, err := repositories.NewArtistRecordRepository(app).FindPublishedArtist(id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.Logger().Error("Find published artist", "slug", slug, "error", err.Error())
		}
		return utils.NotFoundError(c)
	}

	expectedSlug := utils.GenerateArtistSlug(artist)
	fullUrl := "/artists/" + expectedSlug

	if slug != expectedSlug {
		return c.Redirect(http.StatusMovedPermanently, fullUrl)
	}

	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		app.Logger().Error("Build artist record", "slug", slug, "error", err.Error())
		return utils.ServerFaultError(c)
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", view.FilingName, view.LifeSummary))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, artist.GetString("bio"))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, utils.AssetUrl(fullUrl))
	if image := artistOpenGraphImage(view); image != "" {
		ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, image)
	}

	c.Response.Header().Set("HX-Push-Url", fullUrl)

	var buff bytes.Buffer
	if utils.IsHtmxRequest(c) {
		err = pages.ArtistRecordBlock(view).Render(ctx, &buff)
	} else {
		err = pages.ArtistRecordPage(view).Render(ctx, &buff)
	}
	if err != nil {
		app.Logger().Error("Error rendering artist page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

// buildArtistRecordView assembles the page-owned artist record view from the
// bounded read-model.
func buildArtistRecordView(app *pocketbase.PocketBase, artist *core.Record) (pages.ArtistView, error) {
	expectedSlug := utils.GenerateArtistSlug(artist)

	repo := repositories.NewArtistRecordRepository(app)

	workCount, err := repo.CountPublishedWorks(artist.Id)
	if err != nil {
		return pages.ArtistView{}, err
	}
	works, err := repo.ListPublishedWorks(artist.Id, 0)
	if err != nil {
		return pages.ArtistView{}, err
	}

	selections, err := buildSelectionPreviews(app, artist, workCount)
	if err != nil {
		return pages.ArtistView{}, err
	}

	schoolNames, err := repo.ListSchoolNames(artist.GetStringSlice("school"))
	if err != nil {
		return pages.ArtistView{}, err
	}

	periodRecords, err := repo.ListMatchingArtPeriods(artist.GetInt("year_of_birth"))
	if err != nil {
		return pages.ArtistView{}, err
	}

	glossaryEntries, glossaryErr := glossary.GetGlossaryEntries(app)
	if glossaryErr != nil {
		app.Logger().Warn("Failed to load glossary entries", "error", glossaryErr)
	}
	bio := annotateBiography(artist.GetString("bio"), glossaryEntries)

	periodSong, err := repo.MatchPeriodSong(artist.GetInt("year_of_birth"))
	if err != nil {
		return pages.ArtistView{}, err
	}

	personJsonLd := jsonld.ArtistJsonLd(artist)
	marshalled, err := json.Marshal(personJsonLd)
	if err != nil {
		app.Logger().Error("Error marshalling artist jsonld", "artistId", artist.Id, "error", err.Error())
	}

	view := pages.ArtistView{
		FilingName:      artist.GetString("filing_name"),
		ShortName:       artist.GetString("short_name"),
		Url:             "/artists/" + expectedSlug,
		LifeSummary:     artistLifeSummary(artist),
		Schools:         strings.Join(schoolNames, ", "),
		Period:          unambiguousPeriodName(periodRecords),
		Profession:      artist.GetString("profession"),
		Aliases:         resolveAliases(app, artist.GetStringSlice("also_known_as")),
		Portrait:        url.GenerateArtistPortraitImageURL(artist, url.DeliveryProfilePortraitRecordAndWorkFallback, ""),
		Bio:             bio,
		Works:           buildRecordWorkImages(artist, works),
		WorkCount:       workCount,
		WorksURL:        buildArtistWorksURL(artist.GetString("filing_name")),
		Selections:      selections,
		Music:           buildRecordMusic(periodSong),
		Citation:        buildArtistCitation(artist, expectedSlug),
		Jsonld:          fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled),
		ShowBreadcrumbs: true,
	}

	return view, nil
}

// unambiguousPeriodName returns the period name only when exactly one period
// row matched the birth year; ambiguous or empty matches return "" so the
// record never invents a period label.
func unambiguousPeriodName(records []*core.Record) string {
	if len(records) != 1 {
		return ""
	}

	return records[0].GetString("name")
}

// artistLifeSummary builds the truthful birth/death summary from the stored
// year and place fields without inventing missing values.
func artistLifeSummary(artist *core.Record) string {
	birth := artist.GetInt("year_of_birth")
	death := artist.GetInt("year_of_death")

	var parts []string
	if birth > 0 {
		parts = append(parts, "b. "+yearAndPlace(birth, artist.GetString("place_of_birth")))
	}
	if death > 0 {
		parts = append(parts, "d. "+yearAndPlace(death, artist.GetString("place_of_death")))
	}

	return strings.Join(parts, " · ")
}

func yearAndPlace(year int, place string) string {
	if place != "" {
		return strconv.Itoa(year) + " " + place
	}

	return strconv.Itoa(year)
}

// resolveAliases returns the comma-separated display names of the artist's
// also-known-as relations, or an empty string when none exist.
func resolveAliases(app *pocketbase.PocketBase, ids []string) string {
	if len(ids) == 0 {
		return ""
	}

	records, err := app.FindRecordsByIds(constants.CollectionArtists, ids)
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(records))
	for _, record := range records {
		if name := record.GetString("filing_name"); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	return strings.Join(names, ", ")
}

// buildRecordWorkImages maps published works onto the canonical record-card
// projection with handler-resolved 500-profile-or-original images and no
// viewer hooks.
func buildRecordWorkImages(artist *core.Record, works []*core.Record) dto.ImageGrid {
	images := dto.ImageGrid{}
	for _, work := range works {
		image := utils.AssetUrl("/assets/images/no-image.png")
		if work.GetString("image") != "" {
			image = url.GenerateArtworkImageURL(work, url.DeliveryProfileCardAndArtistIndex, "")
		}

		images = append(images, dto.Image{
			Id:        work.GetString("id"),
			Title:     work.GetString("title"),
			Technique: work.GetString("technique"),
			Metadata:  artworkDateMetadata(work),
			Image:     image,
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   artist.GetString("name"),
				ArtistId:     artist.Id,
				ArtworkTitle: work.GetString("title"),
				ArtworkId:    work.GetString("id"),
			}),
			Artist: dto.Artist{
				FilingName: artist.GetString("filing_name"),
				ShortName:  artist.GetString("short_name"),
				Name:       artist.GetString("filing_name"),
			},
		})
	}

	return images
}

func artworkDateMetadata(artwork *core.Record) string {
	if dateEnd := artwork.GetInt("date_end"); dateEnd > 0 {
		return strconv.Itoa(dateEnd)
	}

	return artwork.GetString("year")
}

// buildArtistWorksURL returns the wider catalogue route filtered to the
// artist's authoritative filing name.
func buildArtistWorksURL(filingName string) string {
	values := neturl.Values{}
	values.Set("artist", filingName)

	return "/artworks?" + values.Encode()
}

// selectionPreviewWorkLimit bounds the representative works shown in an artist
// record selection preview. The full ordered membership remains available on
// the dedicated selection route.
const selectionPreviewWorkLimit = 4

// buildSelectionPreviews assembles the artist record's selection previews. It
// returns previews only when the artist has more than one published source-backed
// selection, so a single selection never distorts the ordinary works holding.
// Each preview carries the supplied display title, selected and catalogued
// counts, sanitised commentary, bounded representative works, and the stable
// selection route.
func buildSelectionPreviews(app *pocketbase.PocketBase, artist *core.Record, workCount int) ([]pages.SelectionPreview, error) {
	repo := repositories.NewArtistSelectionsRepository(app)

	count, err := repo.CountPublishedSelections(artist.Id)
	if err != nil {
		return nil, err
	}
	if count <= 1 {
		return nil, nil
	}

	selections, err := repo.ListPublishedSelections(artist.Id, 0)
	if err != nil {
		return nil, err
	}

	slug := utils.GenerateArtistSlug(artist)
	previews := make([]pages.SelectionPreview, 0, len(selections))
	for _, selection := range selections {
		works, err := repo.ListSelectionArtworks(artist.Id, selection)
		if err != nil {
			return nil, err
		}

		commentary := sanitizeSelectionCommentary(selection.GetString("commentary"))
		previews = append(previews, pages.SelectionPreview{
			URL:             buildSelectionURL(slug, selection.Id),
			DisplayTitle:    selection.GetString("display_title"),
			SelectedCount:   len(works),
			CataloguedCount: workCount,
			Commentary:      commentary,
			HasCommentary:   commentary != "",
			Works:           buildRecordWorkImages(artist, capSelectionPreviewWorks(works)),
		})
	}

	return previews, nil
}

// capSelectionPreviewWorks bounds the representative works shown in a preview.
func capSelectionPreviewWorks(works []*core.Record) []*core.Record {
	if len(works) <= selectionPreviewWorkLimit {
		return works
	}

	return works[:selectionPreviewWorkLimit]
}

// buildRecordMusic maps a deterministic period-song match onto the validated
// player-route card, or returns an empty card when no truthful match exists.
func buildRecordMusic(song *repositories.PeriodSong) components.MusicPeriodCard {
	if song == nil || song.Record == nil {
		return components.MusicPeriodCard{}
	}

	piece := strings.TrimSpace(song.Record.GetString("title"))
	if piece == "" || song.Record.GetString("source") == "" {
		return components.MusicPeriodCard{}
	}

	return components.MusicPeriodCard{
		SongID:    song.Record.Id,
		Piece:     piece,
		PlayerURL: "/player?song=" + song.Record.Id,
	}
}

// buildArtistCitation builds the canonical BibTeX citation for the artist.
func buildArtistCitation(artist *core.Record, expectedSlug string) components.Citation {
	return components.Citation{
		Key:   "wga-" + artist.GetString("slug"),
		Title: artist.GetString("filing_name"),
		URL:   utils.AssetUrl("/artists/" + expectedSlug),
	}
}

func artistOpenGraphImage(view pages.ArtistView) string {
	if view.Portrait != "" {
		return utils.AssetUrl(view.Portrait)
	}
	if len(view.Works) > 0 {
		return utils.AssetUrl(view.Works[0].Image)
	}

	return ""
}
