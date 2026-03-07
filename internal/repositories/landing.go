package repositories

import (
	"fmt"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type LandingRepository struct {
	app *pocketbase.PocketBase
}

type countRow struct {
	Count int `db:"c"`
}

func NewLandingRepository(app *pocketbase.PocketBase) *LandingRepository {
	return &LandingRepository{app: app}
}

func (r *LandingRepository) GetWelcomeContent() (string, error) {
	record, err := r.app.FindFirstRecordByData(constants.CollectionStrings, "name", "welcome")
	if err != nil {
		return "", err
	}

	return getRecordStringField(record, "content")
}

func (r *LandingRepository) CountPublishedArtists() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM artists WHERE published IS true").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func (r *LandingRepository) CountPublishedArtworks() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM artworks WHERE published IS true").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func getRecordStringField(record *core.Record, field string) (string, error) {
	value, ok := record.Get(field).(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", field)
	}

	return value, nil
}
