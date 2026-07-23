package seed

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const (
	syntheticSeedMarkerID      = "syntheticseedv1"
	syntheticSeedMarkerName    = "synthetic_seed_v1"
	syntheticSeedMarkerContent = "Synthetic source database v1"
	knownMinimalArtistBio      = "<p>Mara Example is a fictional artist included as starter content for local testing.</p>"
	knownMinimalArtworkComment = "<p>A fictional landscape included as starter content for testing search and comparison.</p>"
	knownMinimalGlossary        = "A composition whose main subject is natural scenery."
	knownMinimalPrivacyContent  = "<p>Web Gallery of Art respects visitor privacy. This starter database contains fictional data for testing only; configure the production privacy policy before collecting visitor information.</p>"
	knownMinimalWelcomeContent  = `<p>
The Web Gallery of Art is a searchable collection of European painting, sculpture, decorative arts, and architecture from the 3rd century to the early 20th century. It began as a Renaissance-focused project and grew into a broader archive built for students, teachers, and curious visitors who want images and context in the same place.
</p>
<p>
This version of the site is organized around a few strong routes. Start with the <a href="/artists">artists</a> index for biographies and related works, move into <a href="/artworks">artwork search</a> when you know what you want to filter, open <a href="/dual-mode" target="_blank">Dual Mode</a> to compare two pages side by side, or use <a href="/inspire">Inspiration</a> when you want the collection to surprise you.
</p>
<p>
The project remains an independent public resource: open, interconnected, and designed for study as much as enjoyment. If the collection helps you, leave a note in the <a href="/guestbook">guestbook</a> or explore how the site is maintained by its <a href="/contributors">contributors</a>.
</p>`
)

func SeedDatabase(app core.App, options SourceOptions) error {
	paths, err := resolveSourcePaths(options)
	if err != nil {
		return err
	}
	defer func() {
		_ = paths.Close()
	}()

	data, err := loadSourceData(paths)
	if err != nil {
		return err
	}

	return app.RunInTransaction(func(txApp core.App) error {
		alreadySeeded, err := prepareSeedTarget(txApp, options.ReplaceMinimal)
		if err != nil {
			return err
		}
		if alreadySeeded {
			return nil
		}
		if err := importSyntheticTaxonomy(txApp, "schools", data.schools, true); err != nil {
			return err
		}
		if err := importSyntheticTaxonomy(txApp, "art_forms", data.forms, true); err != nil {
			return err
		}
		if err := importSyntheticTaxonomy(txApp, "art_types", data.types, true); err != nil {
			return err
		}
		if err := importSyntheticTaxonomy(txApp, "professions", data.professions, false); err != nil {
			return err
		}
		if err := importSyntheticArtists(txApp, data); err != nil {
			return err
		}
		if err := importSyntheticBiographies(txApp, data.biographies); err != nil {
			return err
		}
		if err := importSyntheticBiographyLinks(txApp, data.biographyLinks); err != nil {
			return err
		}
		if err := importSyntheticArtworks(txApp, data); err != nil {
			return err
		}
		if err := importSyntheticGlossary(txApp, data.glossaryEntries); err != nil {
			return err
		}
		if err := importSyntheticGuestbook(txApp, data.guestbookEntries); err != nil {
			return err
		}
		if err := importSyntheticMusic(txApp, data); err != nil {
			return err
		}
		if err := importSyntheticAttributions(txApp, data.sourceAttributions); err != nil {
			return err
		}
		if err := importSyntheticStrings(txApp, data.strings); err != nil {
			return err
		}

		if err := importSyntheticStaticPages(txApp, data.staticPages); err != nil {
			return err
		}

		return recordSyntheticSeedMarker(txApp)
	})
}

func prepareSeedTarget(app core.App, replaceMinimal bool) (bool, error) {
	alreadySeeded, err := hasSyntheticSeedMarker(app)
	if err != nil {
		return false, err
	}
	if alreadySeeded {
		return true, nil
	}

	hasRecords, err := hasApplicationRecords(app)
	if err != nil {
		return false, err
	}
	if !hasRecords {
		return false, nil
	}
	if !replaceMinimal {
		return false, errors.New("seed target contains application records; use a fresh data directory or --replace-minimal")
	}

	isMinimal, err := isKnownMinimalSeedTarget(app)
	if err != nil {
		return false, err
	}
	if !isMinimal {
		return false, errors.New("seed target contains application records; use a fresh data directory")
	}

	return false, removeMinimalSeedRecords(app)
}

