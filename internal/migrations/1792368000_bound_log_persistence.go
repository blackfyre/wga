package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const persistedLogMaxDays = 7

func init() {
	m.Register(boundLogPersistence, nil)
}

// boundLogPersistence shortens persisted log retention so that a burst of
// failed requests or protection events ages out within a week.
func boundLogPersistence(app core.App) error {
	settings := app.Settings()
	settings.Logs.MaxDays = persistedLogMaxDays

	return app.Save(settings)
}
