package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	// Keep the historical filename registered for databases that applied its
	// earlier source-schema version. The corrective migration removes that schema.
	m.Register(func(core.App) error {
		return nil
	}, func(core.App) error {
		return nil
	})
}
