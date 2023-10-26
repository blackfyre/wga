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

func ArtFormQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&ArtForm{})
}

func GetArtForms(dao *daos.Dao) ([]*ArtForm, error) {
	var c []*ArtForm
	err := ArtFormQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}
