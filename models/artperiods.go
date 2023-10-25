package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type ArtPeriod struct {
	models.BaseModel
	Name        string `db:"name" json:"name"`
	Start       int    `db:"start" json:"start"`
	End         int    `db:"end" json:"end"`
	Description string `db:"description" json:"description"`
}

var _ models.Model = (*ArtPeriod)(nil)

func (m *ArtPeriod) TableName() string {
	return "art_periods" // the name of your collection
}
