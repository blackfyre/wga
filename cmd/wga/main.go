package main

import (
	"errors"
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
	itineraryhandlers "github.com/blackfyre/wga/internal/handlers/itineraries"
	"github.com/blackfyre/wga/internal/hooks"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/migrations"
	"github.com/blackfyre/wga/internal/observability"
	"github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/requesttrust"

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
		serverConfig, err = serverConfigFor(runtimeConfig)
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
	tracer := observability.Tracer{}

	if err := migrations.Configure(runtimeConfig.Migrations()); err != nil {
		log.Fatal(err)
	}

	if capability == commandNeedsServer {
		utils.ConfigurePublicURL(serverConfig.PublicURL)
		logging.RegisterRequestIDMiddleware(app)
		tracer, err = observability.ConfigureTracing(serverConfig.Environment, app.Logger())
		if err != nil {
			log.Fatal(err)
		}
		tracer.Register(app)
		monitor = observability.Configure(serverConfig.Sentry, serverConfig.Environment, app.Logger())
		monitor.Register(app)
		monitor.RegisterTestRoute(app, serverConfig.Environment)
		contributorStore, err := contributors.NewStore(app)
		if err != nil {
			log.Fatal(err)
		}
		contributorProvider := contributors.NewGitHubProvider(&http.Client{Timeout: 10 * time.Second})
		captchaVerifier := antiabuse.NewRecaptchaVerifier(&http.Client{Timeout: 5 * time.Second}, serverConfig.Captcha.Secret())
		clientIdentity := requesttrust.New(requesttrust.Source(serverConfig.ClientIPSource))
		itineraryPolicy, err := itinerarySecurityPolicy(serverConfig, clientIdentity)
		if err != nil {
			log.Fatal(err)
		}
		if err := handlers.RegisterHandlers(app, serverConfig.Environment, serverConfig.Captcha, serverConfig.Postcards.TokenKeyring(), contributorStore, captchaVerifier, itineraryPolicy, clientIdentity); err != nil {
			log.Fatal(err)
		}
		crontab.RegisterCronJobs(app, serverConfig.Postcards, serverConfig.Sitemap(), contributors.NewRefreshJob(app, contributorProvider, contributorStore))
	}

	hooks.RegisterHooks(app)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Enable auto creation of migration files when making collection changes in the Admin UI
		// (the `isGoRun` check is to enable it only during development)
		Automigrate: false,
	})
	postcards.RegisterCommands(app, runtimeConfig.PostcardTokenKeyring)

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
		flushTracing(tracer, app)
		monitor.Flush()
		log.Fatal(err)
	}
	flushTracing(tracer, app)
	monitor.Flush()
}

func flushTracing(tracer observability.Tracer, app *pocketbase.PocketBase) {
	if err := tracer.Shutdown(); err != nil {
		app.Logger().Error("OpenTelemetry trace flush failed",
			"event", "observability.otel.shutdown_failed",
			"error_type", logging.ErrorType(err),
		)
	}
}

func serverConfigFor(runtimeConfig config.Config) (config.Server, error) {
	serverConfig, serverErr := runtimeConfig.Server()
	_, keyringErr := runtimeConfig.PostcardTokenKeyring()

	return serverConfig, errors.Join(serverErr, keyringErr)
}

// itinerarySecurityPolicy constructs the visitor-itinerary security policy from
// the validated server configuration. The production __Host- cookie is always
// Secure; the HTTP development cookie is only opted in for development and test
// environments. The trusted client identity resolver is selected solely by the
// configured WGA_CLIENT_IP_SOURCE, never by backend TLS state.
func itinerarySecurityPolicy(serverConfig config.Server, clientIdentity requesttrust.Resolver) (itineraryhandlers.SecurityPolicy, error) {
	policy := itineraryhandlers.SecurityPolicy{
		CanonicalOrigin: serverConfig.PublicURL.String(),
		Production: itineraryhandlers.CookiePolicy{
			Name:   itineraryworkflow.ProductionSessionCookieName,
			Secure: true,
		},
		TrustedClientID: itineraryhandlers.TrustedClientID(clientIdentity),
	}

	if serverConfig.Environment.IsDevelopment() || serverConfig.Environment == config.EnvironmentTest {
		policy.Development = itineraryhandlers.CookiePolicy{
			Name:   itineraryworkflow.DevelopmentSessionCookieName,
			Secure: false,
		}
	}

	if err := policy.Validate(); err != nil {
		return itineraryhandlers.SecurityPolicy{}, err
	}

	return policy, nil
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
		case "serve":
			return commandNeedsServer
		default:
			return commandNeedsNothing
		}
	}

	return commandNeedsServer
}
