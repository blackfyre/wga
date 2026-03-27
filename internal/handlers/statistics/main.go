package statistics

import (
	"bytes"
	"context"
	"encoding/json"
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

const statisticsCacheTTL = 60 * time.Minute

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
	utils.SetCachedValue(app, key, result, 15*time.Minute)

	return result, nil
}

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
	utils.SetCachedValue(app, key, result, 15*time.Minute)

	return result, nil
}

func marshalStats(app *pocketbase.PocketBase, key string, fetch func() (interface{}, error)) (string, error) {
	if cached, ok := utils.GetCachedValue[string](app, key); ok {
		return cached, nil
	}

	data, err := fetch()
	if err != nil {
		return "[]", err
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "[]", err
	}

	result := string(jsonBytes)
	utils.SetCachedValue(app, key, result, statisticsCacheTTL)

	return result, nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/statistics", func(c *core.RequestEvent) error {
			landingRepo := repositories.NewLandingRepository(app)
			statsRepo := repositories.NewStatisticsRepository(app)

			artistCount, err := getArtistCount(app, landingRepo)
			if err != nil {
				return utils.ServerFaultError(c)
			}

			artworkCount, err := getArtworkCount(app, landingRepo)
			if err != nil {
				return utils.ServerFaultError(c)
			}

			artFormData, err := marshalStats(app, "statistics:art_form_distribution", func() (interface{}, error) {
				return statsRepo.GetArtFormDistribution()
			})
			if err != nil {
				app.Logger().Error("Error getting art form distribution", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			artworksByPeriod, err := marshalStats(app, "statistics:artworks_by_school_period", func() (interface{}, error) {
				return statsRepo.GetArtworksBySchoolAndPeriod()
			})
			if err != nil {
				app.Logger().Error("Error getting artworks by period", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			artistsByPeriod, err := marshalStats(app, "statistics:artists_by_school_period", func() (interface{}, error) {
				return statsRepo.GetArtistsBySchoolAndPeriod()
			})
			if err != nil {
				app.Logger().Error("Error getting artists by period", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			content := pages.StatisticsPageDTO{
				ArtistCount:               artistCount,
				ArtworkCount:              artworkCount,
				ArtFormDistribution:       artFormData,
				ArtworksBySchoolAndPeriod: artworksByPeriod,
				ArtistsBySchoolAndPeriod:  artistsByPeriod,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Statistics")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Statistics about the Web Gallery of Art collection")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, tmplUtils.AssetUrl("/statistics"))

			pushUrl := utils.GenerateCurrentRelativePageUrl(c)
			c.Response.Header().Set("HX-Push-Url", pushUrl)

			var buf bytes.Buffer

			err = pages.StatisticsPage(content).Render(ctx, &buf)
			if err != nil {
				app.Logger().Error("Error rendering statistics page", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			return c.HTML(http.StatusOK, buf.String())
		})

		return se.Next()
	})
}
