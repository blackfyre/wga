package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Glossary struct {
	models.BaseModel
	Expression string `db:"expression" json:"expression"`
	Definition string `db:"definition" json:"definition"`
}

var _ models.Model = (*Glossary)(nil)

func (m *Glossary) TableName() string {
	return "glossary" // the name of your collection
}
