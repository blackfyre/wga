package artists

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	neturl "net/url"
	"path"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/glossary"
	"github.com/blackfyre/wga/internal/utils/jsonld"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// findPublishedArtwork returns the published artwork with the given id, or
// sql.ErrNoRows when it is missing or unpublished.
func findPublishedArtwork(app *pocketbase.PocketBase, id string) (*core.Record, error) {
	return app.FindRecordById(constants.CollectionArtworks, id, func(q *dbx.SelectQuery) error {
		q.AndWhere(dbx.NewExp("published = true"))
		return nil
	})
}

// processArtwork processes the artwork based on the given context and PocketBase application.
// It retrieves the artist and artwork information, generates JSON-LD content, and renders the HTML template.
// If the artwork is found in the cache, it retrieves the HTML from the cache. Otherwise, it fetches the data from the database,
// generates the HTML, and stores it in the cache for future use.
// It also sets the "HX-Push-Url" header in the response.
// Parameters:
// - c: The echo.Context object representing the HTTP request and response.
// - app: The PocketBase application instance.
// Returns:
// - An error if any error occurs during the processing, or nil if the processing is successful.
func processArtwork(c *core.RequestEvent, app *pocketbase.PocketBase, environment config.Environment) error {
	artistSlug := c.Request.PathValue("name")
	artworkSlug := c.Request.PathValue("awid")

	// Split the slug on the last dash and use the last part as the artist id
	artistSlugParts := strings.Split(artistSlug, "-")
	artistId := artistSlugParts[len(artistSlugParts)-1]

	artist, err := repositories.NewArtistRecordRepository(app).FindPublishedArtist(artistId)

	// If the artist is not found or unpublished, return an indistinguishable 404.
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.Logger().Error("Artist not found: ", artistSlug, err)
		}
		return utils.NotFoundError(c)
	}

	// Generate the expected slug for the artist
	expectedArtistSlug := artist.GetString("slug") + "-" + artist.GetString("id")

	// Split the slug on the last dash and use the last part as the artwork id
	artworkSlugParts := strings.Split(artworkSlug, "-")
	artworkId := artworkSlugParts[len(artworkSlugParts)-1]

	// find the artwork by id, published only
	aw, err := findPublishedArtwork(app, artworkId)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.Logger().Error("Error finding artwork: ", artworkSlug, err)
		}
		return utils.NotFoundError(c)
	}

	belongsToArtist := false
	for _, authorID := range aw.GetStringSlice("author") {
		if authorID == artistId {
			belongsToArtist = true
			break
		}
	}
	if !belongsToArtist {
		return utils.NotFoundError(c)
	}

	// Generate the expected slug for the artwork
	expectedArtworkSlug := utils.Slugify(aw.GetString("title")) + "-" + aw.GetString("id")

	expectedPageUrl := "/artists/" + expectedArtistSlug + "/" + expectedArtworkSlug

	// Parse and normalise the related-work basis from the query, falling back to
	// BY ARTIST for an absent or unknown value.
	basis := repositories.ParseRelatedWorkBasis(c.Request.URL.Query().Get("basis"))
	canonicalURL := relatedWorkURL(expectedPageUrl, basis)

	// Redirect to the correct URL if either slug is not correct
	if artistSlug != expectedArtistSlug || artworkSlug != expectedArtworkSlug {
		return c.Redirect(http.StatusMovedPermanently, canonicalURL)
	}

	var img dto.Image

	img.Id = aw.GetString("id")
	img.Title = aw.GetString("title")
	img.Comment = aw.GetString("comment")
	img.Technique = aw.GetString("technique")
	if aw.GetString("image") != "" {
		img.Image = url.GenerateArtworkImageURL(aw, url.DeliveryProfileArtworkRecordTourPage, "")
		img.Zoom = url.GenerateArtworkImageURL(aw, url.DeliveryProfileViewer, "")
		img.Thumb = img.Image
	} else {
		img.Image = utils.AssetUrl("/assets/images/no-image.png")
		img.Thumb = utils.AssetUrl("/assets/images/no-image.png")
		img.Zoom = img.Image
	}

	content := dto.Artwork{
		Id:        aw.GetString("id"),
		Title:     aw.GetString("title"),
		Comment:   aw.GetString("comment"),
		Technique: aw.GetString("technique"),
		Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
			ArtistName:   artist.GetString("name"),
			ArtistId:     artist.Id,
			ArtworkId:    aw.GetString("id"),
			ArtworkTitle: aw.GetString("title"),
		}),
		Image: img,
		Artist: dto.Artist{
			Id:              artist.GetString("id"),
			FilingName:      artist.GetString("filing_name"),
			ShortName:       artist.GetString("short_name"),
			Name:            artist.GetString("filing_name"),
			Bio:             artist.GetString("bio"),
			Profession:      artist.GetString("profession"),
			ShowBreadcrumbs: true,
			Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
				ArtistId:   artist.GetString("id"),
				ArtistName: artist.GetString("name"),
			}),
		},
		ShowBreadcrumbs: true,
		ReproFile:       artworkReproductionFile(aw),
		SourceURL:       url.GenerateArtworkSourceURL(aw),
	}
	populateArtworkMetadata(app, aw, &content)
	populateArtworkCitation(&content)
	populateArtworkSourceData(app, aw, &content, environment)
	populateArtworkRelated(app, aw, &content, basis, expectedPageUrl)

	school := artist.GetStringSlice("school")

	var schoolCollector []string

	for _, s := range school {
		r, err := app.FindRecordById(constants.CollectionSchools, s)

		if err != nil {
			app.Logger().Error("school not found", "error", err.Error())
			continue
		}

		schoolCollector = append(schoolCollector, r.GetString("name"))

		content.Schools = strings.Join(schoolCollector, ", ")

	}

	// Annotate the source-backed commentary with glossary terms.
	glossaryEntries, glossaryErr := glossary.GetGlossaryEntries(app)
	if glossaryErr != nil {
		app.Logger().Warn("Failed to load glossary entries", "error", glossaryErr)
	} else {
		content.SourceComment = glossary.AnnotateHTML(content.SourceComment, glossaryEntries)
	}

	jsonLd := jsonld.ArtworkJsonLd(aw, artist)

	marshalled, err := json.Marshal(jsonLd)

	if err != nil {
		app.Logger().Error("Error marshalling artwork jsonld for"+aw.GetString("id"), "error", err.Error())
	}

	content.Jsonld = fmt.Sprintf(`<script type="application/ld+json">%s</script>`, marshalled)

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", content.Title, content.FilingName))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, aw.GetString("comment"))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, utils.AssetUrl(canonicalURL))
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgImageKey, utils.AssetUrl(content.Image.Image))

	c.Response.Header().Set("HX-Push-Url", canonicalURL)

	var buff bytes.Buffer

	err = pages.ArtworkPage(content).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artwork page", "error", err.Error())
		return c.String(http.StatusInternalServerError, "failed to render response template")
	}

	return c.HTML(http.StatusOK, buff.String())
}

