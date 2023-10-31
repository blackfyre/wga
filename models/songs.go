package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Song struct {
	models.BaseModel
	Title  string `db:"title" json:"title"`
	URL    string `db:"url" json:"url"`
	Source []string `db:"source" json:"source"`
	ComposerID uint `db:"composer_id" json:"composer_id"`
}

var _ models.Model = (*Song)(nil)


func (m *Song) TableName() string {
	return "Song" // the name of your collection
}