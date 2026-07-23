package seed

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/resources/synthetic"
	_ "modernc.org/sqlite"
)

type SourceOptions struct {
	Environment config.Environment
	SQLitePath  string
	StoragePath string
	ReplaceMinimal bool
}

type sourcePaths struct {
	sqlitePath string
	storage    iofs.FS
	cleanup    func() error
}

type sourceData struct {
	schools            []sourceTaxonomy
	forms              []sourceTaxonomy
	types              []sourceTaxonomy
	professions        []sourceTaxonomy
	artists            []sourceArtist
	artistSchools      map[string][]string
	artistProfessions  map[string][]string
	biographies        []sourceBiography
	biographyLinks     []sourceBiographyLink
	artworks           []sourceArtwork
	glossaryEntries    []sourceGlossaryEntry
	guestbookEntries   []sourceGuestbookEntry
	musicTracks        []sourceMusicTrack
	sourceAttributions []sourceAttribution
	strings            []sourceString
	staticPages        []sourceStaticPage
	artworkFiles       map[string]string
	musicFiles         map[string]string
}

type sourceTaxonomy struct {
	ID   string
	Name string
}

type sourceArtist struct {
	ID                string
	SourcePath        string
	SourceHash        string
	DebugHash         string
	SourceDisplayName string
	DisplayName       string
	ArtistURL         string
	BirthYear         int
	DeathYear         int
	BirthPlace        string
	DeathPlace        string
	ActivityText      string
	ActivityStartYear int
	ActivityEndYear   int
	ArtistIndexPath   string
	BiographyImagePath string
}

type sourceBiography struct {
	ID               string
	ArtistID         string
	RawLifeDetail    string
	RawBiographyHTML string
	BiographyHTML    string
	BiographyText    string
}

type sourceBiographyLink struct {
	ID          string
	BiographyID string
	LinkType    string
	TargetPath  string
	LinkText    string
	LinkOrder   int
}

type sourceArtwork struct {
	ID                       string
	AuthorID                 string
	Title                    string
	DateText                 string
	DateStart                int
	DateEnd                  int
	IsCirca                  bool
	DateQualifier            string
	Technique                string
	TechniqueWithoutDimensions string
	Dimensions               string
	Location                 string
	URL                      string
	ImagePath                string
	OutputImagePath          string
	SchoolID                 string
	FormID                   string
	TypeID                   string
	SourceRow                int
}

type sourceGlossaryEntry struct {
	ID         string
	Term       string
	Definition string
	Anchor     string
	SourcePage string
	SortOrder  int
}

type sourceGuestbookEntry struct {
	ID       string
	Name     string
	Email    string
	Location string
	Message  string
	Created  string
	Updated  string
}

type sourceMusicTrack struct {
	ID          string
	TrackOrder  int
	Title       string
	Period      string
	Composer    string
	Origin      string
	PlaybackURL string
	MediaFormat string
	PlayerURL   string
	LocalPath   string
	Part        int
	PartCount   int
}

type sourceAttribution struct {
	ID               string
	AttributionOrder int
	Category         string
	Subcategory      string
	Title            string
	Citation         string
	SourceURL        string
}

type sourceString struct {
	ID      string
	Name    string
	Content string
}

type sourceStaticPage struct {
	ID      string
	Title   string
	Slug    string
	Content string
}

func resolveSourcePaths(options SourceOptions) (sourcePaths, error) {
	if options.SQLitePath == "" {
		if options.Environment != config.EnvironmentDevelopment && options.Environment != config.EnvironmentStaging {
			return sourcePaths{}, fmt.Errorf("WGA_SEED_SQLITE_PATH is required in %q", options.Environment)
		}

		return embeddedSourcePaths(options.StoragePath)
	}

	return externalSourcePaths(options.SQLitePath, options.StoragePath)
}

func embeddedSourcePaths(storagePath string) (sourcePaths, error) {
	sqlitePath, cleanup, err := materializeEmbeddedSQLite()
	if err != nil {
		return sourcePaths{}, err
	}

	storage, err := embeddedStorageFS(storagePath)
	if err != nil {
		_ = cleanup()
		return sourcePaths{}, err
	}

	return sourcePaths{
		sqlitePath: sqlitePath,
		storage:    storage,
		cleanup:    cleanup,
	}, nil
}

