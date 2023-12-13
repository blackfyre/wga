package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Music_composer struct {
	models.BaseModel
	ID	     string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Date     string `db:"date" json:"date"`
	Language string `db:"language" json:"language"`
	Century  string `db:"century" json:"century"`
	Songs    []Music_song `db:"songs" json:"songs"`
}

var _ models.Model = (*Music_composer)(nil)

func (m *Music_composer) TableName() string {
	return "music_composer" // the name of your collection
}
