package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const persistedLogMaxDays = 7

func init() {
	m.Register(boundLogPersistence, nil)
}

// boundLogPersistence caps persisted log retention so that a burst of failed
// requests or protection events ages out within a week. A shorter retention,
// including 0 (persistence disabled), is an operator choice and is kept.
func boundLogPersistence(app core.App) error {
	settings := app.Settings()
	if settings.Logs.MaxDays <= persistedLogMaxDays {
		return nil
	}
	settings.Logs.MaxDays = persistedLogMaxDays

	return app.Save(settings)
}
