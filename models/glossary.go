package models

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type GlossaryItem struct {
	models.BaseModel
	Expression string `db:"expression" json:"expression"`
	Definition string `db:"definition" json:"definition"`
}

var _ models.Model = (*GlossaryItem)(nil)

func (m *GlossaryItem) TableName() string {
	return "glossary" // the name of your collection
}

func GlossaryQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&GlossaryItem{})
}

func GetGlossaryItems(dao *daos.Dao) ([]*GlossaryItem, error) {
	var c []*GlossaryItem
	err := GlossaryQuery(dao).OrderBy("expression asc").All(&c)
	return c, err
}
