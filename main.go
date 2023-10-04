package main

import (
	"log"

	"blackfyre.ninja/wga/handlers"
	"blackfyre.ninja/wga/hooks"
	_ "blackfyre.ninja/wga/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	app := pocketbase.New()

	handlers.RegisterHandlers(app)
	hooks.RegisterHooks(app)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		// (the isGoRun check is to enable it only during development)
		Automigrate: false,
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