func hasSyntheticSeedMarker(app core.App) (bool, error) {
	record, err := app.FindRecordById(constants.CollectionStrings, syntheticSeedMarkerID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if record.GetString("name") != syntheticSeedMarkerName || record.GetString("content") != syntheticSeedMarkerContent {
		return false, errors.New("seed marker has unexpected content")
	}

	return true, nil
}

func recordSyntheticSeedMarker(app core.App) error {
	record, err := findOrNewRecord(app, constants.CollectionStrings, syntheticSeedMarkerID)
	if err != nil {
		return err
	}

	record.Set("name", syntheticSeedMarkerName)
	record.Set("content", syntheticSeedMarkerContent)

	return app.Save(record)
}

func removeMinimalSeedRecords(app core.App) error {

	artist, err := findRecord(app, constants.CollectionArtists, "slug", "mara-example")
	if err != nil {
		return err
	}
	if artist == nil || artist.GetString("name") != "Mara Example" {
		return nil
	}

	artworks, err := app.FindRecordsByFilter(constants.CollectionArtworks, "", "", 0, 0)
	if err != nil {
		return err
	}

	var demonstrationArtwork *core.Record
	for _, artwork := range artworks {
		if artwork.GetString("title") != "Cobalt Horizon" {
			continue
		}
		if contains(artwork.GetStringSlice("author"), artist.Id) {
			demonstrationArtwork = artwork
			break
		}
	}
	if demonstrationArtwork == nil {
		return nil
	}

	if err := app.Delete(demonstrationArtwork); err != nil {
		return err
	}
	if err := app.Delete(artist); err != nil {
		return err
	}

	for _, item := range []struct {
		collection string
		field      string
		value      string
	}{
		{collection: "art_forms", field: "slug", value: "painting"},
		{collection: "art_types", field: "slug", value: "landscape"},
		{collection: "schools", field: "slug", value: "american"},
		{collection: "glossary", field: "expression", value: "landscape"},
		{collection: "strings", field: "name", value: "welcome"},
		{collection: "static_pages", field: "slug", value: "privacy-policy"},
	} {
		record, err := findRecord(app, item.collection, item.field, item.value)
		if err != nil {
			return err
		}
		if record == nil {
			continue
		}
		if err := app.Delete(record); err != nil {
			return err
		}
	}

	return nil
}

func isMinimalSeedTarget(app core.App) (bool, error) {
	expectedCounts := map[string]int{
		"artists":             1,
		"art_forms":           1,
		"art_periods":         0,
		"art_types":           1,
		"artworks":            1,
		"biographies":         0,
		"biography_links":     0,
		"feedbacks":           0,
		"glossary":            1,
		"guestbook":           0,
		"music_composer":      0,
		"music_song":          0,
		"postcards":           0,
		"professions":         0,
		"schools":             1,
		"source_attributions": 0,
		"static_pages":        1,
		"strings":             1,
	}

	collections, err := app.FindAllCollections()
	if err != nil {
		return false, err
	}
	seen := map[string]bool{}
	for _, collection := range collections {
		if collection.System {
			continue
		}

		expected, known := expectedCounts[collection.Id]
		if known {
			seen[collection.Id] = true
		}
		records, err := app.FindRecordsByFilter(collection.Id, "", "", 0, 0)
		if err != nil {
			return false, err
		}
		if !known {
			if len(records) > 0 {
				return false, nil
			}
			continue
		}
		if len(records) != expected {
			return false, nil
		}
	}
	for collectionName := range expectedCounts {
		if !seen[collectionName] {
			return false, nil
		}
	}

	return true, nil
}

func isKnownMinimalSeedTarget(app core.App) (bool, error) {
	isMinimal, err := isMinimalSeedTarget(app)
	if err != nil || !isMinimal {
		return isMinimal, err
	}

	school, err := findRecord(app, "schools", "slug", "american")
	if err != nil {
		return false, err
	}
	form, err := findRecord(app, "art_forms", "slug", "painting")
	if err != nil {
		return false, err
	}
	artType, err := findRecord(app, "art_types", "slug", "landscape")
	if err != nil {
		return false, err
	}
	artist, err := findRecord(app, constants.CollectionArtists, "slug", "mara-example")
	if err != nil {
		return false, err
	}
	artwork, err := findRecord(app, constants.CollectionArtworks, "title", "Cobalt Horizon")
	if err != nil {
		return false, err
	}
	glossary, err := findRecord(app, "glossary", "expression", "landscape")
	if err != nil {
		return false, err
	}
	welcome, err := findRecord(app, "strings", "name", "welcome")
	if err != nil {
		return false, err
	}
	privacyPage, err := findRecord(app, "static_pages", "slug", "privacy-policy")
	if err != nil {
		return false, err
	}

	if school == nil || form == nil || artType == nil || artist == nil || artwork == nil || glossary == nil || welcome == nil || privacyPage == nil {
		return false, nil
	}
	if school.GetString("name") != "American" || form.GetString("name") != "Painting" || artType.GetString("name") != "Landscape" {
		return false, nil
	}
	if !matchesKnownMinimalArtist(artist, school.Id) {
		return false, nil
	}
	if !matchesKnownMinimalArtwork(artwork, artist.Id, school.Id, form.Id, artType.Id) {
		return false, nil
	}
	if !matchesKnownMinimalGlossary(glossary) || welcome.GetString("content") != knownMinimalWelcomeContent || privacyPage.GetString("title") != "Privacy Policy" || privacyPage.GetString("content") != knownMinimalPrivacyContent {
		return false, nil
	}

	return true, nil
}

func matchesKnownMinimalArtist(artist *core.Record, schoolID string) bool {
	if artist.GetString("name") != "Mara Example" || artist.GetString("bio") != knownMinimalArtistBio {
		return false
	}
	if artist.GetInt("year_of_birth") != 1850 || artist.GetInt("year_of_death") != 1910 {
		return false
	}
	if artist.GetString("place_of_birth") != "Sampleton" || artist.GetString("place_of_death") != "Sampleton" {
		return false
	}
	if artist.GetBool("exact_year_of_birth") || artist.GetBool("exact_year_of_death") {
		return false
	}
	if artist.GetString("profession") != "Painter" || artist.GetString("known_place_of_birth") != "n/a" || artist.GetString("known_place_of_death") != "n/a" {
		return false
	}
	if !artist.GetBool("published") || !hasOnlyID(artist.GetStringSlice("school"), schoolID) || len(artist.GetStringSlice("also_known_as")) != 0 || len(artist.GetStringSlice("professions")) != 0 {
		return false
	}

	for _, field := range []string{
		"source_path",
		"source_hash",
		"debug_hash",
		"source_display_name",
		"artist_url",
		"activity_text",
		"artist_index_path",
		"biography_image_path",
	} {
		if artist.GetString(field) != "" {
			return false
		}
	}

	return artist.GetInt("activity_start_year") == 0 && artist.GetInt("activity_end_year") == 0
}

func matchesKnownMinimalArtwork(artwork *core.Record, artistID string, schoolID string, formID string, typeID string) bool {
	if artwork.GetString("technique") != "Oil on canvas" || artwork.GetString("comment") != knownMinimalArtworkComment || artwork.GetString("image") != "" || !artwork.GetBool("published") {
		return false
	}
	if !hasOnlyID(artwork.GetStringSlice("author"), artistID) || !hasOnlyID(artwork.GetStringSlice("school"), schoolID) || !hasOnlyID(artwork.GetStringSlice("form"), formID) || !hasOnlyID(artwork.GetStringSlice("type"), typeID) {
		return false
	}

	for _, field := range []string{
		"date_text",
		"date_qualifier",
		"technique_without_dimensions",
		"dimensions",
		"location",
		"source_url",
		"source_image_path",
		"output_image_path",
	} {
		if artwork.GetString(field) != "" {
			return false
		}
	}

	return artwork.GetInt("date_start") == 0 && artwork.GetInt("date_end") == 0 && !artwork.GetBool("is_circa") && artwork.GetInt("source_row") == 0
}

func matchesKnownMinimalGlossary(glossary *core.Record) bool {
	return glossary.GetString("definition") == knownMinimalGlossary && glossary.GetString("anchor") == "" && glossary.GetString("source_page") == "" && glossary.GetInt("sort_order") == 0
}

func hasApplicationRecords(app core.App) (bool, error) {
	collections, err := app.FindAllCollections()
	if err != nil {
		return false, err
	}

	for _, collection := range collections {
		if collection.System {
			continue
		}

		records, err := app.FindRecordsByFilter(collection.Id, "", "", 1, 0)
		if err != nil {
			return false, err
		}
		if len(records) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func importSyntheticTaxonomy(app core.App, collectionName string, items []sourceTaxonomy, withSlug bool) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, collectionName, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		if withSlug {
			record.Set("slug", utils.Slugify(item.Name))
		}

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save %s %q: %w", collectionName, item.ID, err)
		}
	}

	return nil
}

