package repositories

import "github.com/pocketbase/pocketbase"

type LandingRepository struct {
	app *pocketbase.PocketBase
}

type countRow struct {
	Count int `db:"c"`
}

func NewLandingRepository(app *pocketbase.PocketBase) *LandingRepository {
	return &LandingRepository{app: app}
}

func (r *LandingRepository) CountPublishedArtists() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM Artists WHERE published IS true").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func (r *LandingRepository) CountPublishedArtworks() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM Artworks WHERE published IS true").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}

func (r *LandingRepository) CountSchools() (int, error) {
	row := countRow{}
	err := r.app.DB().NewQuery("SELECT COUNT(*) as c FROM Schools").One(&row)
	if err != nil {
		return 0, err
	}

	return row.Count, nil
}
