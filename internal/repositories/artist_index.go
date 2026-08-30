package repositories

import (
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// publishedArtistIdentity is the SQL predicate a published artist must satisfy
// before appearing on any public artist surface: both authoritative identity
// fields (filing and short form) must be present. Prior-bootstrap records carry
// blank filing/short fields and are denied rather than reconstructed.
const publishedArtistIdentity = "filing_name IS NOT NULL AND TRIM(filing_name) != '' AND short_name IS NOT NULL AND TRIM(short_name) != ''"

// filingInitialFold maps the non-ASCII producer filing initials present in the
// supplied dataset to their ASCII navigation letter. Navigation only: the
// displayed filing-name bytes are never altered, and no general name reversal,
// uppercasing, or reconstruction is applied.
var filingInitialFold = map[string]string{
	"Á": "A", // U+00C1 LATIN CAPITAL LETTER A WITH ACUTE
	"Ś": "S", // U+015A LATIN CAPITAL LETTER S WITH ACUTE
}

// artistsIdentityFieldsPresent reports whether the artists collection schema
// carries both authoritative identity fields. A legacy or malformed
// pre-migration instance whose collection lacks either field fails closed: no
// artist identity is rendered rather than emitting SQL errors or fabricating
// fallback names. A missing artists collection is denied the same way. Only a
// genuine failure to read collection metadata is returned as an error.
func artistsIdentityFieldsPresent(app core.App) (bool, error) {
	collection, err := app.FindCachedCollectionByNameOrId("artists")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return collection.Fields.GetByName("filing_name") != nil &&
		collection.Fields.GetByName("short_name") != nil, nil
}

// ArtistSort is the accepted artist-index sort order.
type ArtistSort string

const (
	ArtistSortNameAsc  ArtistSort = "az"
	ArtistSortNameDesc ArtistSort = "za"
	ArtistSortBirth    ArtistSort = "birth"
)

// ArtistIndexFilter is the bounded query contract for the artist index.
type ArtistIndexFilter struct {
	Query        string
	Letter       string
	School       string
	PeriodActive bool
	PeriodStart  int
	PeriodEnd    int
	BornActive   bool
	BornFrom     int
	BornTo       int
	Sort         ArtistSort
	Limit        int
	Offset       int
}

// IndexedArtist pairs a published artist record with its page availability.
type IndexedArtist struct {
	Record    *core.Record
	Available bool
}

// ArtistIndexRepository is the bounded read-model for the public artist index.
type ArtistIndexRepository struct {
	app core.App
}

func NewArtistIndexRepository(app core.App) *ArtistIndexRepository {
	return &ArtistIndexRepository{app: app}
}

// CountArtists returns the number of published artists matching the filter.
func (r *ArtistIndexRepository) CountArtists(filter ArtistIndexFilter) (int, error) {
	present, err := artistsIdentityFieldsPresent(r.app)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, nil
	}

	query := r.app.RecordQuery("artists")
	for _, expr := range r.artistWhere(filter) {
		query.AndWhere(expr)
	}
	query.Select("COUNT(DISTINCT id)")

	var count int
	if err := query.Row(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// ListArtists returns the published artists matching the filter in deterministic
// order. Availability is resolved once per page with a single bound aggregate
// query and is skipped entirely for an empty page.
func (r *ArtistIndexRepository) ListArtists(filter ArtistIndexFilter) ([]IndexedArtist, error) {
	present, err := artistsIdentityFieldsPresent(r.app)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}

	query := r.app.RecordQuery("artists")
	for _, expr := range r.artistWhere(filter) {
		query.AndWhere(expr)
	}

	switch filter.Sort {
	case ArtistSortNameDesc:
		query.OrderBy("filing_name DESC", "id ASC")
	case ArtistSortBirth:
		// Known birth years ascending, unknown years last, then filing name and id for
		// a deterministic tiebreak.
		query.OrderBy("(year_of_birth = 0) ASC", "year_of_birth ASC", "filing_name ASC", "id ASC")
	default:
		query.OrderBy("filing_name ASC", "id ASC")
	}

	if filter.Limit > 0 {
		query.Limit(int64(filter.Limit))
	}
	if filter.Offset > 0 {
		query.Offset(int64(filter.Offset))
	}

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = record.Id
	}
	available, err := r.publishedArtworkAuthorIDs(ids)
	if err != nil {
		return nil, err
	}

	result := make([]IndexedArtist, len(records))
	for i, record := range records {
		result[i] = IndexedArtist{Record: record, Available: available[record.Id]}
	}

	return result, nil
}

// ListAvailableLetters returns the sorted uppercase initials of published artist
// filing names. The filter argument is accepted for call-site symmetry; letter
// availability reflects the complete published set, never the active filters.
func (r *ArtistIndexRepository) ListAvailableLetters(_ ArtistIndexFilter) ([]string, error) {
	present, err := artistsIdentityFieldsPresent(r.app)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}

	rows := []struct {
		Letter string `db:"letter"`
	}{}
	err = r.app.DB().NewQuery(`
		SELECT DISTINCT UPPER(SUBSTR(TRIM(filing_name), 1, 1)) AS letter
		FROM Artists
		WHERE published IS true AND ` + publishedArtistIdentity + `
	`).All(&rows)
	if err != nil {
		return nil, err
	}

	letters := []string{}
	for _, row := range rows {
		letter := row.Letter
		if folded, ok := filingInitialFold[letter]; ok {
			letter = folded
		}
		if len(letter) == 1 && letter[0] >= 'A' && letter[0] <= 'Z' {
			letters = append(letters, letter)
		}
	}
	sort.Strings(letters)

	return letters, nil
}

