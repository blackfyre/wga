package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type ArtPeriod struct {
	models.BaseModel
	Name        string `db:"art_periods_name" json:"name"`
	Start       int    `db:"art_periods_start" json:"start"`
	End         int    `db:"art_periods_end" json:"end"`
	Description string `db:"art_periods_description" json:"description"`
}

var _ models.Model = (*ArtPeriod)(nil)

func (m *ArtPeriod) TableName() string {
	return "art_periods" // the name of your collection
}
