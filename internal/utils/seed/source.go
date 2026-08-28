package seed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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

	"github.com/blackfyre/wga/resources/synthetic"
	_ "modernc.org/sqlite"
)

type sourcePaths struct {
	sqlitePath      string
	storage         iofs.FS
	storageRoot     string
	preseededAssets bool
	cleanup         func() error
}

type sourceFile struct {
	name            string
	content         []byte
	size            int64
	preseededAssets bool
}

type sourceData struct {
	schools           []sourceTaxonomy
	forms             []sourceTaxonomy
	types             []sourceTaxonomy
	artPeriods        []sourceArtPeriod
	locations         []sourceLocation
	professions       []sourceTaxonomy
	artists           []sourceArtist
	artistSchools     map[string][]string
	artistProfessions map[string][]string
	biographies       []sourceBiography
	artworks          []sourceArtwork
	glossaryEntries   []sourceGlossaryEntry
	guestbookEntries  []sourceGuestbookEntry
	musicTracks       []sourceMusicTrack
	strings           []sourceString
	staticPages       []sourceStaticPage
	selections        []sourceSelection
	artworkFiles      map[string]sourceFile
	musicFiles        map[string]sourceFile
}

type sourceTaxonomy struct {
	ID   string
	Name string
}

type sourceArtPeriod struct {
	ID          string
	Name        string
	Start       int
	End         int
	Description string
}

// sourceLocation is one producer locations record. City and country are
// optional (the producer leaves them empty for unlocated private collections),
// while museum and is_public are the producer's boolean flags.
type sourceLocation struct {
	ID       string
	Name     string
	City     string
	Country  string
	Museum   bool
	IsPublic bool
}

// sourceArtist is one producer artists record. DisplayName carries the
// encyclopaedic filing form from source_display_name, while ShortName carries
// the supplied short form from display_name. Both are preserved verbatim; the
// importer never parses, reverses, or reconstructs either value.
type sourceArtist struct {
	ID                   string
	DisplayName          string
	ShortName            string
	BirthYear            int
	DeathYear            int
	BirthPlace           string
	DeathPlace           string
	Portrait             string
	BiographyImageWidth  int
	BiographyImageHeight int
}

type sourceBiography struct {
	ArtistID      string
	BiographyHTML string
}

type sourceArtwork struct {
	ID                   string
	AuthorID             string
	Title                string
	DateText             string
	Technique            string
	Dimensions           string
	Location             string
	SourceURL            string
	SourcePath           string
	ImagePath            string
	ImageWidth           int
	ImageHeight          int
	SchoolID             string
	FormID               string
	TypeID               string
	SourceRow            int
	DateStart            int
	DateEnd              int
	IsCirca              bool
	DateQualifier        string
	TimeframeText        string
	CurrentLocationID    string
	ArtPeriodID          string
	SourceComment        string
	Comment              string
	ColourPalette        []sourceColourPaletteEntry
	ColourSignature      *sourceColourSignature
	ColourProfileVersion string
	ColourImageHash      string
}

// sourceColourPaletteEntry is one hex/weight entry in the producer's
// image-derived colour palette. Weight is the entry's quantised share.
type sourceColourPaletteEntry struct {
	Hex    string `json:"hex"`
	Weight int    `json:"weight"`
}

// sourceColourSignature is the producer's quantised colour signature used for
// palette-distance ranking. Space names the profile space and Bins carries the
// weighted histogram.
type sourceColourSignature struct {
	Space string `json:"space"`
	Bins  []int  `json:"bins"`
}

