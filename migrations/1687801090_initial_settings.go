package migrations

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		settings, _ := dao.FindSettings()
		settings.Meta.AppName = "Web Gallery of Art"
		settings.Logs.MaxDays = 30
		settings.Meta.SenderName = "Web Gallery of Art"
		settings.Meta.SenderAddress = "info@wga.hu"
		settings.S3.Enabled = true
		settings.S3.Endpoint = os.Getenv("WGA_S3_ENDPOINT")
		settings.S3.AccessKey = os.Getenv("WGA_S3_ACCESS_KEY")
		settings.S3.Bucket = os.Getenv("WGA_S3_BUCKET")
		settings.S3.Secret = os.Getenv("WGA_S3_ACCESS_SECRET")
		settings.S3.Region = os.Getenv("WGA_S3_REGION")
		settings.S3.ForcePathStyle = true

		return dao.SaveSettings(settings)
	}, nil)
}
