package timeline

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// repository owns the timeline's bounded persistence reads. It is private to
// the timeline capability so no other module can reach into the artwork
// date-span queries (ADR 0008).
type repository struct {
	app core.App
}

func newRepository(app core.App) *repository {
	return &repository{app: app}
}

// artPeriod is one approved art-period span with its descriptive label.
type artPeriod struct {
	id          string
	name        string
	start       int
	end         int
	description string
}

// artistRow is the bounded projection of a published artist with a known birth
// year for the timeline's artist lane.
type artistRow struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	FilingName  string `db:"filing_name"`
	YearOfBirth int    `db:"year_of_birth"`
}

// publishedArtistLaneWhere is the predicate a published artist must satisfy
// before it is linked from the timeline: a positive stored birth year and both
// authoritative identity fields. The identity clause mirrors the canonical
// public-record condition used by the artist index/record surfaces, so an
// artist entry is only linked when its published public record resolves.
const publishedArtistLaneWhere = `
		published = true
		AND year_of_birth > 0
		AND filing_name IS NOT NULL AND TRIM(filing_name) != ''
		AND short_name IS NOT NULL AND TRIM(short_name) != ''`

// artworkRow is the bounded projection of a published artwork for the timeline.
type artworkRow struct {
	ID         string `db:"id"`
	Title      string `db:"title"`
	Image      string `db:"image"`
	ImageWidth int    `db:"image_width"`
	DateStart  int    `db:"date_start"`
	DateEnd    int    `db:"date_end"`
	IsCirca    int    `db:"is_circa"`
	Qualifier  string `db:"date_qualifier"`
	ArtistID   string `db:"artist_id"`
	ArtistName string `db:"artist_name"`
}

type countRow struct {
	Count int `db:"count"`
}

type boundsRow struct {
	Min int `db:"min_year"`
	Max int `db:"max_year"`
}

// dateSpan is a published artwork's inclusive creation-date range.
type dateSpan struct {
	Start int `db:"date_start"`
	End   int `db:"date_end"`
}

const (
	publishedWorksJoin = `
		FROM Artworks AS aw
		INNER JOIN Artists AS ar ON ar.id = json_extract(aw.author, '$[0]')
		WHERE aw.published = true AND ar.published = true`
)

// artworkBounds returns the inclusive published-artwork year range [min, max],
// or (0, 0) when no published artwork has a known date.
func (r *repository) artworkBounds() (int, int, error) {
	row := boundsRow{}
	err := r.app.DB().NewQuery(`
		SELECT
			COALESCE(MIN(aw.date_start), 0) AS min_year,
			COALESCE(MAX(CASE WHEN aw.date_end > 0 THEN aw.date_end ELSE aw.date_start END), 0) AS max_year` +
		publishedWorksJoin + `
			AND aw.date_start > 0`,
	).One(&row)
	if err != nil {
		return 0, 0, err
	}

	return row.Min, row.Max, nil
}

// listPeriods returns all approved art periods ordered by start, then name,
// then id so equal-span or equal-name periods sort deterministically.
func (r *repository) listPeriods() ([]artPeriod, error) {
	records, err := r.app.FindRecordsByFilter("art_periods", "", "+start,+name,+id", 0, 0)
	if err != nil {
		return nil, err
	}

	periods := make([]artPeriod, 0, len(records))
	for _, record := range records {
		periods = append(periods, artPeriod{
			id:          record.Id,
			name:        record.GetString("name"),
			start:       record.GetInt("start"),
			end:         record.GetInt("end"),
			description: record.GetString("description"),
		})
	}

	return periods, nil
}

// countWorks returns the number of published works whose date span overlaps the
// inclusive window [from, to].
func (r *repository) countWorks(from int, to int) (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery(`
		SELECT COUNT(*) AS count` +
		publishedWorksJoin + `
			AND aw.date_start > 0
			AND aw.date_start <= {:to}
			AND (CASE WHEN aw.date_end > 0 THEN aw.date_end ELSE aw.date_start END) >= {:from}`,
	).Bind(dbx.Params{"from": from, "to": to}).One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

// dateSpans returns the (start, end) pairs of every published work whose date
// span overlaps the inclusive window [from, to], ordered by start year. The
// caller buckets these into decade density bins using the same overlap
// semantics as marks and counts.
func (r *repository) dateSpans(from int, to int) ([]dateSpan, error) {
	rows := []dateSpan{}
	err := r.app.DB().NewQuery(`
		SELECT
			aw.date_start AS date_start,
			aw.date_end AS date_end` +
		publishedWorksJoin + `
			AND aw.date_start > 0
			AND aw.date_start <= {:to}
			AND (CASE WHEN aw.date_end > 0 THEN aw.date_end ELSE aw.date_start END) >= {:from}
		ORDER BY aw.date_start ASC`,
	).Bind(dbx.Params{"from": from, "to": to}).All(&rows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// countArtists returns the number of published artists whose known birth year
// falls inside the inclusive window [from, to].
func (r *repository) countArtists(from int, to int) (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM Artists
		WHERE` + publishedArtistLaneWhere + `
			AND year_of_birth >= {:from}
			AND year_of_birth <= {:to}`,
	).Bind(dbx.Params{"from": from, "to": to}).One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

// listArtists returns at most limit published artists whose known birth year
// falls inside the inclusive window [from, to], ordered deterministically by
// birth year, then filing name, then id.
func (r *repository) listArtists(from int, to int, limit int) ([]artistRow, error) {
	if limit <= 0 {
		return []artistRow{}, nil
	}

	rows := []artistRow{}
	err := r.app.DB().NewQuery(`
		SELECT
			id AS id,
			name AS name,
			filing_name AS filing_name,
			year_of_birth AS year_of_birth
		FROM Artists
		WHERE` + publishedArtistLaneWhere + `
			AND year_of_birth >= {:from}
			AND year_of_birth <= {:to}
		ORDER BY year_of_birth ASC, filing_name ASC, id ASC
		LIMIT {:limit}`,
	).Bind(dbx.Params{"from": from, "to": to, "limit": limit}).All(&rows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// listWorks returns at most limit published works whose date span overlaps the
// inclusive window [from, to], ordered deterministically by start year then id,
// starting at the given offset.
func (r *repository) listWorks(from int, to int, limit int, offset int) ([]artworkRow, error) {
	if limit <= 0 {
		return []artworkRow{}, nil
	}
	if offset < 0 {
		offset = 0
	}

	rows := []artworkRow{}
	err := r.app.DB().NewQuery(`
		SELECT
			aw.id AS id,
			aw.title AS title,
			aw.image AS image,
			aw.image_width AS image_width,
			aw.date_start AS date_start,
			aw.date_end AS date_end,
			aw.is_circa AS is_circa,
			aw.date_qualifier AS date_qualifier,
			ar.id AS artist_id,
			ar.name AS artist_name` +
		publishedWorksJoin + `
			AND aw.date_start > 0
			AND aw.date_start <= {:to}
			AND (CASE WHEN aw.date_end > 0 THEN aw.date_end ELSE aw.date_start END) >= {:from}
		ORDER BY aw.date_start ASC, aw.id ASC
		LIMIT {:limit} OFFSET {:offset}`,
	).Bind(dbx.Params{"from": from, "to": to, "limit": limit, "offset": offset}).All(&rows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}
