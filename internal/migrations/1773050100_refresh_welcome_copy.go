package migrations

import (
	"encoding/json"
	"fmt"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		data, err := assets.InternalFiles.ReadFile("reference/strings.json")
		if err != nil {
			return err
		}

		var stringsData []PublicString
		if err := json.Unmarshal(data, &stringsData); err != nil {
			return err
		}

		var welcomeContent string
		for _, record := range stringsData {
			if record.Name == "welcome" {
				welcomeContent = record.Content
				break
			}
		}

		if welcomeContent == "" {
			return fmt.Errorf("welcome content not found in reference strings")
		}

		record, err := app.FindFirstRecordByData("strings", "name", "welcome")
		if err != nil {
			return err
		}

		record.Set("content", welcomeContent)

		return app.Save(record)
	}, func(app core.App) error {
		return nil
	})
}
