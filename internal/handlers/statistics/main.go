package statistics

import (
	"bytes"
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
	"golang.org/x/sync/singleflight"
)

const statisticsCacheTTL = 60 * time.Minute

type marshaledStatistics[T any] struct {
	data T
	json string
}

var statisticsFetchGroup singleflight.Group

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

func marshalStats[T any](app *pocketbase.PocketBase, key string, fetch func() (T, error)) (T, string, error) {
	var zero T

	if cached, ok := utils.GetCachedValue[marshaledStatistics[T]](app, key); ok {
		return cached.data, cached.json, nil
	}

	value, err, _ := statisticsFetchGroup.Do(key, func() (interface{}, error) {
		if cached, ok := utils.GetCachedValue[marshaledStatistics[T]](app, key); ok {
			return cached, nil
		}

		data, err := fetch()
		if err != nil {
			return nil, err
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		cached := marshaledStatistics[T]{
			data: data,
			json: string(jsonBytes),
		}
		utils.SetCachedValue(app, key, cached, statisticsCacheTTL)

		return cached, nil
	})
	if err != nil {
		return zero, "[]", err
	}

	cached := value.(marshaledStatistics[T])
	return cached.data, cached.json, nil
}

func summarizeArtFormRows(rows []repositories.ArtFormDistribution) []pages.StatisticsArtFormRow {
	summary := make([]pages.StatisticsArtFormRow, len(rows))
	for i, row := range rows {
		summary[i] = pages.StatisticsArtFormRow{
			Name:  row.Name,
			Count: row.Count,
		}
	}

	return summary
}

func summarizeSchoolPeriodRows(rows []repositories.SchoolPeriodRow) []pages.StatisticsSchoolPeriodRow {
	summary := make([]pages.StatisticsSchoolPeriodRow, len(rows))
	for i, row := range rows {
		summary[i] = pages.StatisticsSchoolPeriodRow{
			Period: fmt.Sprintf("%d–%d", row.PeriodStart, row.PeriodStart+49),
			School: row.School,
			Count:  row.Count,
		}
	}

	return summary
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

			artFormRows, artFormData, err := marshalStats(app, "statistics:art_form_distribution", statsRepo.GetArtFormDistribution)
			if err != nil {
				app.Logger().Error("Error getting art form distribution", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			artworksByPeriodRows, artworksByPeriodData, err := marshalStats(app, "statistics:artworks_by_school_period", statsRepo.GetArtworksBySchoolAndPeriod)
			if err != nil {
				app.Logger().Error("Error getting artworks by period", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			artistsByPeriodRows, artistsByPeriodData, err := marshalStats(app, "statistics:artists_by_school_period", statsRepo.GetArtistsBySchoolAndPeriod)
			if err != nil {
				app.Logger().Error("Error getting artists by period", "error", err.Error())
				return utils.ServerFaultError(c)
			}

			content := pages.StatisticsPageDTO{
				ArtistCount:           artistCount,
				ArtworkCount:          artworkCount,
				ArtFormData:           artFormData,
				ArtworksPeriodData:    artworksByPeriodData,
				ArtistsPeriodData:     artistsByPeriodData,
				ArtFormSummary:        summarizeArtFormRows(artFormRows),
				ArtworksPeriodSummary: summarizeSchoolPeriodRows(artworksByPeriodRows),
				ArtistsPeriodSummary:  summarizeSchoolPeriodRows(artistsByPeriodRows),
			}

			ctx := tmplUtils.DecorateContext(c.Request.Context(), tmplUtils.TitleKey, "Statistics")
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
