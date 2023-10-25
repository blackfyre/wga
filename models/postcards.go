package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Postcard struct {
	models.BaseModel
	Name string `db:"name" json:"name"`
}

var _ models.Model = (*Postcard)(nil)

func (m *Postcard) TableName() string {
	return "postcards" // the name of your collection
}
