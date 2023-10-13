package migrations

import (
	"log"
	"os"
	"strconv"

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
		settings.Meta.SenderName = os.Getenv("WGA_SENDER_NAME")
		settings.Meta.SenderAddress = os.Getenv("WGA_SENDER_ADDRESS")
		settings.Smtp.Enabled = true
		settings.Smtp.Host = os.Getenv("WGA_SMTP_HOST")
		settings.Smtp.Port, _ = strconv.Atoi(os.Getenv("WGA_SMTP_PORT"))
		settings.Smtp.Username = os.Getenv("WGA_SMTP_USERNAME")
		settings.Smtp.Password = os.Getenv("WGA_SMTP_PASSWORD")

		return dao.SaveSettings(settings)
	}, nil)
}
