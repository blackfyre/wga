package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Composer struct {
	models.BaseModel
	Name     string `db:"name" json:"name"`
	Date     string `db:"date" json:"date"`
	Language string `db:"language" json:"language"`
	Century  string `db:"century" json:"century"`
	Songs    []Song `db:"-" json:"songs" goqu:"skipinsert,skipupdate"`
}

var _ models.Model = (*Composer)(nil)

func (m *Composer) TableName() string {
	return "Composer" // the name of your collection
}
