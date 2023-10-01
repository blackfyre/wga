package main

import (
	"blackfyre.ninja/wga/handlers"
	"blackfyre.ninja/wga/hooks"
	_ "blackfyre.ninja/wga/migrations"
	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	handlers.RegisterHome(app)
	handlers.RegisterArtist(app)
	hooks.RegisterHooks(app)

	// if err := app.Start(); err != nil {
	// 	log.Fatal(err)
	// }
}
