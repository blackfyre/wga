package models

import (
	"github.com/pocketbase/pocketbase/models"
)

type Feedback struct {
	models.BaseModel
	Name    string `db:"name" json:"name"`
	Message string `db:"message" json:"message"`
	Email   string `db:"email" json:"email"`
	ReferTo string `db:"refer_to" json:"refer_to"`
	Handled bool   `db:"handled" json:"handled"`
}

var _ models.Model = (*Feedback)(nil)

func (m *Feedback) TableName() string {
	return "feedbacks" // the name of your collection
}