func externalSourcePaths(sqlitePath string, storagePath string) (sourcePaths, error) {

	absSQLitePath, err := filepath.Abs(sqlitePath)
	if err != nil {
		return sourcePaths{}, fmt.Errorf("resolve seed SQLite path: %w", err)
	}

	info, err := os.Stat(absSQLitePath)
	if err != nil {
		return sourcePaths{}, fmt.Errorf("seed SQLite path: %w", err)
	}
	if info.IsDir() {
		return sourcePaths{}, fmt.Errorf("seed SQLite path %q is a directory", absSQLitePath)
	}

	if storagePath == "" {
		storagePath = filepath.Join(filepath.Dir(absSQLitePath), "storage")
	}
	absStoragePath, err := filepath.Abs(storagePath)
	if err != nil {
		return sourcePaths{}, fmt.Errorf("resolve seed storage path: %w", err)
	}

	return sourcePaths{
		sqlitePath: absSQLitePath,
		storage:    os.DirFS(absStoragePath),
	}, nil
}


func materializeEmbeddedSQLite() (string, func() error, error) {
	data, err := synthetic.Files.ReadFile("wga-test.sqlite")
	if err != nil {
		return "", nil, fmt.Errorf("read embedded seed SQLite database: %w", err)
	}

	file, err := os.CreateTemp("", "wga-seed-*.sqlite")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary seed SQLite database: %w", err)
	}
	cleanup := func() error {
		return os.Remove(file.Name())
	}

	written, writeErr := file.Write(data)
	if writeErr != nil {
		_ = file.Close()
		_ = cleanup()
		return "", nil, fmt.Errorf("write temporary seed SQLite database: %w", writeErr)
	}
	if written != len(data) {
		_ = file.Close()
		_ = cleanup()
		return "", nil, io.ErrShortWrite
	}
	if err := file.Close(); err != nil {
		_ = cleanup()
		return "", nil, fmt.Errorf("close temporary seed SQLite database: %w", err)
	}

	return file.Name(), cleanup, nil
}

func embeddedStorageFS(storagePath string) (iofs.FS, error) {
	if storagePath == "" {
		storage, err := iofs.Sub(synthetic.Files, "storage")
		if err != nil {
			return nil, fmt.Errorf("open embedded seed storage: %w", err)
		}

		return storage, nil
	}

	absStoragePath, err := filepath.Abs(storagePath)
	if err != nil {
		return nil, fmt.Errorf("resolve seed storage path: %w", err)
	}

	return os.DirFS(absStoragePath), nil
}

func (paths sourcePaths) Close() error {
	if paths.cleanup == nil {
		return nil
	}

	return paths.cleanup()
}

func loadSourceData(paths sourcePaths) (sourceData, error) {
	connectionURL := (&url.URL{
		Scheme:   "file",
		Path:     paths.sqlitePath,
		RawQuery: "mode=ro",
	}).String()

	db, err := sql.Open("sqlite", connectionURL)
	if err != nil {
		return sourceData{}, fmt.Errorf("open seed SQLite database: %w", err)
	}
	defer closeDatabase(db)

	data := sourceData{
		artistSchools:     map[string][]string{},
		artistProfessions: map[string][]string{},
		artworkFiles:      map[string]string{},
		musicFiles:        map[string]string{},
	}

	if data.schools, err = loadTaxonomy(db, "schools"); err != nil {
		return sourceData{}, err
	}
	if data.forms, err = loadTaxonomy(db, "forms"); err != nil {
		return sourceData{}, err
	}
	if data.types, err = loadTaxonomy(db, "types"); err != nil {
		return sourceData{}, err
	}
	if data.professions, err = loadTaxonomy(db, "professions"); err != nil {
		return sourceData{}, err
	}
	if data.artists, err = loadArtists(db); err != nil {
		return sourceData{}, err
	}
	if data.artistSchools, err = loadArtistRelations(db, "artist_schools", "school_id"); err != nil {
		return sourceData{}, err
	}
	if data.artistProfessions, err = loadArtistRelations(db, "artist_professions", "profession_id"); err != nil {
		return sourceData{}, err
	}
	if data.biographies, err = loadBiographies(db); err != nil {
		return sourceData{}, err
	}
	if data.biographyLinks, err = loadBiographyLinks(db); err != nil {
		return sourceData{}, err
	}
	if data.artworks, err = loadArtworks(db); err != nil {
		return sourceData{}, err
	}
	if data.glossaryEntries, err = loadGlossaryEntries(db); err != nil {
		return sourceData{}, err
	}
	if data.guestbookEntries, err = loadGuestbookEntries(db); err != nil {
		return sourceData{}, err
	}
	if data.musicTracks, err = loadMusicTracks(db); err != nil {
		return sourceData{}, err
	}
	if data.sourceAttributions, err = loadSourceAttributions(db); err != nil {
		return sourceData{}, err
	}
	if data.strings, err = loadStrings(db); err != nil {
		return sourceData{}, err
	}
	if data.staticPages, err = loadStaticPages(db); err != nil {
		return sourceData{}, err
	}

	if err := validateSourceRelations(data); err != nil {
		return sourceData{}, err
	}
	return data, nil
}

