package migrations

import (
	"errors"

	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(importProductionGuestbook, func(core.App) error {
		return errors.New("production guestbook archive import cannot be rolled back safely")
	})
}

func importProductionGuestbook(app core.App) error {
	configuration, err := configuredMigrations()
	if err != nil {
		return err
	}

	return seed.PromoteProductionGuestbookEntries(app, configuration.SeedSQLitePath())
}
