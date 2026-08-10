package migrations

import (
	"errors"

	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(seedCurrentSyntheticData, func(core.App) error {
		return errors.New("synthetic bootstrap data cannot be rolled back safely")
	})
}

func seedCurrentSyntheticData(app core.App) error {
	if err := seed.RequireEmptyApplicationDatabase(app); err != nil {
		if errors.Is(err, seed.ErrApplicationRecords) {
			return nil
		}

		return err
	}

	configuration, err := configuredMigrations()
	if err != nil {
		return err
	}

	return seed.Import(app, configuration.SeedSQLitePath())
}
