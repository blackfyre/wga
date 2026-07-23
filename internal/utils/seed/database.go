package seed

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
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
	{name: constants.CollectionArtists, fields: []string{
		"name", "slug", "bio", "year_of_birth", "year_of_death", "place_of_birth", "place_of_death",
		"exact_year_of_birth", "exact_year_of_death", "profession", "known_place_of_birth",
		"known_place_of_death", "school", "published",
	}},
	{name: constants.CollectionArtworks, fields: []string{
		"title", "author", "form", "type", "technique", "school", "comment", "published", "image",
	}},
	{name: "glossary", fields: []string{"expression", "definition"}},
	{name: constants.CollectionGuestbook, fields: []string{"name", "email", "location", "message"}},
	{name: "music_composer", fields: []string{"name", "century", "language"}},
	{name: "music_song", fields: []string{"title", "composer", "source"}},
	{name: constants.CollectionStrings, fields: []string{"name", "content"}},
	{name: constants.CollectionStaticPages, fields: []string{"title", "slug", "content"}},
}

func ImportEmbedded(app core.App) error {
	paths, err := embeddedSourcePaths()
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
	if err := loadSourceFiles(paths.storage, &data); err != nil {
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
	if err := importSyntheticArtists(app, data); err != nil {
		return err
	}
	if err := importSyntheticArtworks(app, data); err != nil {
		return err
	}
	if err := importSyntheticGlossary(app, data.glossaryEntries); err != nil {
		return err
	}
	if err := importSyntheticGuestbook(app, data.guestbookEntries); err != nil {
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

	if err := validateFileFieldSizes(collections[constants.CollectionArtworks], "image", data.artworkFiles); err != nil {
		return err
	}

	return validateFileFieldSizes(collections["music_song"], "source", data.musicFiles)
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
		record, err := newRecord(app, constants.CollectionArtists, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.DisplayName)
		record.Set("slug", utils.Slugify(item.DisplayName))
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
		record.Set("published", true)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save artist %q: %w", item.ID, err)
		}
	}

	return nil
}

func importSyntheticArtworks(app core.App, data sourceData) error {
	for _, item := range data.artworks {
		file, ok := data.artworkFiles[item.ID]
		if !ok {
			return fmt.Errorf("artwork %q has no source file", item.ID)
		}
		image, err := filesystem.NewFileFromBytes(file.content, file.name)
		if err != nil {
			return fmt.Errorf("create artwork %q image: %w", item.ID, err)
		}

		record, err := newRecord(app, constants.CollectionArtworks, item.ID)
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
		record.Set("image", image)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save artwork %q: %w", item.ID, err)
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

func importSyntheticGuestbook(app core.App, items []sourceGuestbookEntry) error {
	for _, item := range items {
		record, err := newRecord(app, constants.CollectionGuestbook, item.ID)
		if err != nil {
			return err
		}

		record.Set("name", item.Name)
		record.Set("email", item.Email)
		record.Set("location", item.Location)
		record.Set("message", item.Message)
		if err := app.Save(record); err != nil {
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
		source, err := filesystem.NewFileFromBytes(file.content, file.name)
		if err != nil {
			return fmt.Errorf("create music track %q source: %w", track.ID, err)
		}

		record, err := newRecord(app, "music_song", track.ID)
		if err != nil {
			return err
		}
		record.Set("title", track.Title)
		record.Set("composer", []string{composerIDs[track.Composer]})
		record.Set("source", source)

		if err := app.Save(record); err != nil {
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

func centuryForPeriod(period string) (string, error) {
	switch period {
	case "Baroque":
		return "18", nil
	case "Romantic":
		return "19", nil
	default:
		return "", fmt.Errorf("unsupported music period %q", period)
	}
}
