package models

import (
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
