package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {

	m.Register(func(app core.App) error {
		migrations, err := configuredMigrations()
		if err != nil {
			return err
		}

		administrator, err := migrations.Administrator()
		if err != nil {
			return fmt.Errorf("default admin migration: %w", err)
		}

		if administrator.Enabled {

			superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
			if err != nil {
				return err
			}

			record := core.NewRecord(superusers)

			record.Set("email", administrator.Email.Address)
			record.Set("password", administrator.Password.Value())

			return app.Save(record)
		}

		return nil

	}, func(app core.App) error {
		migrations, err := configuredMigrations()
		if err != nil {
			return err
		}

		administrator, err := migrations.Administrator()
		if err != nil {
			return fmt.Errorf("default admin migration: %w", err)
		}
		if !administrator.Enabled {
			return nil
		}

		record, _ := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, administrator.Email.Address)
		if record == nil {
			return nil // probably already deleted
		}

		return app.Delete(record)
	})
}
