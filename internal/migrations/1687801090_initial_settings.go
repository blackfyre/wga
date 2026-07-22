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

		settingsConfig, err := migrations.InitialSettings()
		if err != nil {
			return fmt.Errorf("initial settings migration: %w", err)
		}

		settings := app.Settings()

		settings.Meta.AppName = "Web Gallery of Art"
		settings.Logs.MaxDays = 30
		settings.Meta.SenderName = "Web Gallery of Art"
		settings.Meta.SenderAddress = "info@wga.hu"
		settings.Meta.AppURL = settingsConfig.PublicURL.String()
		if settingsConfig.Storage.Enabled {
			settings.S3.Enabled = true
			settings.S3.Endpoint = settingsConfig.Storage.Endpoint.String()
			settings.S3.AccessKey = settingsConfig.Storage.AccessKey
			settings.S3.Bucket = settingsConfig.Storage.Bucket
			settings.S3.Secret = settingsConfig.Storage.AccessSecret.Value()
			settings.S3.Region = settingsConfig.Storage.Region
			settings.S3.ForcePathStyle = true
		}
		settings.Meta.SenderName = settingsConfig.Mail.Sender.Name
		settings.Meta.SenderAddress = settingsConfig.Mail.Sender.Address.Address

		settings.SMTP.Enabled = true
		settings.SMTP.Host = settingsConfig.Mail.SMTP.Host
		settings.SMTP.Port = settingsConfig.Mail.SMTP.Port
		settings.SMTP.Username = settingsConfig.Mail.SMTP.Username
		settings.SMTP.Password = settingsConfig.Mail.SMTP.Password.Value()

		return app.Save(settings)
	}, nil)
}

func deleteCollection(app core.App, name string) error {
	collection, err := app.FindCollectionByNameOrId(name)

	if err != nil {
		return nil
	}

	err = app.Delete(collection)

	if err != nil {
		app.Logger().Error("Error deleting collection %s: %v", name, err)
	}

	return nil
}
