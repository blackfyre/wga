package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Postcard struct {
	models.BaseModel
	SenderName  string `db:"sender_name" json:"sender_name"`
	SenderEmail string `db:"sender_email" json:"sender_email"`
	Recipients  string `db:"recipients" json:"recipients"`
	Message     string `db:"message" json:"message"`
}

var _ models.Model = (*Postcard)(nil)

func (m *Postcard) TableName() string {
	return "postcards" // the name of your collection
}
