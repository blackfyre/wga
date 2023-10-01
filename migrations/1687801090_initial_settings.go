package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		settings, _ := dao.FindSettings()
		settings.Meta.AppName = "Web Gallery of Art"
		settings.Logs.MaxDays = 30
		settings.Meta.SenderName = "Web Gallery of Art"
		settings.Meta.SenderAddress = "info@wga.hu"

		return dao.SaveSettings(settings)
	}, nil)
}
