package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type GuestbookEntry struct {
	models.BaseModel
	Name     string `db:"name" json:"name"`
	Message  string `db:"message" json:"message"`
	Email    string `db:"email" json:"email"`
	Location string `db:"location" json:"location"`
}

var _ models.Model = (*GuestbookEntry)(nil)

func (m *GuestbookEntry) TableName() string {
	return "guestbook" // the name of your collection
}
