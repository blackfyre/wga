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

func ArtTypeQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&ArtType{})
}

func GetArtTypes(dao *daos.Dao) ([]*ArtType, error) {
	var c []*ArtType
	err := ArtTypeQuery(dao).OrderBy("name asc").All(&c)
	return c, err
}
