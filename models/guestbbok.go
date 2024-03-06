package models

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type GuestbookEntry struct {
	models.BaseModel
	Name     string `db:"name" json:"name"`
	Message  string `db:"message" json:"message"`
	Email    string `db:"email" json:"email"`
	Location string `db:"location" json:"location"`
	Created  string `db:"created" json:"created"`
}

var _ models.Model = (*GuestbookEntry)(nil)

func (m *GuestbookEntry) TableName() string {
	return "Guestbook" // the name of your collection
}

func GuestbookQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&GuestbookEntry{})
}

func FindEntriesForYear(dao *daos.Dao, year string) ([]*GuestbookEntry, error) {
	var entries []*GuestbookEntry

	err := GuestbookQuery(dao).AndWhere(dbx.Like("created", year)).OrderBy("created DESC").All(&entries)

	return entries, err
}
