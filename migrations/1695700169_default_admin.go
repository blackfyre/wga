package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		admin := &models.Admin{}
		admin.Email = "root@wga.hu"
		admin.SetPassword("Testing1234567890")

		return dao.SaveAdmin(admin)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		admin, _ := dao.FindAdminByEmail("test@example.com")
		if admin != nil {
			return dao.DeleteAdmin(admin)
		}

		// already deleted
		return nil
	})
}