func RenderArtworkContent(app *pocketbase.PocketBase, c *core.RequestEvent, artwork *core.Record, hxTarget string, showBreadcrumbs bool) (dto.Artwork, error) {

	authorIDs := artwork.GetStringSlice("author")
	artistId := ""
	if len(authorIDs) > 0 {
		artistId = authorIDs[0]
	}

	var artworkUrl string
	var img dto.Image

	img.Id = artwork.GetString("id")
	img.Title = artwork.GetString("title")
	img.Comment = artwork.GetString("comment")
	img.Technique = artwork.GetString("technique")
	if artwork.GetString("image") != "" {
		img.Image = url.GenerateArtworkImageURL(artwork, url.DeliveryProfileArtworkRecordTourPage, "")
		img.Zoom = url.GenerateArtworkImageURL(artwork, url.DeliveryProfileViewer, "")
	} else {
		img.Image = utils.AssetUrl("/assets/images/no-image.png")
		img.Zoom = img.Image
	}

	content := dto.Artwork{
		Id:              artwork.GetString("id"),
		Title:           artwork.GetString("title"),
		Comment:         artwork.GetString("comment"),
		Technique:       artwork.GetString("technique"),
		Image:           img,
		HxTarget:        hxTarget,
		ShowBreadcrumbs: showBreadcrumbs,
	}
	populateArtworkMetadata(app, artwork, &content)

	if artistId != "" {
		var artist *core.Record

		artist, err := app.FindRecordById(constants.CollectionArtists, artistId)

		if err != nil {
			app.Logger().Error(fmt.Sprintf("Error finding artist (%s) related to artwork (%s)", artistId, artwork.Id), "error", err.Error())
			return dto.Artwork{}, err
		}

		artworkUrl = url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
			ArtistName:   artist.GetString("name"),
			ArtistId:     artist.GetString("id"),
			ArtworkId:    artwork.GetString("id"),
			ArtworkTitle: artwork.GetString("title"),
		})

		content.Artist = dto.Artist{
			Id:              artist.GetString("id"),
			FilingName:      artist.GetString("filing_name"),
			ShortName:       artist.GetString("short_name"),
			Name:            artist.GetString("filing_name"),
			Bio:             artist.GetString("bio"),
			Profession:      artist.GetString("profession"),
			ShowBreadcrumbs: showBreadcrumbs,
			Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
				ArtistId:   artist.GetString("id"),
				ArtistName: artist.GetString("name"),
			}),
		}

	} else {
		artworkUrl = url.GenerateArtworkUrl(url.ArtworkUrlDTO{
			ArtworkId:    artwork.GetString("id"),
			ArtworkTitle: artwork.GetString("title"),
		})
	}

	// Annotate comment with glossary terms
	glossaryEntries, glossaryErr := glossary.GetGlossaryEntries(app)
	if glossaryErr != nil {
		app.Logger().Warn("Failed to load glossary entries", "error", glossaryErr)
	} else {
		content.Comment = glossary.AnnotateHTML(content.Comment, glossaryEntries)
	}

	// Set the URL for the artwork
	content.Url = artworkUrl

	return content, nil
}

