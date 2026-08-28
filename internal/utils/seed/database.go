package seed

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

var ErrApplicationRecords = errors.New("synthetic bootstrap requires an empty application database")

type targetCollection struct {
	name   string
	fields []string
}

var syntheticTargetCollections = []targetCollection{
	{name: "schools", fields: []string{"name", "slug"}},
	{name: "art_forms", fields: []string{"name", "slug"}},
	{name: "art_types", fields: []string{"name", "slug"}},
	{name: "art_periods", fields: []string{"name", "slug", "start", "end", "description"}},
	{name: constants.CollectionLocations, fields: []string{"name", "city", "country", "museum", "is_public"}},
	{name: constants.CollectionArtists, fields: []string{
		"name", "slug", "bio", "year_of_birth", "year_of_death", "place_of_birth", "place_of_death",
		"exact_year_of_birth", "exact_year_of_death", "profession", "known_place_of_birth",
		"known_place_of_death", "school", "portrait", "biography_image_width", "biography_image_height", "published",
		"filing_name", "short_name",
	}},
	{name: constants.CollectionArtworks, fields: []string{
		"title", "author", "form", "type", "technique", "school", "comment", "published", "image", "image_width", "image_height",
		"source_row", "date_start", "date_end", "is_circa", "date_qualifier", "timeframe_text",
		"current_location_id", "art_period_id",
		"source_url", "source_path", "source_comment", "colour_palette", "colour_signature", "colour_profile_version", "colour_image_hash",
		"image_size_bytes",
	}},
	{name: constants.CollectionSelections, fields: []string{
		"artist", "title", "context", "display_title", "commentary", "artworks", "source_path", "source_hash", "content_hash", "published",
	}},
	{name: "glossary", fields: []string{"expression", "definition"}},
	{name: constants.CollectionGuestbook, fields: []string{"name", "email", "location", "message"}},
	{name: "music_composer", fields: []string{"name", "century", "language"}},
	{name: "music_song", fields: []string{"title", "composer", "source"}},
	{name: constants.CollectionStrings, fields: []string{"name", "content"}},
	{name: constants.CollectionStaticPages, fields: []string{"title", "slug", "content"}},
}

func Import(app core.App, sqlitePath string) error {
	paths, err := sourcePathsFor(sqlitePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = paths.Close()
	}()
	var storage *filesystem.System
	if paths.preseededAssets && app.Settings().S3.Enabled {
		storage, err = app.NewFilesystem()
		if err != nil {
			return fmt.Errorf("open configured seed storage: %w", err)
		}
		defer func() {
			_ = storage.Close()
		}()
	}

	data, err := loadSourceData(paths)
	if err != nil {
		return err
	}
	if err := loadSourceFiles(paths, storage, &data); err != nil {
		return err
	}
	if err := validateTargetCollections(app, data); err != nil {
		return err
	}
	if err := RequireEmptyApplicationDatabase(app); err != nil {
		return err
	}

	if err := importSyntheticTaxonomy(app, "schools", data.schools, true); err != nil {
		return err
	}
	if err := importSyntheticTaxonomy(app, "art_forms", data.forms, true); err != nil {
		return err
	}
	if err := importSyntheticTaxonomy(app, "art_types", data.types, true); err != nil {
		return err
	}
	if err := importSyntheticArtPeriods(app, data.artPeriods); err != nil {
		return err
	}
	if err := importSyntheticLocations(app, data.locations); err != nil {
		return err
	}
	if err := importSyntheticArtists(app, data, paths.preseededAssets); err != nil {
		return err
	}
	if err := importSyntheticArtworks(app, data, paths.preseededAssets); err != nil {
		return err
	}
	if err := importSyntheticSelections(app, data.selections); err != nil {
		return err
	}
	if err := importSyntheticGlossary(app, data.glossaryEntries); err != nil {
		return err
	}
	if err := importSyntheticGuestbook(app, data.guestbookEntries, paths.preseededAssets); err != nil {
		return err
	}
	if err := importSyntheticMusic(app, data); err != nil {
		return err
	}
	if err := importSyntheticStrings(app, data.strings); err != nil {
		return err
	}

	return importSyntheticStaticPages(app, data.staticPages)
}

