package main

import (
	"log"
	"os"
	"strings"

	"blackfyre.ninja/wga/crontab"
	"blackfyre.ninja/wga/handlers"
	"blackfyre.ninja/wga/hooks"
	_ "blackfyre.ninja/wga/migrations"

	// "blackfyre.ninja/wga/utils"
	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {

	// utils.ParseMusicListToUrls("./assets/reference/musics.json")
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDebug:   strings.HasPrefix(os.Args[0], os.TempDir()),
		DefaultDataDir: "./wga_data",
	})

	handlers.RegisterHandlers(app)
	hooks.RegisterHooks(app)
	crontab.RegisterCronJobs(app)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		// (the isGoRun check is to enable it only during development)
		Automigrate: false,
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