// BirthYearBounds returns the inclusive minimum and maximum known birth year
// among published artists. When no published artist has a known birth year it
// returns (0, 0).
func (r *ArtistIndexRepository) BirthYearBounds() (int, int, error) {
	present, err := artistsIdentityFieldsPresent(r.app)
	if err != nil {
		return 0, 0, err
	}
	if !present {
		return 0, 0, nil
	}

	row := struct {
		Min int `db:"min"`
		Max int `db:"max"`
	}{}
	err = r.app.DB().NewQuery(`
		SELECT
			COALESCE(MIN(year_of_birth), 0) AS min,
			COALESCE(MAX(year_of_birth), 0) AS max
		FROM Artists
		WHERE published IS true AND year_of_birth > 0 AND ` + publishedArtistIdentity + `
	`).One(&row)
	if err != nil {
		return 0, 0, err
	}

	return row.Min, row.Max, nil
}

func (r *ArtistIndexRepository) artistWhere(filter ArtistIndexFilter) []dbx.Expression {
	exprs := []dbx.Expression{
		dbx.NewExp("published = true"),
		dbx.NewExp(publishedArtistIdentity),
	}

	if filter.Query != "" {
		exprs = append(exprs, dbx.NewExp(
			"(filing_name LIKE {:query} OR filing_name LIKE {:query_upper})",
			dbx.Params{
				"query":       "%" + filter.Query + "%",
				"query_upper": "%" + strings.ToUpper(filter.Query) + "%",
			},
		))
	}
	if filter.Letter != "" {
		exprs = append(exprs, filingLetterExpression(filter.Letter))
	}
	if filter.School != "" {
		exprs = append(exprs, dbx.NewExp(
			"EXISTS (SELECT 1 FROM json_each(Artists.school) je_s JOIN Schools s ON s.id = je_s.value WHERE s.slug = {:school})",
			dbx.Params{"school": filter.School},
		))
	}
	if filter.PeriodActive {
		exprs = append(exprs, dbx.NewExp(
			"year_of_birth >= {:period_start} AND year_of_birth <= {:period_end}",
			dbx.Params{"period_start": filter.PeriodStart, "period_end": filter.PeriodEnd},
		))
	}
	if filter.BornActive {
		// Each bound applies independently so a partial range (one side unset,
		// represented by 0) never emits a hidden contradictory clause.
		if filter.BornFrom > 0 {
			exprs = append(exprs, dbx.NewExp(
				"year_of_birth >= {:born_from}",
				dbx.Params{"born_from": filter.BornFrom},
			))
		}
		if filter.BornTo > 0 {
			exprs = append(exprs, dbx.NewExp(
				"year_of_birth <= {:born_to}",
				dbx.Params{"born_to": filter.BornTo},
			))
		}
	}

	return exprs
}

// filingLetterExpression builds the navigation-letter predicate. The selected
// ASCII initial is bound so a direct repository caller cannot alter the query
// fragment; it matches through case-insensitive LIKE (so a lowercase producer
// initial such as trAUTMANN still resolves), together with any non-ASCII
// producer initials that fold into it (Á→A, Ś→S). The fold initials are fixed
// producer literals drawn from filingInitialFold, never caller input, so they
// are embedded as safe SQL literals.
func filingLetterExpression(letter string) dbx.Expression {
	exprs := []dbx.Expression{
		dbx.NewExp("filing_name LIKE {:letter}", dbx.Params{"letter": letter + "%"}),
	}
	for initial, folded := range filingInitialFold {
		if folded == letter {
			exprs = append(exprs, dbx.NewExp("filing_name LIKE '"+initial+"%'"))
		}
	}

	return dbx.Or(exprs...)
}

// publishedArtworkAuthorIDs returns the subset of candidateIDs that have at least
// one published artwork. It runs a single bound aggregate query; an empty
// candidate set returns an empty result without querying.
func (r *ArtistIndexRepository) publishedArtworkAuthorIDs(candidateIDs []string) (map[string]bool, error) {
	available := map[string]bool{}
	if len(candidateIDs) == 0 {
		return available, nil
	}

	placeholders := make([]string, len(candidateIDs))
	params := dbx.Params{}
	for i, id := range candidateIDs {
		key := "artist_id_" + strconv.Itoa(i)
		placeholders[i] = "{:" + key + "}"
		params[key] = id
	}

	rows := []struct {
		AuthorID string `db:"author_id"`
	}{}
	query := `
		SELECT DISTINCT je.value AS author_id
		FROM Artworks
		CROSS JOIN json_each(Artworks.author) je
		WHERE Artworks.published IS true
		  AND je.value IN (` + strings.Join(placeholders, ", ") + `)
	`
	if err := r.app.DB().NewQuery(query).Bind(params).All(&rows); err != nil {
		return nil, err
	}

	for _, row := range rows {
		available[row.AuthorID] = true
	}

	return available, nil
}
