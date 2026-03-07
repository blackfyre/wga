package landing

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type Content struct {
	FieldName string `db:"name" json:"name"`
	Content   string `db:"content" json:"content"`
}

const landingCacheTTL = 15 * time.Minute

// getWelcomeContent retrieves the welcome content from the application.
// It checks if the content is already stored in the application's store.
// If found, it returns the stored content.
// If not found, it queries the application's DAO to find the content by name.
// If an error occurs during the retrieval process, it logs the error and returns an empty string.
// Finally, it stores the retrieved content in the application's store for future use.
func getWelcomeContent(app *pocketbase.PocketBase, repo *repositories.LandingRepository) (string, error) {

	if cached, ok := utils.GetCachedValue[string](app, "strings:welcome"); ok {
		return cached, nil
	}

	content, err := repo.GetWelcomeContent()
	if err != nil {
		app.Logger().Error("Error getting welcome content", "error", err.Error())
		return "", err
	}

	utils.SetCachedValue(app, "strings:welcome", content, landingCacheTTL)

	return content, nil

}

// getArtistCount retrieves the count of artists from the database.
// It first checks if the count is already stored in the app's store.
// If found, it returns the stored count. Otherwise, it queries the database
// to get the count and stores it in the app's store for future use.
// It returns the count as a string and any error encountered during the process.
func getArtistCount(app *pocketbase.PocketBase, repo *repositories.LandingRepository) (string, error) {

	key := "count:artists"

	if cached, ok := utils.GetCachedValue[string](app, key); ok {
		return cached, nil
	}

	count, err := repo.CountPublishedArtists()
	if err != nil {
		app.Logger().Error("Error getting artist count", "error", err.Error())
		return "0", err
	}

	result := fmt.Sprintf("%d", count)

	utils.SetCachedValue(app, key, result, landingCacheTTL)

	return result, nil

}

// getArtworkCount retrieves the count of artworks from the database.
// It first checks if the count is already stored in the app's store, and if so, returns it.
// Otherwise, it queries the database to get the count of artworks where published is true.
// The count is then stored in the app's store for future use.
// If an error occurs during the retrieval or storage process, it returns an error along with the count "0".
func getArtworkCount(app *pocketbase.PocketBase, repo *repositories.LandingRepository) (string, error) {

	key := "count:artworks"

	if cached, ok := utils.GetCachedValue[string](app, key); ok {
		return cached, nil
	}

	count, err := repo.CountPublishedArtworks()
	if err != nil {
		app.Logger().Error("Error getting artwork count", "error", err.Error())
		return "0", err
	}

	result := fmt.Sprintf("%d", count)

	utils.SetCachedValue(app, key, result, landingCacheTTL)

	return result, nil

}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// This is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		se.Router.GET("/", func(c *core.RequestEvent) error {
			repo := repositories.NewLandingRepository(app)

			welcomeText, err := getWelcomeContent(app, repo)

			if err != nil {
				app.Logger().Error("Error getting welcome content", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			artistCount, err := getArtistCount(app, repo)

			if err != nil {
				app.Logger().Error("Error getting artist count for home page", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			artworkCount, err := getArtworkCount(app, repo)

			if err != nil {
				app.Logger().Error("Error getting artwork count for home page", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			content := pages.HomePage{
				Content:      welcomeText,
				ArtistCount:  artistCount,
				ArtworkCount: artworkCount,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Welcome to the Gallery")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Welcome to the Gallery")

			//TODO: Fix this
			// ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, c.Scheme()+"://"+c.Request().Host+c.Request().URL.String())

			c.Response.Header().Set("HX-Push-Url", "/")

			var buff bytes.Buffer

			err = pages.HomePageWrapped(content).Render(ctx, &buff)

			if err != nil {
				app.Logger().Error("Error rendering home page", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			return c.HTML(http.StatusOK, buff.String())
		})

		return se.Next()
	})
}
