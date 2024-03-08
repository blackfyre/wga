package models

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// WIP - this is a work in progress
type Artist struct {
	models.BaseModel
	Id           string `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Slug         string `db:"slug" json:"slug"`
	Bio          string `db:"bio" json:"bio"`
	YearOfBirth  int    `db:"year_of_birth" json:"year_of_birth"`
	YearOfDeath  int    `db:"year_of_death" json:"year_of_death"`
	PlaceOfBirth string `db:"place_of_birth" json:"place_of_birth"`
	PlaceOfDeath string `db:"place_of_death" json:"place_of_death"`
	Published    bool   `db:"published" json:"published"`
	School       string `db:"school" json:"school"`
	Profession   string `db:"profession" json:"profession"`
}

var _ models.Model = (*Artist)(nil)

func (m *Artist) TableName() string {
	return "artists" // the name of your collection
}

// ArtistQuery returns a new dbx.SelectQuery for the Artist model.
func ArtistQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Artist{})
}

// GetArtists retrieves all art forms from the database and returns them as a slice of Artist pointers.
// It takes a dao object as a parameter and returns the slice of Artist pointers and an error (if any).
func GetArtists(dao *daos.Dao) ([]*Artist, error) {
	var c []*Artist
	err := ArtistQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}

// GetArtistBySlug retrieves an art form from the database by its slug.
// It takes a dao object and a slug string as arguments and returns a pointer to the retrieved Artist object and an error (if any).
func GetArtistBySlug(dao *daos.Dao, slug string) (*Artist, error) {
	var c Artist
	err := ArtistQuery(dao).AndWhere(dbx.NewExp("LOWER(slug)={:slug}", dbx.Params{
		"slug": slug,
	})).
		Limit(1).
		One(&c)
	return &c, err
}

func GetArtistByNameLike(dao *daos.Dao, name string) ([]*Artist, error) {
	var c []*Artist
	err := ArtistQuery(dao).AndWhere(dbx.NewExp("LOWER(name) LIKE {:name}", dbx.Params{
		"name": "%" + name + "%",
	})).All(&c)
	return c, err
}

func GetArtistById(dao *daos.Dao, id string) (*Artist, error) {
	var c Artist
	err := ArtistQuery(dao).AndWhere(dbx.NewExp("id={:id}", dbx.Params{
		"id": id,
	})).
		Limit(1).
		One(&c)
	return &c, err
}