func validateTargetCollections(app core.App, data sourceData) error {
	collections := map[string]*core.Collection{}
	for _, target := range syntheticTargetCollections {
		collection, err := app.FindCollectionByNameOrId(target.name)
		if err != nil {
			return fmt.Errorf("find target collection %q: %w", target.name, err)
		}
		for _, fieldName := range target.fields {
			if collection.Fields.GetByName(fieldName) == nil {
				return fmt.Errorf("target collection %q is missing field %q", target.name, fieldName)
			}
		}
		collections[target.name] = collection
	}

	if err := validateSourceFieldContracts(collections[constants.CollectionArtworks], collections[constants.CollectionSelections]); err != nil {
		return err
	}

	if err := validateFileFieldSizes(collections[constants.CollectionArtworks], "image", data.artworkFiles); err != nil {
		return err
	}

	return validateFileFieldSizes(collections["music_song"], "source", data.musicFiles)
}

// validateSourceFieldContracts fails closed unless the source-imported artwork
// and selection fields carry the concrete PocketBase types, relation targets,
// and cardinality the importer writes. Artworks are saved through
// SaveNoValidate for preseeded sources, so a schema mismatch here would corrupt
// the import silently; it is rejected before any record is saved. Selection
// relation cardinality is enforced because the selection read-model depends on
// the single-artist and multi-artwork shapes.
func validateSourceFieldContracts(artworks *core.Collection, selections *core.Collection) error {
	for _, check := range []struct {
		name string
		typ  string
	}{
		{"source_row", core.FieldTypeNumber},
		{"date_start", core.FieldTypeNumber},
		{"date_end", core.FieldTypeNumber},
		{"is_circa", core.FieldTypeBool},
		{"date_qualifier", core.FieldTypeText},
		{"timeframe_text", core.FieldTypeText},
		{"source_url", core.FieldTypeText},
		{"source_path", core.FieldTypeText},
		{"source_comment", core.FieldTypeText},
		{"colour_palette", core.FieldTypeJSON},
		{"colour_signature", core.FieldTypeJSON},
		{"colour_profile_version", core.FieldTypeText},
		{"colour_image_hash", core.FieldTypeText},
		{"image_size_bytes", core.FieldTypeNumber},
	} {
		field := artworks.Fields.GetByName(check.name)
		if field == nil {
			return fmt.Errorf("artworks field %q is missing", check.name)
		}
		if field.Type() != check.typ {
			return fmt.Errorf("artworks field %q has type %q, want %q", check.name, field.Type(), check.typ)
		}
	}

	if err := validateRelationContract(artworks, "current_location_id", "locations", 0, 1); err != nil {
		return err
	}
	if err := validateRelationContract(artworks, "art_period_id", "art_periods", 0, 1); err != nil {
		return err
	}
	if err := validateRelationContract(selections, "artist", "artists", 1, 1); err != nil {
		return err
	}
	if err := validateRelationContract(selections, "artworks", "artworks", 1, 1000); err != nil {
		return err
	}

	return nil
}

func validateRelationContract(collection *core.Collection, name string, wantCollection string, wantMin int, wantMax int) error {
	field := collection.Fields.GetByName(name)
	relation, ok := field.(*core.RelationField)
	if !ok {
		return fmt.Errorf("collection %q field %q is not a relation field", collection.Id, name)
	}
	if relation.CollectionId != wantCollection {
		return fmt.Errorf("collection %q field %q targets %q, want %q", collection.Id, name, relation.CollectionId, wantCollection)
	}
	if relation.MinSelect != wantMin || relation.MaxSelect != wantMax {
		return fmt.Errorf("collection %q field %q cardinality [%d,%d], want [%d,%d]", collection.Id, name, relation.MinSelect, relation.MaxSelect, wantMin, wantMax)
	}

	return nil
}

