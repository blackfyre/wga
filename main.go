package main

import (
	"log"
	"os"
	"strings"

	"blackfyre.ninja/wga/crontab"
	"blackfyre.ninja/wga/handlers"
	"blackfyre.ninja/wga/hooks"
	_ "blackfyre.ninja/wga/migrations"
	"blackfyre.ninja/wga/utils/seed"
	"blackfyre.ninja/wga/utils/sitemap"
	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"
)

func main() {

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

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "generate-sitemap",
		Short: "Generate sitemap",
		Run: func(cmd *cobra.Command, args []string) {
			sitemap.GenerateSiteMap(app)
		},
	})

	if os.Getenv("WGA_ENV") == "development" {
		app.RootCmd.AddCommand(&cobra.Command{
			Use:   "seed:images",
			Short: "Seed images to the specified S3 bucket",
			Run: func(cmd *cobra.Command, args []string) {
				err := seed.SeedImages(app)

				if err != nil {
					log.Fatal(err)
				}

				log.Println("Done seeding images")

			},
		})
	}

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
