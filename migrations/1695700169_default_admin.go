package migrations

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {

	_ = godotenv.Load()

	email := os.Getenv("WGA_ADMIN_EMAIL")
	password := os.Getenv("WGA_ADMIN_PASSWORD")

	m.Register(func(db dbx.Builder) error {

		if email != "" && password != "" {
			dao := daos.New(db)

			admin := &models.Admin{}
			admin.Email = email
			admin.SetPassword(password)

			return dao.SaveAdmin(admin)
		}

		return nil

	}, func(db dbx.Builder) error {
		if email != "" {
			dao := daos.New(db)

			admin, _ := dao.FindAdminByEmail(email)
			if admin != nil {
				return dao.DeleteAdmin(admin)
			}
		}

		// already deleted
		return nil
	})
}
