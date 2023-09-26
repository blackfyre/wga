package migrations

import (
	"github.com/pocketbase/dbx"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		// add up queries...
		columns := map[string]string{
			"id":       "text",
			"created":  "text",
			"updated":  "text",
			"message":  "text",
			"name":     "text",
			"email":    "text",
			"location": "text",
		}

		q := db.CreateTable("guestbook", columns)
		_, err := q.Execute()

		return err
	}, func(db dbx.Builder) error {
		// add down queries...

		q := db.DropTable("guestbook")
		_, err := q.Execute()

		return err
	})
}