type sourceGlossaryEntry struct {
	ID         string
	Term       string
	Definition string
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
	Title       string
	Period      string
	Composer    string
	LocalPath   string
	ArtPeriodID string
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

// sourceSelection is the flat producer artwork_selections record. ArtworkIDs
// retains the direct source order of the ordered artwork membership.
type sourceSelection struct {
	ID           string
	ArtistID     string
	Title        string
	Context      string
	DisplayTitle string
	Commentary   string
	ArtworkIDs   []string
	SourcePath   string
	SourceHash   string
	ContentHash  string
	Published    bool
}

func sourcePathsFor(sqlitePath string) (sourcePaths, error) {
	if sqlitePath == "" {
		return embeddedSourcePaths()
	}

	return externalSourcePaths(sqlitePath)
}

func externalSourcePaths(sqlitePath string) (sourcePaths, error) {
	info, err := os.Stat(sqlitePath)
	if err != nil {
		return sourcePaths{}, fmt.Errorf("stat seed SQLite database: %w", err)
	}
	if !info.Mode().IsRegular() {
		return sourcePaths{}, fmt.Errorf("seed SQLite database is not a regular file: %s", sqlitePath)
	}

	return sourcePaths{
		sqlitePath:      sqlitePath,
		storageRoot:     filepath.Join(filepath.Dir(sqlitePath), "storage"),
		preseededAssets: true,
	}, nil
}

func embeddedSourcePaths() (sourcePaths, error) {
	sqlitePath, cleanup, err := materializeEmbeddedSQLite()
	if err != nil {
		return sourcePaths{}, err
	}

	storage, err := embeddedStorageFS()
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

func embeddedStorageFS() (iofs.FS, error) {
	storage, err := iofs.Sub(synthetic.Files, "storage")
	if err != nil {
		return nil, fmt.Errorf("open embedded seed storage: %w", err)
	}

	return storage, nil
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
		artworkFiles:      map[string]sourceFile{},
		musicFiles:        map[string]sourceFile{},
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
	if data.artPeriods, err = loadArtPeriods(db); err != nil {
		return sourceData{}, err
	}
	if data.locations, err = loadLocations(db); err != nil {
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
	if data.strings, err = loadStrings(db); err != nil {
		return sourceData{}, err
	}
	if data.staticPages, err = loadStaticPages(db); err != nil {
		return sourceData{}, err
	}
	if data.selections, err = loadSelections(db); err != nil {
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

func loadArtPeriods(db *sql.DB) ([]sourceArtPeriod, error) {
	rows, err := db.Query(`
		SELECT id, name, start_year, end_year, description
		FROM art_periods
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read art periods: %w", err)
	}
	defer closeRows(rows)

	items := []sourceArtPeriod{}
	for rows.Next() {
		item := sourceArtPeriod{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Start, &item.End, &item.Description); err != nil {
			return nil, fmt.Errorf("scan art periods: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// loadLocations reads the producer locations taxonomy. The embedded synthetic
// source and older external exports predate the locations table, so a missing
// table yields an empty set rather than an import failure.
func loadLocations(db *sql.DB) ([]sourceLocation, error) {
	hasLocations, err := hasTable(db, "locations")
	if err != nil {
		return nil, err
	}
	if !hasLocations {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT id, name, COALESCE(city, ''), COALESCE(country, ''), museum, is_public
		FROM locations
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read locations: %w", err)
	}
	defer closeRows(rows)

	items := []sourceLocation{}
	for rows.Next() {
		item := sourceLocation{}
		var museum int
		var isPublic int
		if err := rows.Scan(&item.ID, &item.Name, &item.City, &item.Country, &museum, &isPublic); err != nil {
			return nil, fmt.Errorf("scan locations: %w", err)
		}
		item.Museum = museum != 0
		item.IsPublic = isPublic != 0
		items = append(items, item)
	}

	return items, rows.Err()
}

// loadArtists reads the producer artists records, carrying both the
// encyclopaedic filing form (source_display_name) and the supplied short form
// (display_name) verbatim. A missing display_name column, a NULL value, or a
// blank form fails closed rather than fabricating an identity.
func loadArtists(db *sql.DB) ([]sourceArtist, error) {
	rows, err := db.Query(`
		SELECT id, source_display_name, display_name, COALESCE(birth_year, 0), COALESCE(death_year, 0),
			COALESCE(birth_place, ''), COALESCE(death_place, ''), COALESCE(biography_image_output_path, ''),
			COALESCE(biography_image_width, 0), COALESCE(biography_image_height, 0)
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
		var portraitPath string
		if err := rows.Scan(
			&item.ID,
			&item.DisplayName,
			&item.ShortName,
			&item.BirthYear,
			&item.DeathYear,
			&item.BirthPlace,
			&item.DeathPlace,
			&portraitPath,
			&item.BiographyImageWidth,
			&item.BiographyImageHeight,
		); err != nil {
			return nil, fmt.Errorf("scan artists: %w", err)
		}
		if strings.TrimSpace(item.DisplayName) == "" {
			return nil, fmt.Errorf("artist %q has a blank source_display_name", item.ID)
		}
		if strings.TrimSpace(item.ShortName) == "" {
			return nil, fmt.Errorf("artist %q has a blank display_name", item.ID)
		}
		if portraitPath != "" {
			file, err := preseededSourceFile(portraitPath)
			if err != nil {
				return nil, fmt.Errorf("artist %q portrait path: %w", item.ID, err)
			}
			item.Portrait = file.name
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
	hasLegacyBiographies, err := hasTable(db, "biographies")
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, COALESCE(NULLIF(enriched_biography_html, ''), NULLIF(updated_biography_html, ''), raw_biography_html)
		FROM artists
		ORDER BY id
	`
	if hasLegacyBiographies {
		query = `
			SELECT artist_id, biography_html
			FROM biographies
			ORDER BY artist_id
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("read biographies: %w", err)
	}
	defer closeRows(rows)

	items := []sourceBiography{}
	for rows.Next() {
		item := sourceBiography{}
		if err := rows.Scan(&item.ArtistID, &item.BiographyHTML); err != nil {
			return nil, fmt.Errorf("scan biographies: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func hasTable(db *sql.DB, name string) (bool, error) {
	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)", name).Scan(&exists); err != nil {
		return false, fmt.Errorf("check table %q: %w", name, err)
	}

	return exists, nil
}

// hasColumn reports whether a table declares the named column. It is used to
// carry optional producer artwork columns (date span, location, period)
// that are absent from the embedded synthetic source and older external
// exports.
func hasColumn(db *sql.DB, table string, column string) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect table %q column %q: %w", table, column, err)
	}

	return count > 0, nil
}

// presentColumns reports the presence of each requested column in one pass.
func presentColumns(db *sql.DB, table string, columns ...string) (map[string]bool, error) {
	present := make(map[string]bool, len(columns))
	for _, column := range columns {
		exists, err := hasColumn(db, table, column)
		if err != nil {
			return nil, err
		}
		present[column] = exists
	}

	return present, nil
}

func loadArtworks(db *sql.DB) ([]sourceArtwork, error) {
	present, err := presentColumns(db, "artworks",
		"source_row", "date_start", "date_end", "is_circa", "date_qualifier",
		"timeframe_text", "url", "image_path", "current_location_id", "art_period_id",
		"colour_palette", "colour_signature", "colour_profile_version", "colour_image_hash",
		"source_comment", "comment",
	)
	if err != nil {
		return nil, err
	}

	selects := []string{
		"id", "author_id", "title", "date_text", "technique",
		"COALESCE(dimensions, '')", "location",
		"COALESCE(output_image_path, '')", "COALESCE(image_width, 0)", "COALESCE(image_height, 0)",
		"COALESCE(school_id, '')", "form_id", "COALESCE(type_id, '')",
	}
	if present["source_row"] {
		selects = append(selects, "COALESCE(source_row, 0)")
	}
	if present["date_start"] {
		selects = append(selects, "COALESCE(date_start, 0)")
	}
	if present["date_end"] {
		selects = append(selects, "COALESCE(date_end, 0)")
	}
	if present["is_circa"] {
		selects = append(selects, "COALESCE(is_circa, 0)")
	}
	if present["date_qualifier"] {
		selects = append(selects, "COALESCE(date_qualifier, '')")
	}
	if present["timeframe_text"] {
		selects = append(selects, "COALESCE(timeframe_text, '')")
	}
	if present["url"] {
		selects = append(selects, "COALESCE(url, '')")
	}
	if present["image_path"] {
		selects = append(selects, "COALESCE(image_path, '')")
	}
	if present["current_location_id"] {
		selects = append(selects, "COALESCE(current_location_id, '')")
	}
	if present["art_period_id"] {
		selects = append(selects, "COALESCE(art_period_id, '')")
	}
	if present["colour_palette"] {
		selects = append(selects, "colour_palette")
	}
	if present["colour_signature"] {
		selects = append(selects, "colour_signature")
	}
	if present["colour_profile_version"] {
		selects = append(selects, "COALESCE(colour_profile_version, '')")
	}
	if present["colour_image_hash"] {
		selects = append(selects, "COALESCE(colour_image_hash, '')")
	}
	if present["source_comment"] {
		selects = append(selects, "COALESCE(source_comment, '')")
	}
	if present["comment"] {
		selects = append(selects, "COALESCE(comment, '')")
	}

	rows, err := db.Query("SELECT " + strings.Join(selects, ", ") + " FROM artworks ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read artworks: %w", err)
	}
	defer closeRows(rows)

	items := []sourceArtwork{}
	for rows.Next() {
		item := sourceArtwork{}
		dest := []any{
			&item.ID,
			&item.AuthorID,
			&item.Title,
			&item.DateText,
			&item.Technique,
			&item.Dimensions,
			&item.Location,
			&item.ImagePath,
			&item.ImageWidth,
			&item.ImageHeight,
			&item.SchoolID,
			&item.FormID,
			&item.TypeID,
		}
		var isCirca int
		var colourPalette sql.NullString
		var colourSignature sql.NullString
		if present["source_row"] {
			dest = append(dest, &item.SourceRow)
		}
		if present["date_start"] {
			dest = append(dest, &item.DateStart)
		}
		if present["date_end"] {
			dest = append(dest, &item.DateEnd)
		}
		if present["is_circa"] {
			dest = append(dest, &isCirca)
		}
		if present["date_qualifier"] {
			dest = append(dest, &item.DateQualifier)
		}
		if present["timeframe_text"] {
			dest = append(dest, &item.TimeframeText)
		}
		if present["url"] {
			dest = append(dest, &item.SourceURL)
		}
		if present["image_path"] {
			dest = append(dest, &item.SourcePath)
		}
		if present["current_location_id"] {
			dest = append(dest, &item.CurrentLocationID)
		}
		if present["art_period_id"] {
			dest = append(dest, &item.ArtPeriodID)
		}
		if present["colour_palette"] {
			dest = append(dest, &colourPalette)
		}
		if present["colour_signature"] {
			dest = append(dest, &colourSignature)
		}
		if present["colour_profile_version"] {
			dest = append(dest, &item.ColourProfileVersion)
		}
		if present["colour_image_hash"] {
			dest = append(dest, &item.ColourImageHash)
		}
		if present["source_comment"] {
			dest = append(dest, &item.SourceComment)
		}
		if present["comment"] {
			dest = append(dest, &item.Comment)
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan artworks: %w", err)
		}
		item.IsCirca = isCirca != 0
		if present["colour_palette"] && colourPalette.Valid && colourPalette.String != "" {
			if err := json.Unmarshal([]byte(colourPalette.String), &item.ColourPalette); err != nil {
				return nil, fmt.Errorf("decode artwork %q colour palette: %w", item.ID, err)
			}
		}
		if present["colour_signature"] && colourSignature.Valid && colourSignature.String != "" {
			signature := sourceColourSignature{}
			if err := json.Unmarshal([]byte(colourSignature.String), &signature); err != nil {
				return nil, fmt.Errorf("decode artwork %q colour signature: %w", item.ID, err)
			}
			item.ColourSignature = &signature
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func loadGlossaryEntries(db *sql.DB) ([]sourceGlossaryEntry, error) {
	rows, err := db.Query(`
		SELECT id, term, definition
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
		if err := rows.Scan(&item.ID, &item.Term, &item.Definition); err != nil {
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
		SELECT id, title, period, composer, COALESCE(local_path, ''), COALESCE(art_period_id, '')
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
		if err := rows.Scan(&item.ID, &item.Title, &item.Period, &item.Composer, &item.LocalPath, &item.ArtPeriodID); err != nil {
			return nil, fmt.Errorf("scan music tracks: %w", err)
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

// loadSelections reads the flat producer artwork_selections records. The
// embedded synthetic source and older external exports predate selections, so
// a missing table yields an empty set rather than an import failure.
func loadSelections(db *sql.DB) ([]sourceSelection, error) {
	hasSelections, err := hasTable(db, "artwork_selections")
	if err != nil {
		return nil, err
	}
	if !hasSelections {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT id, artist_id, title, context, display_title, commentary_html,
			artwork_ids, source_path, source_hash, content_hash, published
		FROM artwork_selections
		ORDER BY source_path, id
	`)
	if err != nil {
		return nil, fmt.Errorf("read artwork selections: %w", err)
	}
	defer closeRows(rows)

	items := []sourceSelection{}
	for rows.Next() {
		item := sourceSelection{}
		var artworkIDs string
		var published int
		if err := rows.Scan(
			&item.ID,
			&item.ArtistID,
			&item.Title,
			&item.Context,
			&item.DisplayTitle,
			&item.Commentary,
			&artworkIDs,
			&item.SourcePath,
			&item.SourceHash,
			&item.ContentHash,
			&published,
		); err != nil {
			return nil, fmt.Errorf("scan artwork selections: %w", err)
		}
		ids, err := parseArtworkIDs(artworkIDs)
		if err != nil {
			return nil, fmt.Errorf("selection %q artwork IDs: %w", item.ID, err)
		}
		item.ArtworkIDs = ids
		item.Published = published != 0
		items = append(items, item)
	}

	return items, rows.Err()
}

// parseArtworkIDs decodes the producer's ordered JSON artwork membership. The
// order is preserved verbatim; a non-array or empty payload is rejected, and
// duplicate artwork IDs are rejected before PocketBase relation normalisation
// could silently de-duplicate them.
func parseArtworkIDs(value string) ([]string, error) {
	var ids []string
	if err := json.Unmarshal([]byte(value), &ids); err != nil {
		return nil, fmt.Errorf("decode ordered artwork membership: %w", err)
	}
	if len(ids) == 0 {
		return nil, errors.New("ordered artwork membership is required")
	}
	seen := make(map[string]struct{}, len(ids))
	for index, artworkID := range ids {
		if _, exists := seen[artworkID]; exists {
			return nil, fmt.Errorf("artwork ID %q is duplicated at position %d", artworkID, index+1)
		}
		seen[artworkID] = struct{}{}
	}

	return ids, nil
}

func validateSourceRelations(data sourceData) error {
	artistIDs := makeIDSet(data.artists, func(item sourceArtist) string { return item.ID })
	schoolIDs := makeIDSet(data.schools, func(item sourceTaxonomy) string { return item.ID })
	formIDs := makeIDSet(data.forms, func(item sourceTaxonomy) string { return item.ID })
	typeIDs := makeIDSet(data.types, func(item sourceTaxonomy) string { return item.ID })
	artPeriodIDs := makeIDSet(data.artPeriods, func(item sourceArtPeriod) string { return item.ID })
	professionIDs := makeIDSet(data.professions, func(item sourceTaxonomy) string { return item.ID })
	locationIDs := makeIDSet(data.locations, func(item sourceLocation) string { return item.ID })

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
			return fmt.Errorf("biography references unknown artist %q", biography.ArtistID)
		}
	}

	for _, artwork := range data.artworks {
		if _, ok := artistIDs[artwork.AuthorID]; !ok {
			return fmt.Errorf("artwork %q references unknown artist %q", artwork.ID, artwork.AuthorID)
		}
		if artwork.SchoolID != "" {
			if _, ok := schoolIDs[artwork.SchoolID]; !ok {
				return fmt.Errorf("artwork %q references unknown school %q", artwork.ID, artwork.SchoolID)
			}
		}
		if _, ok := formIDs[artwork.FormID]; !ok {
			return fmt.Errorf("artwork %q references unknown form %q", artwork.ID, artwork.FormID)
		}
		if artwork.TypeID != "" {
			if _, ok := typeIDs[artwork.TypeID]; !ok {
				return fmt.Errorf("artwork %q references unknown type %q", artwork.ID, artwork.TypeID)
			}
		}
		if artwork.CurrentLocationID != "" {
			if _, ok := locationIDs[artwork.CurrentLocationID]; !ok {
				return fmt.Errorf("artwork %q references unknown location %q", artwork.ID, artwork.CurrentLocationID)
			}
		}
		if artwork.ArtPeriodID != "" {
			if _, ok := artPeriodIDs[artwork.ArtPeriodID]; !ok {
				return fmt.Errorf("artwork %q references unknown art period %q", artwork.ID, artwork.ArtPeriodID)
			}
		}
	}

	for _, track := range data.musicTracks {
		if track.ArtPeriodID == "" {
			continue
		}
		if _, ok := artPeriodIDs[track.ArtPeriodID]; !ok {
			return fmt.Errorf("music track %q references unknown art period %q", track.ID, track.ArtPeriodID)
		}
	}

	if err := validateSourceSelections(data, artistIDs); err != nil {
		return err
	}

	return nil
}

// validateSourceSelections rejects selection records that are unpublished,
// carry a non-canonical source path or a mismatched deterministic ID/source
// hash, hold a malformed content hash, duplicate an ID or source path,
// reference a missing artist, or list artworks that are missing or belong to a
// different artist. Ordered membership is preserved by the importer and is not
// re-ordered here.
func validateSourceSelections(data sourceData, artistIDs map[string]struct{}) error {
	artworkArtist := make(map[string]string, len(data.artworks))
	for _, artwork := range data.artworks {
		artworkArtist[artwork.ID] = artwork.AuthorID
	}

	seenIDs := make(map[string]struct{}, len(data.selections))
	seenSourcePaths := make(map[string]struct{}, len(data.selections))
	for _, selection := range data.selections {
		if !selection.Published {
			return fmt.Errorf("selection %q is not published", selection.ID)
		}
		if err := validateSelectionProvenance(selection); err != nil {
			return err
		}
		if !isSHA256Hex(selection.ContentHash) {
			return fmt.Errorf("selection %q content hash %q is not a SHA-256 digest", selection.ID, selection.ContentHash)
		}
		if _, ok := artistIDs[selection.ArtistID]; !ok {
			return fmt.Errorf("selection %q references unknown artist %q", selection.ID, selection.ArtistID)
		}
		if _, duplicate := seenIDs[selection.ID]; duplicate {
			return fmt.Errorf("duplicate selection ID %q", selection.ID)
		}
		if _, duplicate := seenSourcePaths[selection.SourcePath]; duplicate {
			return fmt.Errorf("duplicate selection source path %q", selection.SourcePath)
		}
		seenIDs[selection.ID] = struct{}{}
		seenSourcePaths[selection.SourcePath] = struct{}{}

		for _, artworkID := range selection.ArtworkIDs {
			artistID, ok := artworkArtist[artworkID]
			if !ok {
				return fmt.Errorf("selection %q references unknown artwork %q", selection.ID, artworkID)
			}
			if artistID != selection.ArtistID {
				return fmt.Errorf("selection %q artwork %q belongs to artist %q, not %q", selection.ID, artworkID, artistID, selection.ArtistID)
			}
		}
	}

	return nil
}

// validateSelectionProvenance verifies the producer's deterministic selection
// identity: the source path must be canonical, the record ID must equal "r"
// plus the first 14 hex characters of the SHA-256 canonical source path, and
// the source hash must equal the full SHA-256 canonical source path.
func validateSelectionProvenance(selection sourceSelection) error {
	canonical, err := canonicalSelectionSourcePath(selection.SourcePath)
	if err != nil {
		return fmt.Errorf("selection %q: %w", selection.SourcePath, err)
	}
	if canonical != selection.SourcePath {
		return fmt.Errorf("selection source path %q is not canonical (%q)", selection.SourcePath, canonical)
	}

	wantID, err := selectionIDFromSourcePath(selection.SourcePath)
	if err != nil {
		return fmt.Errorf("selection %q: derive deterministic ID: %w", selection.SourcePath, err)
	}
	if selection.ID != wantID {
		return fmt.Errorf("selection %q ID %q does not match deterministic source ID %q", selection.SourcePath, selection.ID, wantID)
	}

	wantHash, err := selectionSourceHashFromPath(selection.SourcePath)
	if err != nil {
		return fmt.Errorf("selection %q: derive deterministic source hash: %w", selection.SourcePath, err)
	}
	if selection.SourceHash != wantHash {
		return fmt.Errorf("selection %q source hash %q does not match deterministic hash %q", selection.SourcePath, selection.SourceHash, wantHash)
	}

	return nil
}

// canonicalSelectionSourcePath normalises a producer source path to the stored
// html/.../index.html source identity, matching the producer algorithm without
// importing it. Physical in/ prefixes are not retained and traversal segments
// are rejected.
func canonicalSelectionSourcePath(value string) (string, error) {
	normalised := strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	if normalised == "" {
		return "", errors.New("selection source path is required")
	}
	for _, segment := range strings.Split(normalised, "/") {
		if segment == ".." {
			return "", errors.New("selection source path must stay within the source root")
		}
	}

	normalised = strings.TrimPrefix(normalised, "/")
	normalised = path.Clean(normalised)
	normalised = strings.TrimPrefix(normalised, "in/")
	if normalised == "." || !strings.HasPrefix(normalised, "html/") {
		return "", errors.New("selection source path must be an html source path")
	}
	if path.Base(normalised) != "index.html" {
		return "", fmt.Errorf("selection source path must end in index.html, got %q", path.Base(normalised))
	}

	return normalised, nil
}

// selectionSourceHashFromPath returns the full SHA-256 hex digest of the
// canonical selection source identity.
func selectionSourceHashFromPath(value string) (string, error) {
	canonical, err := canonicalSelectionSourcePath(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:]), nil
}

// selectionIDFromSourcePath returns the deterministic 15-character producer
// selection record ID derived exclusively from the canonical source identity.
func selectionIDFromSourcePath(value string) (string, error) {
	sourceHash, err := selectionSourceHashFromPath(value)
	if err != nil {
		return "", err
	}
	return "r" + sourceHash[:14], nil
}

func makeIDSet[T any](items []T, id func(T) string) map[string]struct{} {
	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		ids[id(item)] = struct{}{}
	}

	return ids
}

// isSHA256Hex reports whether the value is a lowercase 64-character SHA-256
// hex digest, matching the producer's content-hash contract.
func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') {
			continue
		}
		return false
	}

	return true
}

