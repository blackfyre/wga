package main

import (
	"log"

	"blackfyre.ninja/wga/handlers"
	"blackfyre.ninja/wga/hooks"
	_ "blackfyre.ninja/wga/migrations"
	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	handlers.RegisterHandlers(app)
	hooks.RegisterHooks(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