func populateArtworkMetadata(app *pocketbase.PocketBase, artwork *core.Record, content *dto.Artwork) {
	content.Location, content.Dimensions = artworkLocationAndDimensions(artwork.GetString("comment"))
	if content.Dimensions != "" {
		content.Technique = strings.TrimSpace(strings.TrimSuffix(content.Technique, ", "+content.Dimensions))
	}

	if dateEnd := artwork.GetInt("date_end"); dateEnd > 0 {
		content.Year = strconv.Itoa(dateEnd)
	} else if year := artwork.GetInt("year"); year > 0 {
		content.Year = strconv.Itoa(year)
	}

	for _, typeID := range artwork.GetStringSlice("type") {
		artType, err := app.FindRecordById(constants.CollectionArtTypes, typeID)
		if err == nil {
			content.ArtType = artType.GetString("name")
			return
		}
	}
}

func artworkLocationAndDimensions(comment string) (string, string) {
	parts := strings.Split(tmplUtils.StripHtmlTags(comment), " · ")
	if len(parts) < 3 {
		return "", ""
	}

	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
}

func populateArtworkCitation(artwork *dto.Artwork) {
	artwork.CitationKey = "wga-" + artwork.Id
	artwork.CitationTitle = fmt.Sprintf("%s by %s", artwork.Title, artwork.Artist.FilingName)
	artwork.CitationURL = utils.AssetUrl(artwork.Url)
}

// artworkReproductionFile builds the truthful reproduction-file summary from
// independently available source dimensions, image format, and file weight.
// Absent facts are omitted; it returns an empty string only when no supported
// evidence exists, so the artwork record never fabricates a file caption.
func artworkReproductionFile(artwork *core.Record) string {
	width := artwork.GetInt("image_width")
	height := artwork.GetInt("image_height")
	format := artworkImageFormat(artwork.GetString("image"))

	if width <= 0 || height <= 0 {
		return format
	}

	summary := fmt.Sprintf("%d × %d px", width, height)
	if format != "" {
		summary += " · " + format
	}

	return summary
}

// artworkImageFormat maps a stored image filename extension to its display
// format, or returns "" for unknown extensions.
func artworkImageFormat(filename string) string {
	switch strings.ToLower(path.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "JPEG"
	case ".png":
		return "PNG"
	default:
		return ""
	}
}

