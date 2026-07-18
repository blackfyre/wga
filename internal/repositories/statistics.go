package repositories

import "github.com/pocketbase/pocketbase/core"

type StatisticsRepository struct {
	app core.App
}

type ArtFormDistribution struct {
	Name  string `db:"name" json:"name"`
	Count int    `db:"count" json:"count"`
}

// SchoolPeriodRow holds a count for a (50-year period bucket, school) pair.
// period_start is the lower bound of the 50-year bucket (e.g. 1400 covers 1400-1449).
type SchoolPeriodRow struct {
	PeriodStart int    `db:"period_start" json:"period_start"`
	School      string `db:"school_label" json:"school"`
	Count       int    `db:"count" json:"count"`
}

// top7Schools are singled out; everything else becomes "Other".
const top7Schools = `('Italian','French','Dutch','Flemish','German','English','Spanish')`

func NewStatisticsRepository(app core.App) *StatisticsRepository {
	return &StatisticsRepository{app: app}
}

func (r *StatisticsRepository) GetArtFormDistribution() ([]ArtFormDistribution, error) {
	rows := []ArtFormDistribution{}
	// PocketBase stores multi-select relation fields as JSON arrays (e.g. ["id"]),
	// so we use json_each to unnest the array before joining.
	err := r.app.DB().NewQuery(`
		SELECT af.name AS name, COUNT(DISTINCT a.id) AS count
		FROM Artworks a
		CROSS JOIN json_each(a.form) AS je
		JOIN Art_forms af ON je.value = af.id
		WHERE a.published IS true
		GROUP BY af.id, af.name
		ORDER BY count DESC
	`).All(&rows)

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *StatisticsRepository) GetArtworksBySchoolAndPeriod() ([]SchoolPeriodRow, error) {
	rows := []SchoolPeriodRow{}
	err := r.app.DB().NewQuery(`
		SELECT
			(CAST(ar.year_of_birth / 50 AS INTEGER) * 50) AS period_start,
			CASE WHEN s.name IN ` + top7Schools + `
			     THEN s.name ELSE 'Other' END AS school_label,
			COUNT(DISTINCT aw.id) AS count
		FROM Artworks aw
		CROSS JOIN json_each(aw.author) je_a
		JOIN Artists ar ON je_a.value = ar.id
		CROSS JOIN json_each(aw.school) je_s
		JOIN Schools s ON je_s.value = s.id
		WHERE aw.published IS true AND ar.year_of_birth > 0
		GROUP BY period_start, school_label
		ORDER BY period_start, count DESC
	`).All(&rows)

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *StatisticsRepository) GetArtistsBySchoolAndPeriod() ([]SchoolPeriodRow, error) {
	rows := []SchoolPeriodRow{}
	err := r.app.DB().NewQuery(`
		SELECT
			(CAST(a.year_of_birth / 50 AS INTEGER) * 50) AS period_start,
			CASE WHEN s.name IN ` + top7Schools + `
			     THEN s.name ELSE 'Other' END AS school_label,
			COUNT(DISTINCT a.id) AS count
		FROM Artists a
		CROSS JOIN json_each(a.school) je_s
		JOIN Schools s ON je_s.value = s.id
		WHERE a.published IS true AND a.year_of_birth > 0
		GROUP BY period_start, school_label
		ORDER BY period_start, count DESC
	`).All(&rows)

	if err != nil {
		return nil, err
	}

	return rows, nil
}
