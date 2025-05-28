package migrations

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {

	_ = godotenv.Load()

	m.Register(func(app core.App) error {
		settings := app.Settings()

		settings.Meta.AppName = "Web Gallery of Art"
		settings.Logs.MaxDays = 30
		settings.Meta.SenderName = "Web Gallery of Art"
		settings.Meta.SenderAddress = "info@wga.hu"
		settings.Meta.AppURL = os.Getenv("WGA_PROTOCOL") + "://" + os.Getenv("WGA_HOSTNAME")
		settings.S3.Enabled = true
		settings.S3.Endpoint = os.Getenv("WGA_S3_ENDPOINT")
		settings.S3.AccessKey = os.Getenv("WGA_S3_ACCESS_KEY")
		settings.S3.Bucket = os.Getenv("WGA_S3_BUCKET")
		settings.S3.Secret = os.Getenv("WGA_S3_ACCESS_SECRET")
		settings.S3.Region = os.Getenv("WGA_S3_REGION")
		settings.S3.ForcePathStyle = true
		settings.Meta.SenderName = os.Getenv("WGA_SENDER_NAME")
		settings.Meta.SenderAddress = os.Getenv("WGA_SENDER_ADDRESS")

		settings.SMTP.Enabled = true
		settings.SMTP.Host = os.Getenv("WGA_SMTP_HOST")
		settings.SMTP.Port, _ = strconv.Atoi(os.Getenv("WGA_SMTP_PORT"))
		settings.SMTP.Username = os.Getenv("WGA_SMTP_USERNAME")
		settings.SMTP.Password = os.Getenv("WGA_SMTP_PASSWORD")

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
