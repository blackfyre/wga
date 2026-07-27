package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/antiabuse"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/contributors"
	"github.com/blackfyre/wga/internal/crontab"
	"github.com/blackfyre/wga/internal/handlers"
	"github.com/blackfyre/wga/internal/hooks"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/migrations"
	"github.com/blackfyre/wga/internal/observability"
	"github.com/blackfyre/wga/internal/postcards"

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
	monitor := observability.Monitor{}

	if err := migrations.Configure(runtimeConfig.Migrations()); err != nil {
		log.Fatal(err)
	}

	if capability == commandNeedsServer {
		utils.ConfigurePublicURL(serverConfig.PublicURL)
		logging.RegisterRequestIDMiddleware(app)
		monitor = observability.Configure(serverConfig.Sentry, serverConfig.Environment, app.Logger())
		monitor.Register(app)
		contributorStore, err := contributors.NewStore(app)
		if err != nil {
			log.Fatal(err)
		}
		contributorProvider := contributors.NewGitHubProvider(&http.Client{Timeout: 10 * time.Second})
		captchaVerifier := antiabuse.NewRecaptchaVerifier(&http.Client{Timeout: 5 * time.Second}, serverConfig.Captcha.Secret())
		handlers.RegisterHandlers(app, serverConfig.Captcha, contributorStore, captchaVerifier)
		crontab.RegisterCronJobs(app, serverConfig.Postcards, serverConfig.Sitemap(), contributors.NewRefreshJob(app, contributorProvider, contributorStore))
	}

	hooks.RegisterHooks(app)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Enable auto creation of migration files when making collection changes in the Admin UI
		// (the `isGoRun` check is to enable it only during development)
		Automigrate: false,
	})
	postcards.RegisterCommands(app)
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "sentry-test",
		Short: "Send a Sentry test event in a non-production environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverConfig.Environment == config.EnvironmentProduction {
				return errors.New("sentry-test is unavailable in production")
			}
			if !monitor.CaptureMessage("It works!") {
				return errors.New("Sentry monitoring is disabled")
			}

			monitor.Flush()
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Sentry test event sent.")
			return err
		},
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

	if err := app.Start(); err != nil {
		monitor.Flush()
		log.Fatal(err)
	}
	monitor.Flush()
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
		case "migrate", "generate-music-urls", "seed:images", "superuser", "postcards":
			return commandNeedsNothing
		case "serve", "sentry-test":
			return commandNeedsServer
		default:
			return commandNeedsNothing
		}
	}

	return commandNeedsServer
}
