package main

import (
	"log"
	"os"
	"strings"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/crontab"
	"github.com/blackfyre/wga/internal/handlers"
	"github.com/blackfyre/wga/internal/hooks"
	"github.com/blackfyre/wga/internal/migrations"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/blackfyre/wga/internal/utils/sitemap"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"
)

type commandCapability uint8

const (
	commandNeedsNothing commandCapability = iota
	commandNeedsServer
	commandNeedsSitemap
)

func main() {
	runtimeConfig, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	capability := commandCapabilityFor(os.Args[1:])
	var serverConfig config.Server
	var sitemapConfig config.Sitemap

	switch capability {
	case commandNeedsServer:
		serverConfig, err = runtimeConfig.Server()
	case commandNeedsSitemap:
		sitemapConfig, err = runtimeConfig.Sitemap()
	}
	if err != nil {
		log.Fatal(err)
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: "./wga_data",
	})

	if err := migrations.Configure(runtimeConfig.Migrations()); err != nil {
		log.Fatal(err)
	}

	if capability == commandNeedsServer {
		utils.ConfigurePublicURL(serverConfig.PublicURL)
		handlers.RegisterHandlers(app, serverConfig.Captcha)
		crontab.RegisterCronJobs(app, serverConfig.Postcards, serverConfig.Sitemap())
	}

	hooks.RegisterHooks(app)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Enable auto creation of migration files when making collection changes in the Admin UI
		// (the `isGoRun` check is to enable it only during development)
		Automigrate: false,
	})

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "generate-sitemap",
		Short: "Generate sitemap",
		Run: func(cmd *cobra.Command, args []string) {
			sitemap.GenerateSiteMap(app, sitemapConfig)
		},
	})

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "generate-music-urls",
		Short: "Generate music urls",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := utils.ParseMusicListToUrls("./assets/reference/musics.json"); err != nil {
				log.Fatal(err)
			}
		},
	})

	if runtimeConfig.Environment().IsDevelopment() {
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

	var replaceMinimalSeed bool
	seedDataCommand := &cobra.Command{
		Use:   "seed:data",
		Short: "Import records from the configured seed SQLite database",
		RunE: func(cmd *cobra.Command, args []string) error {
			seedConfig := runtimeConfig.Seed()
			return seed.SeedDatabase(app, seed.SourceOptions{
				Environment: seedConfig.Environment,
				SQLitePath:  seedConfig.SQLitePath,
				StoragePath: seedConfig.StoragePath,
				ReplaceMinimal: replaceMinimalSeed,
			})
		},
	}
	seedDataCommand.Flags().BoolVar(&replaceMinimalSeed, "replace-minimal", false, "replace the known minimal starter dataset")
	app.RootCmd.AddCommand(seedDataCommand)

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "seed:storage",
		Short: "Upload seed storage assets for imported records",
		RunE: func(cmd *cobra.Command, args []string) error {
			seedConfig := runtimeConfig.Seed()
			return seed.SeedStorage(app, seed.SourceOptions{
				Environment: seedConfig.Environment,
				SQLitePath:  seedConfig.SQLitePath,
				StoragePath: seedConfig.StoragePath,
			})
		},
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func commandCapabilityFor(args []string) commandCapability {
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "--version", "-v", "help", "version":
			return commandNeedsNothing
		}
	}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--dir" || arg == "--encryptionEnv" || arg == "--queryTimeout" || arg == "--origins" || arg == "--http" || arg == "--https" {
			index++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}

		switch arg {
		case "generate-sitemap":
			return commandNeedsSitemap
		case "migrate", "generate-music-urls", "seed:data", "seed:images", "seed:storage", "superuser":
			return commandNeedsNothing
		case "serve":
			return commandNeedsServer
		default:
			return commandNeedsNothing
		}
	}

	return commandNeedsServer
}
