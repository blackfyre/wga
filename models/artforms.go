package models

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type ArtForm struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
	Slug string `db:"slug" json:"slug"`
}

var _ models.Model = (*ArtForm)(nil)

func (m *ArtForm) TableName() string {
	return "art_forms" // the name of your collection
}

// ArtFormQuery returns a new dbx.SelectQuery for the ArtForm model.
func ArtFormQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&ArtForm{})
}

// GetArtForms retrieves all art forms from the database and returns them as a slice of ArtForm pointers.
// It takes a dao object as a parameter and returns the slice of ArtForm pointers and an error (if any).
func GetArtForms(dao *daos.Dao) ([]*ArtForm, error) {
	var c []*ArtForm
	err := ArtFormQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}

// GetArtFormBySlug retrieves an art form from the database by its slug.
// It takes a dao object and a slug string as arguments and returns a pointer to the retrieved ArtForm object and an error (if any).
func GetArtFormBySlug(dao *daos.Dao, slug string) (*ArtForm, error) {
	var c ArtForm
	err := ArtFormQuery(dao).AndWhere(dbx.NewExp("LOWER(slug)={:slug}", dbx.Params{
		"slug": slug,
	})).
		Limit(1).
		One(&c)
	return &c, err
}