func loadSourceFiles(paths sourcePaths, data *sourceData) error {
	if paths.preseededAssets {
		return loadPreseededSourceFiles(paths.storageRoot, data)
	}

	return loadEmbeddedSourceFiles(paths.storage, data)
}

// loadPreseededSourceFiles resolves the preseeded storage filename for each
// source record and, for image-backed artworks, stats the paired staged
// original to record its exact byte size. An artwork with no image metadata is
// a valid image-less artwork and simply has no file entry; a declared non-empty
// path must resolve to a safe relative storage path whose staged original is a
// present, non-empty, regular file, or the import fails closed.
func loadPreseededSourceFiles(storageRoot string, data *sourceData) error {
	for _, artwork := range data.artworks {
		if artwork.ImagePath == "" {
			continue
		}
		file, err := preseededArtworkFile(storageRoot, artwork.ImagePath)
		if err != nil {
			return fmt.Errorf("artwork %q storage path: %w", artwork.ID, err)
		}
		data.artworkFiles[artwork.ID] = file
	}

	for _, track := range data.musicTracks {
		file, err := preseededSourceFile(track.LocalPath)
		if err != nil {
			return fmt.Errorf("music track %q storage path: %w", track.ID, err)
		}
		data.musicFiles[track.ID] = file
	}

	return nil
}