// populateArtworkSourceData fills the file weight, palette, commentary, and
// music fields of the artwork DTO from the persisted record. Every value is
// source-backed; absent values remain empty so presentation never invents
// content. No reproduction source or licence claim is ever populated: the
// records hold no such field, and the plate must not claim provenance the
// archive cannot back. The environment parameter is retained only because
// callers (including the release acceptance suite) pass it positionally.
func populateArtworkSourceData(app *pocketbase.PocketBase, artwork *core.Record, content *dto.Artwork, _ config.Environment) {
	content.OriginalFileBytes = artwork.GetInt("image_size_bytes")
	if content.OriginalFileBytes > 0 {
		if content.ReproFile == "" {
			content.ReproFile = formatFileSize(content.OriginalFileBytes)
		} else {
			content.ReproFile += " · " + formatFileSize(content.OriginalFileBytes)
		}
	}
	content.SourceComment = artworkCommentaryHTML(artwork.GetString("source_comment"))
	content.HasCommentary = artwork.GetString("source_comment") != ""
	content.Palette = parseArtworkPalette(artwork)
	content.Music = buildArtworkMusic(app, artwork)
}

// formatFileSize renders a byte count in decimal SI units with one decimal
// place, matching the accepted reference FILE presentation (e.g. "1.4 MB").
// The exact byte count remains available in OriginalFileBytes.
func formatFileSize(bytes int) string {
	const (
		kilobyte = 1_000
		megabyte = 1_000_000
		gigabyte = 1_000_000_000
	)

	switch {
	case bytes >= gigabyte:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gigabyte))
	case bytes >= megabyte:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(megabyte))
	case bytes >= kilobyte:
		return fmt.Sprintf("%.1f kB", float64(bytes)/float64(kilobyte))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// artworkCommentaryHTML converts the raw source commentary to safe display HTML:
// escaped text split into paragraphs on blank lines. The glossary annotation is
// applied separately, after this conversion.
func artworkCommentaryHTML(sourceComment string) string {
	if sourceComment == "" {
		return ""
	}

	escaped := html.EscapeString(sourceComment)
	paragraphs := strings.Split(escaped, "\n\n")
	for i, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(strings.ReplaceAll(paragraph, "\n", "<br/>"))
		paragraphs[i] = "<p>" + paragraph + "</p>"
	}

	return strings.Join(paragraphs, "")
}

// populateArtworkRelated resolves the active related-work basis and fills the
// related-work images and basis controls into the artwork DTO.
func populateArtworkRelated(app *pocketbase.PocketBase, artwork *core.Record, content *dto.Artwork, basis repositories.RelatedWorkBasis, baseURL string) {
	result, err := repositories.NewRelatedWorkResolver(app).Resolve(artwork, basis)
	if err != nil {
		app.Logger().Warn("Failed to resolve related artworks", "artwork_id", artwork.Id, "error", err)
	}

	content.RelatedWorks = buildRelatedWorkImages(app, result.Basis, result.Works, content.HxTarget)
	content.Related = buildRelatedWorkState(result.Basis, result.Holding, content, artwork, baseURL)
}

// buildRelatedWorkImages maps resolver records onto the related-work card
// projection using the source-eligible related-card profile and the canonical
// per-work artwork URL.
func buildRelatedWorkImages(app *pocketbase.PocketBase, basis repositories.RelatedWorkBasis, works []*core.Record, hxTarget string) dto.ImageGrid {
	related := dto.ImageGrid{}
	for _, work := range works {
		image := utils.AssetUrl("/assets/images/no-image.png")
		if imageName := work.GetString("image"); imageName != "" {
			image = url.GenerateArtworkImageURL(work, url.DeliveryProfileRelatedTimelineCard, "")
		}

		routeName, filingName, shortName, artistID := resolveWorkArtist(app, work)
		workURL := url.GenerateArtworkUrl(url.ArtworkUrlDTO{
			ArtworkTitle: work.GetString("title"),
			ArtworkId:    work.Id,
		})
		if routeName != "" {
			workURL = url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   routeName,
				ArtistId:     artistID,
				ArtworkTitle: work.GetString("title"),
				ArtworkId:    work.Id,
			})
		}

		metadata := filingName
		if basis == repositories.RelatedByArtist {
			metadata = artworkDateMetadata(work)
		}

		related = append(related, dto.Image{
			Id:        work.Id,
			Title:     work.GetString("title"),
			Image:     image,
			Technique: work.GetString("technique"),
			Metadata:  metadata,
			Url:       workURL,
			Artist: dto.Artist{
				FilingName: filingName,
				ShortName:  shortName,
				Name:       filingName,
			},
			HxTarget: hxTarget,
		})
	}

	return related
}

