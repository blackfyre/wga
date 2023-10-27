package models

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type ArtType struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
	Slug string `db:"slug" json:"slug"`
}

var _ models.Model = (*ArtType)(nil)

func (m *ArtType) TableName() string {
	return "art_types" // the name of your collection
}

// ArtTypeQuery returns a new SelectQuery for the ArtType model.
func ArtTypeQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&ArtType{})
}

// GetArtTypes retrieves all art types from the database and returns them as a slice of ArtType pointers.
// It takes a pointer to a dao object as an argument and returns the slice of ArtType pointers and an error (if any).
func GetArtTypes(dao *daos.Dao) ([]*ArtType, error) {
	var c []*ArtType
	err := ArtTypeQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}

// GetArtTypeBySlug retrieves an ArtType from the database by its slug.
// It takes a dao object and a slug string as parameters.
// It returns a pointer to the retrieved ArtType and an error if any.
func GetArtTypeBySlug(dao *daos.Dao, slug string) (*ArtType, error) {
	var c ArtType
	err := ArtTypeQuery(dao).AndWhere(dbx.NewExp("LOWER(slug)={:slug}", dbx.Params{
		"slug": slug,
	})).
		Limit(1).
		One(&c)
	return &c, err
}
