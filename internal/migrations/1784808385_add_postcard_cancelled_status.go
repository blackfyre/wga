package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// init registers the cancelled postcard status migration.
func init() {
	m.Register(func(app core.App) error {
		postcards, err := app.FindCollectionByNameOrId("postcards")
		if err != nil {
			return err
		}
		status, ok := postcards.Fields.GetByName("status").(*core.SelectField)
		if !ok {
			return fmt.Errorf("postcards.status must be a select field")
		}
		for _, value := range status.Values {
			if value == "cancelled" {
				return nil
			}
		}
		status.Values = append(status.Values, "cancelled")
		return app.Save(postcards)
	}, func(app core.App) error {
		postcards, err := app.FindCollectionByNameOrId("postcards")
		if err != nil {
			return err
		}
		status, ok := postcards.Fields.GetByName("status").(*core.SelectField)
		if !ok {
			return fmt.Errorf("postcards.status must be a select field")
		}
		cancelled, err := app.FindRecordsByFilter("postcards", `status = 'cancelled'`, "", 1, 0)
		if err != nil {
			return err
		}
		if len(cancelled) != 0 {
			return fmt.Errorf("cannot remove cancelled postcard status while cancelled postcards exist")
		}
		values := make([]string, 0, len(status.Values))
		for _, value := range status.Values {
			if value != "cancelled" {
				values = append(values, value)
			}
		}
		status.Values = values
		return app.Save(postcards)
	})
}
