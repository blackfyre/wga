package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type ArtForm struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
}

var _ models.Model = (*ArtForm)(nil)

func (m *ArtForm) TableName() string {
	return "art_forms" // the name of your collection
}