func preseededSourceFile(value string) (sourceFile, error) {
	sourcePath, err := safeRelativePath(value)
	if err != nil {
		return sourceFile{}, err
	}

	return sourceFile{name: path.Base(sourcePath), preseededAssets: true}, nil
}

// preseededArtworkFile resolves the preseeded artwork storage filename and
// records the exact byte size of the paired staged original. The size is
// statted from the staged file itself — never inferred from the filename,
// extension, or dimensions. Both the storage root and the staged original are
// resolved to their canonical absolute paths with symlinks followed, and the
// resolved original must remain inside the resolved root, so a file or
// parent-directory symlink escape fails the import closed. A missing,
// non-regular, or empty declared original likewise fails closed.
func preseededArtworkFile(storageRoot string, value string) (sourceFile, error) {
	sourcePath, err := safeRelativePath(value)
	if err != nil {
		return sourceFile{}, err
	}

	absRoot, err := filepath.Abs(storageRoot)
	if err != nil {
		return sourceFile{}, fmt.Errorf("resolve storage root %q: %w", storageRoot, err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return sourceFile{}, fmt.Errorf("resolve storage root %q: %w", storageRoot, err)
	}

	resolvedCandidate, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, sourcePath))
	if err != nil {
		return sourceFile{}, fmt.Errorf("resolve staged original %q: %w", sourcePath, err)
	}

	rel, err := filepath.Rel(resolvedRoot, resolvedCandidate)
	if err != nil {
		return sourceFile{}, fmt.Errorf("staged original %q escapes storage root: %w", sourcePath, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return sourceFile{}, fmt.Errorf("staged original %q escapes storage root", sourcePath)
	}

	info, err := os.Stat(resolvedCandidate)
	if err != nil {
		return sourceFile{}, fmt.Errorf("stat staged original %q: %w", sourcePath, err)
	}
	if !info.Mode().IsRegular() {
		return sourceFile{}, fmt.Errorf("staged original %q is not a regular file", sourcePath)
	}
	if info.Size() == 0 {
		return sourceFile{}, fmt.Errorf("staged original %q is empty", sourcePath)
	}

	return sourceFile{name: path.Base(sourcePath), preseededAssets: true, size: info.Size()}, nil
}