// resolveWorkArtist returns the first published author's route, filing, short name and id for
// a work, or empty values when the work has no published author. Unpublished
// authors are skipped so a related card never leaks unpublished artist data or
// links to an unpublished artist record.
func resolveWorkArtist(app *pocketbase.PocketBase, work *core.Record) (string, string, string, string) {
	authorIDs := work.GetStringSlice("author")
	repo := repositories.NewArtistRecordRepository(app)
	for _, authorID := range authorIDs {
		artist, err := repo.FindPublishedArtist(authorID)
		if err != nil {
			continue
		}
		return artist.GetString("name"), artist.GetString("filing_name"), artist.GetString("short_name"), artist.Id
	}

	return "", "", "", ""
}

// parseArtworkPalette reads the compact image-derived palette from the persisted
// JSON field. Malformed or absent values yield an empty palette.
func parseArtworkPalette(artwork *core.Record) []dto.ColourSwatch {
	data, err := json.Marshal(artwork.Get("colour_palette"))
	if err != nil {
		return nil
	}

	var entries []struct {
		Hex    string `json:"hex"`
		Weight int    `json:"weight"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	swatches := make([]dto.ColourSwatch, 0, len(entries))
	for _, entry := range entries {
		if entry.Hex == "" {
			continue
		}
		swatches = append(swatches, dto.ColourSwatch{Hex: entry.Hex, Weight: entry.Weight})
	}

	return swatches
}

// buildArtworkMusic derives the deterministic period-music card from the
// artwork's known creation date. An unknown date or missing published match
// yields an unavailable card.
func buildArtworkMusic(app *pocketbase.PocketBase, artwork *core.Record) dto.MusicPeriod {
	song, err := repositories.NewArtistRecordRepository(app).MatchPeriodSong(artwork.GetInt("date_start"))
	if err != nil {
		app.Logger().Warn("Failed to match period music", "artwork_id", artwork.Id, "error", err)
		return dto.MusicPeriod{}
	}
	if song == nil || song.Record == nil {
		return dto.MusicPeriod{}
	}

	piece := strings.TrimSpace(song.Record.GetString("title"))
	if piece == "" || song.Record.GetString("source") == "" {
		return dto.MusicPeriod{}
	}

	return dto.MusicPeriod{
		Available: true,
		SongID:    song.Record.Id,
		Piece:     piece,
		Composer:  song.Composer,
		PlayerURL: "/player?song=" + song.Record.Id,
	}
}

// buildRelatedWorkState assembles the basis controls, connection heading,
// sparse-result explanation, and counted holding link for the active basis.
func buildRelatedWorkState(basis repositories.RelatedWorkBasis, holding *repositories.RelatedWorkHolding, content *dto.Artwork, artwork *core.Record, baseURL string) dto.RelatedWorkState {
	state := dto.RelatedWorkState{
		ActiveBasis: string(basis),
		Connection:  relatedConnection(basis, content.Artist.ShortName, artwork.GetInt("date_start")),
		Sparse:      len(content.RelatedWorks) < relatedWorksLimit,
		Bases:       relatedWorkBases(baseURL, basis),
	}
	if state.Sparse {
		state.SparseNote = relatedSparseNote(basis)
		state.Alternative, state.AlternativeURL = relatedAlternative(basis, baseURL)
	}
	// The holding count includes the current record, so it must exceed the
	// rendered sample by more than one before the link adds anything beyond
	// the current work itself.
	if holding != nil && holding.Count > len(content.RelatedWorks)+1 {
		if u := relatedHoldingURL(holding); u != "" {
			state.Holding = &dto.RelatedWorkHoldingLink{
				Label: relatedHoldingLabel(holding.Count),
				URL:   u,
			}
		}
	}

	return state
}

// relatedHoldingURL builds the artwork-search holding link from the resolver's
// filterable holding. Only the artist, venue, and period filter keys are
// accepted; any other key yields no link so a holding can never produce an
// arbitrary query string.
func relatedHoldingURL(holding *repositories.RelatedWorkHolding) string {
	switch holding.QueryKey {
	case "artist", "venue", "period":
		values := neturl.Values{}
		values.Set(holding.QueryKey, holding.QueryValue)
		return "/artworks?" + values.Encode()
	default:
		return ""
	}
}

// relatedHoldingLabel returns the counted holding link label. The count is the
// artwork-search total for the filter (including the current record), matching
// the search result total exactly.
func relatedHoldingLabel(count int) string {
	return fmt.Sprintf("FIND MORE %d IN THE ARTWORK SEARCH →", count)
}

// relatedWorkLimit is the resolver's bounded result cap, used to detect sparse
// results (fewer than the cap).
const relatedWorksLimit = 4

// relatedWorkURL appends the basis query for a non-default basis so the URL
// restores the active basis; the default basis keeps the plain canonical URL.
func relatedWorkURL(baseURL string, basis repositories.RelatedWorkBasis) string {
	if basis.IsDefault() {
		return baseURL
	}

	return baseURL + "?basis=" + string(basis)
}

// relatedWorkBases returns the four basis controls with labels and canonical
// URLs, marking the active one.
func relatedWorkBases(baseURL string, active repositories.RelatedWorkBasis) []dto.RelatedWorkBasis {
	order := []struct {
		value repositories.RelatedWorkBasis
		label string
	}{
		{repositories.RelatedByArtist, "BY ARTIST"},
		{repositories.RelatedByCollection, "SAME COLLECTION"},
		{repositories.RelatedByPeriod, "SAME PERIOD"},
		{repositories.RelatedByPalette, "SIMILAR PALETTE"},
	}

	bases := make([]dto.RelatedWorkBasis, 0, len(order))
	for _, basis := range order {
		bases = append(bases, dto.RelatedWorkBasis{
			Value:  string(basis.value),
			Label:  basis.label,
			URL:    relatedWorkURL(baseURL, basis.value),
			Active: basis.value == active,
		})
	}

	return bases
}

// relatedConnection returns the truthful section heading for the active basis.
func relatedConnection(basis repositories.RelatedWorkBasis, artistName string, dateStart int) string {
	switch basis {
	case repositories.RelatedByCollection:
		return "SAME COLLECTION"
	case repositories.RelatedByPalette:
		return "WORKS WITH A SIMILAR PALETTE"
	case repositories.RelatedByPeriod:
		if dateStart > 0 {
			return fmt.Sprintf("ARTISTS WORKING %d–%d", dateStart-relatedPeriodWindow, dateStart+relatedPeriodWindow)
		}
		return "ARTISTS FROM THE SAME PERIOD"
	default:
		return "OTHER WORKS BY " + strings.ToUpper(artistName)
	}
}

// relatedPeriodWindow matches the resolver's forty-year SAME PERIOD window.
const relatedPeriodWindow = 40

// relatedSparseNote returns the honest sparse-result explanation for the active
// basis.
func relatedSparseNote(basis repositories.RelatedWorkBasis) string {
	switch basis {
	case repositories.RelatedByCollection:
		return "The archive catalogues no further works from this collection."
	case repositories.RelatedByPalette:
		return "The archive holds no comparable published colour profile."
	case repositories.RelatedByPeriod:
		return "No other artist is catalogued within forty years of this work."
	default:
		return "The archive catalogues no further works by this artist."
	}
}

// relatedAlternative returns the label and URL of the basis most likely to
// supply records when the active basis is sparse.
func relatedAlternative(basis repositories.RelatedWorkBasis, baseURL string) (string, string) {
	if basis == repositories.RelatedByPalette {
		return "BY ARTIST", relatedWorkURL(baseURL, repositories.RelatedByArtist)
	}

	return "SIMILAR PALETTE", relatedWorkURL(baseURL, repositories.RelatedByPalette)
}