func loadTaxonomy(db *sql.DB, table string) ([]sourceTaxonomy, error) {
	rows, err := db.Query("SELECT id, name FROM " + table + " ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", table, err)
	}
	defer closeRows(rows)

	items := []sourceTaxonomy{}
	for rows.Next() {
		item := sourceTaxonomy{}
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadArtists(db *sql.DB) ([]sourceArtist, error) {
	rows, err := db.Query(`
		SELECT id, source_path, source_hash, debug_hash, source_display_name,
			display_name, artist_url, COALESCE(birth_year, 0), COALESCE(death_year, 0),
			COALESCE(birth_place, ''), COALESCE(death_place, ''), COALESCE(activity_text, ''),
			COALESCE(activity_start_year, 0), COALESCE(activity_end_year, 0),
			COALESCE(artist_index_path, ''), COALESCE(biography_image_path, '')
		FROM artists
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read artists: %w", err)
	}
	defer closeRows(rows)

	items := []sourceArtist{}
	for rows.Next() {
		item := sourceArtist{}
		if err := rows.Scan(
			&item.ID,
			&item.SourcePath,
			&item.SourceHash,
			&item.DebugHash,
			&item.SourceDisplayName,
			&item.DisplayName,
			&item.ArtistURL,
			&item.BirthYear,
			&item.DeathYear,
			&item.BirthPlace,
			&item.DeathPlace,
			&item.ActivityText,
			&item.ActivityStartYear,
			&item.ActivityEndYear,
			&item.ArtistIndexPath,
			&item.BiographyImagePath,
		); err != nil {
			return nil, fmt.Errorf("scan artists: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadArtistRelations(db *sql.DB, table string, relationColumn string) (map[string][]string, error) {
	rows, err := db.Query("SELECT artist_id, " + relationColumn + " FROM " + table + " ORDER BY artist_id, " + relationColumn)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", table, err)
	}
	defer closeRows(rows)

	relations := map[string][]string{}
	for rows.Next() {
		var artistID string
		var relationID string
		if err := rows.Scan(&artistID, &relationID); err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		relations[artistID] = append(relations[artistID], relationID)
	}

	return relations, rows.Err()
}

func loadBiographies(db *sql.DB) ([]sourceBiography, error) {
	rows, err := db.Query(`
		SELECT id, artist_id, raw_life_detail, raw_biography_html, biography_html, biography_text
		FROM biographies
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read biographies: %w", err)
	}
	defer closeRows(rows)

	items := []sourceBiography{}
	for rows.Next() {
		item := sourceBiography{}
		if err := rows.Scan(&item.ID, &item.ArtistID, &item.RawLifeDetail, &item.RawBiographyHTML, &item.BiographyHTML, &item.BiographyText); err != nil {
			return nil, fmt.Errorf("scan biographies: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadBiographyLinks(db *sql.DB) ([]sourceBiographyLink, error) {
	rows, err := db.Query(`
		SELECT id, biography_id, link_type, target_path, link_text, link_order
		FROM biography_links
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read biography links: %w", err)
	}
	defer closeRows(rows)

	items := []sourceBiographyLink{}
	for rows.Next() {
		item := sourceBiographyLink{}
		if err := rows.Scan(&item.ID, &item.BiographyID, &item.LinkType, &item.TargetPath, &item.LinkText, &item.LinkOrder); err != nil {
			return nil, fmt.Errorf("scan biography links: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadArtworks(db *sql.DB) ([]sourceArtwork, error) {
	rows, err := db.Query(`
		SELECT id, author_id, title, date_text, COALESCE(date_start, 0), COALESCE(date_end, 0),
			is_circa, COALESCE(date_qualifier, ''), technique,
			COALESCE(technique_without_dimensions, ''), COALESCE(dimensions, ''), location, url,
			COALESCE(image_path, ''), COALESCE(output_image_path, ''), COALESCE(school_id, ''), form_id,
			COALESCE(type_id, ''), source_row
		FROM artworks
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read artworks: %w", err)
	}
	defer closeRows(rows)

	items := []sourceArtwork{}
	for rows.Next() {
		item := sourceArtwork{}
		var isCirca int
		if err := rows.Scan(
			&item.ID,
			&item.AuthorID,
			&item.Title,
			&item.DateText,
			&item.DateStart,
			&item.DateEnd,
			&isCirca,
			&item.DateQualifier,
			&item.Technique,
			&item.TechniqueWithoutDimensions,
			&item.Dimensions,
			&item.Location,
			&item.URL,
			&item.ImagePath,
			&item.OutputImagePath,
			&item.SchoolID,
			&item.FormID,
			&item.TypeID,
			&item.SourceRow,
		); err != nil {
			return nil, fmt.Errorf("scan artworks: %w", err)
		}
		item.IsCirca = isCirca != 0
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadGlossaryEntries(db *sql.DB) ([]sourceGlossaryEntry, error) {
	rows, err := db.Query(`
		SELECT id, term, definition, COALESCE(anchor, ''), source_page, sort_order
		FROM glossary_entries
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read glossary entries: %w", err)
	}
	defer closeRows(rows)

	items := []sourceGlossaryEntry{}
	for rows.Next() {
		item := sourceGlossaryEntry{}
		if err := rows.Scan(&item.ID, &item.Term, &item.Definition, &item.Anchor, &item.SourcePage, &item.SortOrder); err != nil {
			return nil, fmt.Errorf("scan glossary entries: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadGuestbookEntries(db *sql.DB) ([]sourceGuestbookEntry, error) {
	rows, err := db.Query(`
		SELECT id, name, email, location, message, created, updated
		FROM guestbook_entries
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read guestbook entries: %w", err)
	}
	defer closeRows(rows)

	items := []sourceGuestbookEntry{}
	for rows.Next() {
		item := sourceGuestbookEntry{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Location, &item.Message, &item.Created, &item.Updated); err != nil {
			return nil, fmt.Errorf("scan guestbook entries: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadMusicTracks(db *sql.DB) ([]sourceMusicTrack, error) {
	rows, err := db.Query(`
		SELECT id, track_order, title, period, composer, COALESCE(origin, ''), playback_url,
			media_format, player_url, COALESCE(local_path, ''), part, part_count
		FROM music_tracks
		ORDER BY track_order
	`)
	if err != nil {
		return nil, fmt.Errorf("read music tracks: %w", err)
	}
	defer closeRows(rows)

	items := []sourceMusicTrack{}
	for rows.Next() {
		item := sourceMusicTrack{}
		if err := rows.Scan(
			&item.ID,
			&item.TrackOrder,
			&item.Title,
			&item.Period,
			&item.Composer,
			&item.Origin,
			&item.PlaybackURL,
			&item.MediaFormat,
			&item.PlayerURL,
			&item.LocalPath,
			&item.Part,
			&item.PartCount,
		); err != nil {
			return nil, fmt.Errorf("scan music tracks: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadSourceAttributions(db *sql.DB) ([]sourceAttribution, error) {
	rows, err := db.Query(`
		SELECT id, attribution_order, category, COALESCE(subcategory, ''), title, citation, source_url
		FROM source_attributions
		ORDER BY attribution_order
	`)
	if err != nil {
		return nil, fmt.Errorf("read source attributions: %w", err)
	}
	defer closeRows(rows)

	items := []sourceAttribution{}
	for rows.Next() {
		item := sourceAttribution{}
		if err := rows.Scan(&item.ID, &item.AttributionOrder, &item.Category, &item.Subcategory, &item.Title, &item.Citation, &item.SourceURL); err != nil {
			return nil, fmt.Errorf("scan source attributions: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadStrings(db *sql.DB) ([]sourceString, error) {
	rows, err := db.Query("SELECT id, name, content FROM strings ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read strings: %w", err)
	}
	defer closeRows(rows)

	items := []sourceString{}
	for rows.Next() {
		item := sourceString{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Content); err != nil {
			return nil, fmt.Errorf("scan strings: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadStaticPages(db *sql.DB) ([]sourceStaticPage, error) {
	rows, err := db.Query("SELECT id, title, slug, content FROM static_pages ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read static pages: %w", err)
	}
	defer closeRows(rows)

	items := []sourceStaticPage{}
	for rows.Next() {
		item := sourceStaticPage{}
		if err := rows.Scan(&item.ID, &item.Title, &item.Slug, &item.Content); err != nil {
			return nil, fmt.Errorf("scan static pages: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func validateSourceRelations(data sourceData) error {
	artistIDs := makeIDSet(data.artists, func(item sourceArtist) string { return item.ID })
	schoolIDs := makeIDSet(data.schools, func(item sourceTaxonomy) string { return item.ID })
	formIDs := makeIDSet(data.forms, func(item sourceTaxonomy) string { return item.ID })
	typeIDs := makeIDSet(data.types, func(item sourceTaxonomy) string { return item.ID })
	professionIDs := makeIDSet(data.professions, func(item sourceTaxonomy) string { return item.ID })
	biographyIDs := makeIDSet(data.biographies, func(item sourceBiography) string { return item.ID })

	for artistID, schools := range data.artistSchools {
		if _, ok := artistIDs[artistID]; !ok {
			return fmt.Errorf("artist school references unknown artist %q", artistID)
		}
		for _, schoolID := range schools {
			if _, ok := schoolIDs[schoolID]; !ok {
				return fmt.Errorf("artist school references unknown school %q", schoolID)
			}
		}
	}

	for artistID, professions := range data.artistProfessions {
		if _, ok := artistIDs[artistID]; !ok {
			return fmt.Errorf("artist profession references unknown artist %q", artistID)
		}
		for _, professionID := range professions {
			if _, ok := professionIDs[professionID]; !ok {
				return fmt.Errorf("artist profession references unknown profession %q", professionID)
			}
		}
	}

	for _, biography := range data.biographies {
		if _, ok := artistIDs[biography.ArtistID]; !ok {
			return fmt.Errorf("biography %q references unknown artist %q", biography.ID, biography.ArtistID)
		}
	}

	for _, link := range data.biographyLinks {
		if _, ok := biographyIDs[link.BiographyID]; !ok {
			return fmt.Errorf("biography link %q references unknown biography %q", link.ID, link.BiographyID)
		}
	}

	for _, artwork := range data.artworks {
		if _, ok := artistIDs[artwork.AuthorID]; !ok {
			return fmt.Errorf("artwork %q references unknown artist %q", artwork.ID, artwork.AuthorID)
		}
		if _, ok := schoolIDs[artwork.SchoolID]; !ok {
			return fmt.Errorf("artwork %q references unknown school %q", artwork.ID, artwork.SchoolID)
		}
		if _, ok := formIDs[artwork.FormID]; !ok {
			return fmt.Errorf("artwork %q references unknown form %q", artwork.ID, artwork.FormID)
		}
		if _, ok := typeIDs[artwork.TypeID]; !ok {
			return fmt.Errorf("artwork %q references unknown type %q", artwork.ID, artwork.TypeID)
		}
	}

	return nil
}

func makeIDSet[T any](items []T, id func(T) string) map[string]struct{} {
	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		ids[id(item)] = struct{}{}
	}
	return ids
}

func loadSourceFiles(storage iofs.FS, data *sourceData) error {
	for _, artwork := range data.artworks {
		filename, err := singleSourceFile(storage, path.Join("Artworks", artwork.ID))
		if err != nil {
			return fmt.Errorf("artwork %q storage: %w", artwork.ID, err)
		}
		data.artworkFiles[artwork.ID] = filename
	}

	for _, track := range data.musicTracks {
		relativePath, err := safeRelativePath(track.LocalPath)
		if err != nil {
			return fmt.Errorf("music track %q storage path: %w", track.ID, err)
		}
		info, err := iofs.Stat(storage, relativePath)
		if err != nil {
			return fmt.Errorf("music track %q storage file: %w", track.ID, err)
		}
		if info.IsDir() {
			return fmt.Errorf("music track %q storage file %q is a directory", track.ID, relativePath)
		}
		data.musicFiles[track.ID] = path.Base(relativePath)
	}

	return nil
}

func singleSourceFile(storage iofs.FS, dir string) (string, error) {
	entries, err := iofs.ReadDir(storage, dir)
	if err != nil {
		return "", err
	}

	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	if len(files) != 1 {
		return "", fmt.Errorf("expected one file, found %d", len(files))
	}

	return files[0], nil
}

func safeRelativePath(value string) (string, error) {
	if value == "" {
		return "", errors.New("is empty")
	}

	cleaned := path.Clean(value)
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("must stay within the storage root: %q", value)
	}

	return cleaned, nil
}

func sourceFilePath(directory string, recordID string, filename string) string {
	return path.Join(directory, recordID, filename)
}

func sourceMusicFilePath(track sourceMusicTrack) (string, error) {
	relativePath, err := safeRelativePath(track.LocalPath)
	if err != nil {
		return "", err
	}

	return relativePath, nil
}

func sortedValues(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)

	return copyValues
}

func closeDatabase(db *sql.DB) {
	_ = db.Close()
}

func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}
