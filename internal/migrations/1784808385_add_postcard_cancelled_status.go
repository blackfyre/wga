package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		postcards, err := app.FindCollectionByNameOrId("postcards")
		if err != nil {
			return err
		}
		status, ok := postcards.Fields.GetByName("status").(*core.SelectField)
		if !ok {
			return nil
		}
		for _, value := range status.Values {
			if value == "cancelled" {
				return nil
			}
		}
		status.Values = append(status.Values, "cancelled")
		return app.Save(postcards)
	}, nil)
}
