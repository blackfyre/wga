package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type ArtType struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
}

var _ models.Model = (*ArtType)(nil)

func (m *ArtType) TableName() string {
	return "art_types" // the name of your collection
}