func importSyntheticArtists(app core.App, data sourceData) error {
	professionNames := make(map[string]string, len(data.professions))
	for _, profession := range data.professions {
		professionNames[profession.ID] = profession.Name
	}

	biographies := make(map[string]sourceBiography, len(data.biographies))
	for _, biography := range data.biographies {
		biographies[biography.ArtistID] = biography
	}

	for _, item := range data.artists {
		record, err := findOrNewRecord(app, constants.CollectionArtists, item.ID)
		if err != nil {
			return err
		}

		biography := biographies[item.ID]
		record.Set("name", item.DisplayName)
		record.Set("slug", utils.Slugify(item.DisplayName))
		record.Set("bio", biography.BiographyHTML)
		record.Set("year_of_birth", item.BirthYear)
		record.Set("year_of_death", item.DeathYear)
		record.Set("place_of_birth", item.BirthPlace)
		record.Set("place_of_death", item.DeathPlace)
		record.Set("exact_year_of_birth", item.BirthYear != 0)
		record.Set("exact_year_of_death", item.DeathYear != 0)
		record.Set("known_place_of_birth", knownPlace(item.BirthPlace))
		record.Set("known_place_of_death", knownPlace(item.DeathPlace))
		record.Set("school", data.artistSchools[item.ID])
		record.Set("professions", data.artistProfessions[item.ID])
		record.Set("profession", joinedProfessionNames(data.artistProfessions[item.ID], professionNames))
		record.Set("published", true)
		record.Set("source_path", item.SourcePath)
		record.Set("source_hash", item.SourceHash)
		record.Set("debug_hash", item.DebugHash)
		record.Set("source_display_name", item.SourceDisplayName)
		record.Set("artist_url", item.ArtistURL)
		record.Set("activity_text", item.ActivityText)
		record.Set("activity_start_year", item.ActivityStartYear)
		record.Set("activity_end_year", item.ActivityEndYear)
		record.Set("artist_index_path", item.ArtistIndexPath)
		record.Set("biography_image_path", item.BiographyImagePath)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save artist %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticBiographies(app core.App, items []sourceBiography) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, "biographies", item.ID)
		if err != nil {
			return err
		}

		record.Set("artist", []string{item.ArtistID})
		record.Set("raw_life_detail", item.RawLifeDetail)
		record.Set("raw_biography_html", item.RawBiographyHTML)
		record.Set("biography_html", item.BiographyHTML)
		record.Set("biography_text", item.BiographyText)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save biography %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticBiographyLinks(app core.App, items []sourceBiographyLink) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, "biography_links", item.ID)
		if err != nil {
			return err
		}

		record.Set("biography", []string{item.BiographyID})
		record.Set("link_type", item.LinkType)
		record.Set("target_path", item.TargetPath)
		record.Set("link_text", item.LinkText)
		record.Set("link_order", item.LinkOrder)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save biography link %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticArtworks(app core.App, data sourceData) error {
	for _, item := range data.artworks {
		record, err := findOrNewRecord(app, constants.CollectionArtworks, item.ID)
		if err != nil {
			return err
		}

		record.Set("title", item.Title)
		record.Set("author", []string{item.AuthorID})
		record.Set("form", []string{item.FormID})
		record.Set("type", []string{item.TypeID})
		record.Set("school", []string{item.SchoolID})
		record.Set("technique", item.Technique)
		record.Set("comment", artworkComment(item))
		record.Set("published", true)
		if filename := data.artworkFiles[item.ID]; filename != "" {
			record.Set("image", filename)
		}
		record.Set("date_text", item.DateText)
		record.Set("date_start", item.DateStart)
		record.Set("date_end", item.DateEnd)
		record.Set("is_circa", item.IsCirca)
		record.Set("date_qualifier", item.DateQualifier)
		record.Set("technique_without_dimensions", item.TechniqueWithoutDimensions)
		record.Set("dimensions", item.Dimensions)
		record.Set("location", item.Location)
		record.Set("source_url", item.URL)
		record.Set("source_image_path", item.ImagePath)
		record.Set("output_image_path", item.OutputImagePath)
		record.Set("source_row", item.SourceRow)

		if err := app.SaveNoValidate(record); err != nil {
			return fmt.Errorf("save artwork %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticGlossary(app core.App, items []sourceGlossaryEntry) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, "glossary", item.ID)
		if err != nil {
			return err
		}

		record.Set("expression", item.Term)
		record.Set("definition", item.Definition)
		record.Set("anchor", item.Anchor)
		record.Set("source_page", item.SourcePage)
		record.Set("sort_order", item.SortOrder)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save glossary entry %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticGuestbook(app core.App, items []sourceGuestbookEntry) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, constants.CollectionGuestbook, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		record.Set("email", item.Email)
		record.Set("location", item.Location)
		record.Set("message", item.Message)
		record.Set("created", item.Created)
		record.Set("updated", item.Updated)

		if err := app.SaveNoValidate(record); err != nil {
			return fmt.Errorf("save guestbook entry %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticMusic(app core.App, data sourceData) error {
	composerIDs := map[string]string{}
	for _, track := range data.musicTracks {
		if _, ok := composerIDs[track.Composer]; ok {
			continue
		}

		composerID := syntheticID("music-composer:" + track.Composer)
		record, err := findOrNewRecord(app, "music_composer", composerID)
		if err != nil {
			return err
		}
		record.Set("name", track.Composer)
		record.Set("century", centuryForPeriod(track.Period))
		record.Set("language", "")
		if err := app.Save(record); err != nil {
			return fmt.Errorf("save music composer %q: %w", track.Composer, err)
		}
		composerIDs[track.Composer] = composerID
	}

	for _, track := range data.musicTracks {
		record, err := findOrNewRecord(app, "music_song", track.ID)
		if err != nil {
			return err
		}

		record.Set("title", track.Title)
		record.Set("composer", []string{composerIDs[track.Composer]})
		if filename := data.musicFiles[track.ID]; filename != "" {
			record.Set("source", filename)
		}
		record.Set("track_order", track.TrackOrder)
		record.Set("period", track.Period)
		record.Set("origin", track.Origin)
		record.Set("playback_url", track.PlaybackURL)
		record.Set("media_format", track.MediaFormat)
		record.Set("player_url", track.PlayerURL)
		record.Set("local_path", track.LocalPath)
		record.Set("part", track.Part)
		record.Set("part_count", track.PartCount)

		if err := app.SaveNoValidate(record); err != nil {
			return fmt.Errorf("save music track %q: %w", track.ID, err)
		}
	}

	return nil
}

func importSyntheticAttributions(app core.App, items []sourceAttribution) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, "source_attributions", item.ID)
		if err != nil {
			return err
		}

		record.Set("attribution_order", item.AttributionOrder)
		record.Set("category", item.Category)
		record.Set("subcategory", item.Subcategory)
		record.Set("title", item.Title)
		record.Set("citation", item.Citation)
		record.Set("source_url", item.SourceURL)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save source attribution %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticStrings(app core.App, items []sourceString) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, constants.CollectionStrings, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		record.Set("content", item.Content)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save string %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticStaticPages(app core.App, items []sourceStaticPage) error {
	for _, item := range items {
		record, err := findOrNewRecord(app, constants.CollectionStaticPages, item.ID)
		if err != nil {
			return err
		}

		record.Set("title", item.Title)
		record.Set("slug", item.Slug)
		record.Set("content", item.Content)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save static page %q: %w", item.ID, err)
		}
	}

	return nil
}

func findOrNewRecord(app core.App, collectionName string, id string) (*core.Record, error) {
	record, err := app.FindRecordById(collectionName, id)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, err
	}

	record = core.NewRecord(collection)
	record.Set("id", id)

	return record, nil
}

func findRecord(app core.App, collectionName string, field string, value string) (*core.Record, error) {
	record, err := app.FindFirstRecordByData(collectionName, field, value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return record, err
}

func knownPlace(value string) string {
	if value == "" {
		return "n/a"
	}

	return "yes"
}

func joinedProfessionNames(ids []string, names map[string]string) string {
	values := []string{}
	for _, id := range ids {
		name, ok := names[id]
		if ok {
			values = append(values, name)
		}
	}

	return strings.Join(sortedValues(values), ", ")
}

func artworkComment(artwork sourceArtwork) string {
	parts := []string{artwork.DateText, artwork.Location}
	if artwork.Dimensions != "" {
		parts = append(parts, artwork.Dimensions)
	}

	return "<p>" + html.EscapeString(strings.Join(parts, " · ")) + "</p>"
}

func syntheticID(value string) string {
	digest := sha256.Sum256([]byte(value))

	return fmt.Sprintf("%x", digest)[:15]
}

func centuryForPeriod(period string) string {
	switch period {
	case "Baroque":
		return "18"
	case "Romantic":
		return "19"
	default:
		return "20"
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}

func hasOnlyID(values []string, expected string) bool {
	return len(values) == 1 && values[0] == expected
}
