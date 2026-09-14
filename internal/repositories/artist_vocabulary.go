package repositories

import (
	"sort"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

// ArtistSchoolOption is one complete, source-backed school vocabulary entry.
type ArtistSchoolOption struct {
	ID   string
	Slug string
	Name string
}

// ArtistPeriodOption is an art period associated with at least one public artist.
type ArtistPeriodOption struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Start int    `db:"start"`
	End   int    `db:"end"`
}

// ListArtistSchools returns the complete approved school roster, including
// entries which currently have no artist holdings.
func ListArtistSchools(app core.App) ([]ArtistSchoolOption, error) {
	records, err := app.FindRecordsByFilter(constants.CollectionSchools, "", "", 0, 0)
	if err != nil {
		return nil, err
	}
	result := make([]ArtistSchoolOption, 0, len(records))
	for _, record := range records {
		name := strings.TrimSpace(record.GetString("name"))
		slug := strings.TrimSpace(record.GetString("slug"))
		if name == "" || slug == "" {
			continue
		}
		result = append(result, ArtistSchoolOption{ID: record.Id, Slug: slug, Name: name})
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := strings.ToLower(result[i].Name), strings.ToLower(result[j].Name)
		if left == right {
			return result[i].ID < result[j].ID
		}
		return left < right
	})
	return result, nil
}

// ListArtistPeriods returns only periods whose range is authoritatively
// associated with at least one published artist carrying public identity data.
func ListArtistPeriods(app core.App) ([]ArtistPeriodOption, error) {
	present, err := artistsIdentityFieldsPresent(app)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}

	rows := []ArtistPeriodOption{}
	err = app.DB().NewQuery(`
		SELECT period.id, period.name, period.start, period.end
		FROM Art_periods AS period
		WHERE period.name != '' AND period.start > 0 AND period.end >= period.start
		AND EXISTS (
			SELECT 1 FROM Artists AS artist
			WHERE artist.published IS true
			AND TRIM(artist.filing_name) != '' AND TRIM(artist.short_name) != ''
			AND artist.year_of_birth BETWEEN period.start AND period.end
		)
		ORDER BY LOWER(period.name), period.name, period.id
	`).All(&rows)
	return rows, err
}
