package landing

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const landingCacheTTL = 15 * time.Minute

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

func getSchoolCount(app *pocketbase.PocketBase, repo *repositories.LandingRepository) (string, error) {
	const key = "count:schools"
	if cached, ok := utils.GetCachedValue[string](app, key); ok {
		return cached, nil
	}

	count, err := repo.CountSchools()
	if err != nil {
		return "0", err
	}

	result := fmt.Sprintf("%d", count)
	utils.SetCachedValue(app, key, result, landingCacheTTL)
	return result, nil
}

func getFeaturedArtwork(app *pocketbase.PocketBase) (pages.HomeFeaturedArtwork, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollectionArtworks,
		"published = true",
		"-created",
		1,
		0,
		nil,
	)
	if err != nil || len(records) == 0 {
		return pages.HomeFeaturedArtwork{}, err
	}

	artwork := records[0]
	authorIDs := artwork.GetStringSlice("author")
	if len(authorIDs) == 0 {
		return pages.HomeFeaturedArtwork{}, nil
	}

	artist, err := app.FindRecordById(constants.CollectionArtists, authorIDs[0])
	if err != nil {
		return pages.HomeFeaturedArtwork{}, err
	}

	featured := pages.HomeFeaturedArtwork{
		Title:  artwork.GetString("title"),
		Artist: artist.GetString("name"),
		Year:   artwork.GetString("year"),
		URL: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
			ArtistId:     artist.Id,
			ArtistName:   artist.GetString("name"),
			ArtworkId:    artwork.Id,
			ArtworkTitle: artwork.GetString("title"),
		}),
	}
	if imageName := artwork.GetString("image"); imageName != "" {
		featured.Image = url.GenerateThumbUrl(constants.CollectionArtworks, artwork.Id, imageName, url.ThumbnailArtworkFeature, "")
	}

	return featured, nil
}

func getRecentArtworks(app *pocketbase.PocketBase) ([]pages.HomeRecentArtwork, error) {
	records, err := app.FindRecordsByFilter(constants.CollectionArtworks, "published = true", "-created", 4, 0)
	if err != nil {
		return nil, err
	}

	works := make([]pages.HomeRecentArtwork, 0, len(records))
	for _, artwork := range records {
		authorIDs := artwork.GetStringSlice("author")
		if len(authorIDs) == 0 {
			continue
		}

		artist, err := app.FindRecordById(constants.CollectionArtists, authorIDs[0])
		if err != nil {
			return nil, err
		}

		work := pages.HomeRecentArtwork{
			Title:  artwork.GetString("title"),
			Artist: artist.GetString("name"),
			Year:   artwork.GetString("year"),
			URL: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId:     artist.Id,
				ArtistName:   artist.GetString("name"),
				ArtworkId:    artwork.Id,
				ArtworkTitle: artwork.GetString("title"),
			}),
		}
		if imageName := artwork.GetString("image"); imageName != "" {
			work.Image = url.GenerateThumbUrl(constants.CollectionArtworks, artwork.Id, imageName, url.ThumbnailArtworkCard, "")
		}

		works = append(works, work)
	}

	return works, nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// This is safe to be used by multiple goroutines
		// (it acts as store for the parsed templates)

		se.Router.GET("/", func(c *core.RequestEvent) error {
			repo := repositories.NewLandingRepository(app)

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
			schoolCount, err := getSchoolCount(app, repo)

			if err != nil {
				app.Logger().Error("Error getting school count for home page", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			featuredArtwork, err := getFeaturedArtwork(app)
			if err != nil {
				app.Logger().Error("Error getting featured artwork for home page", "error", err)
				return utils.ServerFaultError(c)
			}
			recentArtworks, err := getRecentArtworks(app)

			if err != nil {
				app.Logger().Error("Error getting recent artworks for home page", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			content := pages.HomePage{
				ArtistCount:     artistCount,
				ArtworkCount:    artworkCount,
				SchoolCount:     schoolCount,
				FeaturedArtwork: featuredArtwork,
				RecentArtworks:  recentArtworks,
			}

			ctx := tmplUtils.DecorateContext(c.Request.Context(), tmplUtils.TitleKey, "Web Gallery of Art | Explore artists and artworks")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Explore artists, artworks, and side-by-side comparisons in the Web Gallery of Art.")

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
