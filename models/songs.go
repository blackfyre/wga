package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Music_song struct {
	models.BaseModel
	Title  		string `db:"title" json:"title"`
	URL    		string `db:"url" json:"url"`
	Source 		string `db:"source" json:"source"`
	ComposerID  string `db:"composer_id" json:"composer_id"` // foreign key
}

var _ models.Model = (*Music_song)(nil)


func (m *Music_song) TableName() string {
	return "music_song" // the name of your collection
}