func loadEmbeddedSourceFiles(storage iofs.FS, data *sourceData) error {
	for _, artwork := range data.artworks {
		filename, err := singleSourceFile(storage, path.Join("Artworks", artwork.ID))
		if err != nil {
			return fmt.Errorf("artwork %q storage: %w", artwork.ID, err)
		}
		content, err := readSourceFile(storage, sourceFilePath("Artworks", artwork.ID, filename))
		if err != nil {
			return fmt.Errorf("artwork %q storage: %w", artwork.ID, err)
		}
		data.artworkFiles[artwork.ID] = sourceFile{name: filename, content: content, size: int64(len(content))}
	}

	for _, track := range data.musicTracks {
		sourcePath, err := sourceMusicFilePath(track)
		if err != nil {
			return fmt.Errorf("music track %q storage path: %w", track.ID, err)
		}
		content, err := readSourceFile(storage, sourcePath)
		if err != nil {
			return fmt.Errorf("music track %q storage: %w", track.ID, err)
		}
		data.musicFiles[track.ID] = sourceFile{name: path.Base(sourcePath), content: content}
	}

	return nil
}

func readSourceFile(storage iofs.FS, sourcePath string) ([]byte, error) {
	content, err := iofs.ReadFile(storage, sourcePath)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, errors.New("source file is empty")
	}

	return content, nil
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
	return safeRelativePath(track.LocalPath)
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