func validateFileFieldSizes(collection *core.Collection, fieldName string, files map[string]sourceFile) error {
	field, ok := collection.Fields.GetByName(fieldName).(*core.FileField)
	if !ok {
		return fmt.Errorf("target collection %q field %q is not a file field", collection.Id, fieldName)
	}

	maxSize := field.MaxSize
	if maxSize == 0 {
		maxSize = 5 * 1024 * 1024
	}
	for id, file := range files {
		if file.preseededAssets {
			continue
		}
		if int64(len(file.content)) > maxSize {
			return fmt.Errorf("source file %q for %s/%s is %d bytes, exceeding the %d-byte field limit", id, collection.Id, fieldName, len(file.content), maxSize)
		}
	}

	return nil
}

func RequireEmptyApplicationDatabase(app core.App) error {
	collections, err := app.FindAllCollections()
	if err != nil {
		return err
	}

	for _, collection := range collections {
		if collection.System {
			continue
		}

		records, err := app.FindRecordsByFilter(collection.Id, "", "", 1, 0)
		if err != nil {
			return err
		}
		if len(records) > 0 {
			return ErrApplicationRecords
		}
	}

	return nil
}

func importSyntheticTaxonomy(app core.App, collectionName string, items []sourceTaxonomy, withSlug bool) error {
	for _, item := range items {
		record, err := newRecord(app, collectionName, item.ID)
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

func importSyntheticArtPeriods(app core.App, items []sourceArtPeriod) error {
	for _, item := range items {
		record, err := newRecord(app, "art_periods", item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		record.Set("slug", utils.Slugify(item.Name))
		record.Set("start", item.Start)
		record.Set("end", item.End)
		record.Set("description", item.Description)
		if err := app.Save(record); err != nil {
			return fmt.Errorf("save art period %q: %w", item.ID, err)
		}
	}

	return nil
}

// importSyntheticLocations records the producer locations taxonomy. City and
// country are optional and remain unset when the producer leaves them empty.
func importSyntheticLocations(app core.App, items []sourceLocation) error {
	for _, item := range items {
		record, err := newRecord(app, constants.CollectionLocations, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		if item.City != "" {
			record.Set("city", item.City)
		}
		if item.Country != "" {
			record.Set("country", item.Country)
		}
		record.Set("museum", item.Museum)
		record.Set("is_public", item.IsPublic)
		if err := app.Save(record); err != nil {
			return fmt.Errorf("save location %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticArtists(app core.App, data sourceData, preseededAssets bool) error {
	professionNames := make(map[string]string, len(data.professions))
	for _, profession := range data.professions {
		professionNames[profession.ID] = profession.Name
	}

	biographies := make(map[string]sourceBiography, len(data.biographies))
	for _, biography := range data.biographies {
		biographies[biography.ArtistID] = biography
	}
	usedSlugs := map[string]struct{}{}

	for _, item := range data.artists {
		record, err := newRecord(app, constants.CollectionArtists, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.DisplayName)
		record.Set("filing_name", item.DisplayName)
		record.Set("short_name", item.ShortName)
		record.Set("slug", uniqueArtistSlug(usedSlugs, item.DisplayName, item.ID))
		record.Set("bio", biographies[item.ID].BiographyHTML)
		record.Set("year_of_birth", item.BirthYear)
		record.Set("year_of_death", item.DeathYear)
		record.Set("place_of_birth", item.BirthPlace)
		record.Set("place_of_death", item.DeathPlace)
		record.Set("exact_year_of_birth", item.BirthYear != 0)
		record.Set("exact_year_of_death", item.DeathYear != 0)
		record.Set("known_place_of_birth", knownPlace(item.BirthPlace))
		record.Set("known_place_of_death", knownPlace(item.DeathPlace))
		record.Set("school", data.artistSchools[item.ID])
		record.Set("profession", joinedProfessionNames(data.artistProfessions[item.ID], professionNames))
		record.Set("biography_image_width", item.BiographyImageWidth)
		record.Set("biography_image_height", item.BiographyImageHeight)
		if item.Portrait != "" {
			record.Set("portrait", item.Portrait)
		}
		record.Set("published", true)

		if err := saveSeedRecord(app, record, preseededAssets && item.Portrait != ""); err != nil {
			return fmt.Errorf("save artist %q: %w", item.ID, err)
		}
	}

	return nil
}

func uniqueArtistSlug(used map[string]struct{}, name string, id string) string {
	slug := utils.Slugify(name)
	if _, exists := used[slug]; exists {
		slug += "-" + id
	}
	used[slug] = struct{}{}

	return slug
}

func importSyntheticArtworks(app core.App, data sourceData, preseededAssets bool) error {
	for _, item := range data.artworks {
		file, hasFile := data.artworkFiles[item.ID]
		if item.ImagePath != "" && !hasFile {
			return fmt.Errorf("artwork %q declares image %q but has no source file", item.ID, item.ImagePath)
		}
		record, err := newRecord(app, constants.CollectionArtworks, item.ID)
		if err != nil {
			return err
		}

		record.Set("title", item.Title)
		record.Set("author", []string{item.AuthorID})
		record.Set("form", []string{item.FormID})
		if item.TypeID != "" {
			record.Set("type", []string{item.TypeID})
		}
		if item.SchoolID != "" {
			record.Set("school", []string{item.SchoolID})
		}
		record.Set("technique", item.Technique)
		record.Set("comment", artworkComment(item))
		record.Set("published", true)
		record.Set("image_width", item.ImageWidth)
		record.Set("image_height", item.ImageHeight)
		record.Set("source_row", item.SourceRow)
		record.Set("date_start", item.DateStart)
		record.Set("date_end", item.DateEnd)
		record.Set("is_circa", item.IsCirca)
		if item.DateQualifier != "" {
			record.Set("date_qualifier", item.DateQualifier)
		}
		if item.TimeframeText != "" {
			record.Set("timeframe_text", item.TimeframeText)
		}
		if item.CurrentLocationID != "" {
			record.Set("current_location_id", []string{item.CurrentLocationID})
		}
		if item.ArtPeriodID != "" {
			record.Set("art_period_id", []string{item.ArtPeriodID})
		}
		if item.SourceURL != "" {
			record.Set("source_url", item.SourceURL)
		}
		if item.SourcePath != "" {
			record.Set("source_path", item.SourcePath)
		}
		if item.SourceComment != "" {
			record.Set("source_comment", item.SourceComment)
		}
		if len(item.ColourPalette) > 0 {
			record.Set("colour_palette", item.ColourPalette)
		}
		if item.ColourSignature != nil {
			record.Set("colour_signature", *item.ColourSignature)
		}
		if item.ColourProfileVersion != "" {
			record.Set("colour_profile_version", item.ColourProfileVersion)
		}
		if item.ColourImageHash != "" {
			record.Set("colour_image_hash", item.ColourImageHash)
		}
		if item.ImagePath == "" {
			// image-less artwork: leave the image file field unset
		} else {
			record.Set("image_size_bytes", file.size)
			if file.preseededAssets {
				record.Set("image", file.name)
			} else {
				image, err := filesystem.NewFileFromBytes(file.content, file.name)
				if err != nil {
					return fmt.Errorf("create artwork %q image: %w", item.ID, err)
				}
				record.Set("image", image)
			}
		}

		if err := saveSeedRecord(app, record, preseededAssets); err != nil {
			return fmt.Errorf("save artwork %q: %w", item.ID, err)
		}
	}

	return nil
}

// importSyntheticSelections records the flat producer selection contract,
// preserving the ordered artwork membership and the supplied commentary bytes.
func importSyntheticSelections(app core.App, items []sourceSelection) error {
	for _, item := range items {
		record, err := newRecord(app, constants.CollectionSelections, item.ID)
		if err != nil {
			return err
		}

		record.Set("artist", []string{item.ArtistID})
		record.Set("title", item.Title)
		record.Set("context", item.Context)
		record.Set("display_title", item.DisplayTitle)
		record.Set("commentary", item.Commentary)
		record.Set("artworks", item.ArtworkIDs)
		record.Set("source_path", item.SourcePath)
		record.Set("source_hash", item.SourceHash)
		record.Set("content_hash", item.ContentHash)
		record.Set("published", item.Published)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save selection %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticGlossary(app core.App, items []sourceGlossaryEntry) error {
	for _, item := range items {
		record, err := newRecord(app, "glossary", item.ID)
		if err != nil {
			return err
		}

		record.Set("expression", item.Term)
		record.Set("definition", item.Definition)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save glossary entry %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticGuestbook(app core.App, items []sourceGuestbookEntry, preseededAssets bool) error {
	for _, item := range items {
		record, err := newRecord(app, constants.CollectionGuestbook, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		record.Set("email", item.Email)
		record.Set("location", item.Location)
		record.Set("message", item.Message)
		if err := saveSeedRecord(app, record, preseededAssets); err != nil {
			return fmt.Errorf("save guestbook entry %q: %w", item.ID, err)
		}
		if _, err := app.DB().Update(constants.CollectionGuestbook, dbx.Params{
			"created": item.Created,
			"updated": item.Updated,
		}, dbx.HashExp{"id": record.Id}).Execute(); err != nil {
			return fmt.Errorf("preserve guestbook entry %q timestamps: %w", item.ID, err)
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

		century, err := centuryForPeriod(track.Period)
		if err != nil {
			return err
		}
		composerID := syntheticID("music-composer:" + track.Composer)
		record, err := newRecord(app, "music_composer", composerID)
		if err != nil {
			return err
		}
		record.Set("name", track.Composer)
		record.Set("century", century)
		record.Set("language", "")
		if err := app.Save(record); err != nil {
			return fmt.Errorf("save music composer %q: %w", track.Composer, err)
		}
		composerIDs[track.Composer] = composerID
	}

	for _, track := range data.musicTracks {
		file, ok := data.musicFiles[track.ID]
		if !ok {
			return fmt.Errorf("music track %q has no source file", track.ID)
		}
		record, err := newRecord(app, "music_song", track.ID)
		if err != nil {
			return err
		}
		record.Set("title", track.Title)
		record.Set("composer", []string{composerIDs[track.Composer]})
		if file.preseededAssets {
			record.Set("source", file.name)
		} else {
			source, err := filesystem.NewFileFromBytes(file.content, file.name)
			if err != nil {
				return fmt.Errorf("create music track %q source: %w", track.ID, err)
			}
			record.Set("source", source)
		}

		if err := saveSeedRecord(app, record, file.preseededAssets); err != nil {
			return fmt.Errorf("save music track %q: %w", track.ID, err)
		}
	}

	return nil
}

func importSyntheticStrings(app core.App, items []sourceString) error {
	for _, item := range items {
		record, err := newRecord(app, constants.CollectionStrings, item.ID)
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
		record, err := newRecord(app, constants.CollectionStaticPages, item.ID)
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

func newRecord(app core.App, collectionName string, id string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("id", id)

	return record, nil
}

func saveSeedRecord(app core.App, record *core.Record, preseededAssets bool) error {
	if preseededAssets {
		return app.SaveNoValidate(record)
	}

	return app.Save(record)
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
	if artwork.Comment != "" {
		return artwork.Comment
	}

	// No enriched commentary: retain the truthful source/metadata summary
	// (date, location, dimensions) rather than inventing prose.
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

func centuryForPeriod(period string) (string, error) {
	parts := strings.Fields(period)
	if len(parts) == 2 && parts[1] == "century" {
		ordinal := parts[0]
		if len(ordinal) > 2 {
			century, err := strconv.Atoi(ordinal[:len(ordinal)-2])
			if err == nil && century > 0 {
				return strconv.Itoa(century), nil
			}
		}
	}

	switch period {
	case "Baroque":
		return "18", nil
	case "Romantic", "Romanticism":
		return "19", nil
	default:
		return "", fmt.Errorf("unsupported music period %q", period)
	}
}
