package main

import (
	"log"
	"os"

	"github.com/blackfyre/wga/crontab"
	"github.com/blackfyre/wga/handlers"
	"github.com/blackfyre/wga/hooks"
	_ "github.com/blackfyre/wga/migrations"
	"github.com/labstack/echo/v5/middleware"

	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/seed"
	"github.com/blackfyre/wga/utils/sitemap"
	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"
)

func main() {

	_ = godotenv.Load()

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: "./wga_data",
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
			TokenLookup: "header:X-XSRF-TOKEN",
		}))

		return nil
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

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "generate-music-urls",
		Short: "Generate music urls",
		Run: func(cmd *cobra.Command, args []string) {
			utils.ParseMusicListToUrls("./assets/reference/musics.json")
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
