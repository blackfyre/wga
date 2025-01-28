package migrations

import (
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {

	email := os.Getenv("WGA_ADMIN_EMAIL")
	password := os.Getenv("WGA_ADMIN_PASSWORD")

	m.Register(func(app core.App) error {

		if email != "" && password != "" {

			superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
			if err != nil {
				return err
			}

			record := core.NewRecord(superusers)

			// note: the values can be eventually loaded via os.Getenv(key)
			// or from a special local config file
			record.Set("email", email)
			record.Set("password", password)

			return app.Save(record)
		}

		return nil

	}, func(app core.App) error {
		record, _ := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
		if record == nil {
			return nil // probably already deleted
		}

		return app.Delete(record)
	})
}
