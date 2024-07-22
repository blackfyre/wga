package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func getContributorsFromGithub() ([]pages.GithubContributor, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/blackfyre/wga/contributors", nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "blackfyre/wga")

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
        err := Body.Close()
        if err != nil {

        }
    }(resp.Body)

	var contributors []pages.GithubContributor

	err = json.NewDecoder(resp.Body).Decode(&contributors)

	if err != nil {
		return nil, err
	}

	// write to file
	f, err := os.Create("contributors.json")

	if err != nil {
		return nil, err
	}

	defer func(f *os.File) {
        err := f.Close()
        if err != nil {

        }
    }(f)

	err = json.NewEncoder(f).Encode(contributors)

	if err != nil {
		return nil, err
	}

	return contributors, nil
}

func readStoredContributors() ([]pages.GithubContributor, error) {
	f, err := os.Open("contributors.json")

	if err != nil {
		return nil, err
	}

	defer func(f *os.File) {
        err := f.Close()
        if err != nil {

        }
    }(f)

	var contributors []pages.GithubContributor

	err = json.NewDecoder(f).Decode(&contributors)

	if err != nil {
		return nil, err
	}

	return contributors, nil
}

func registerContributors(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/contributors", func(c echo.Context) error {

			cacheKey := "contributors"
			fullUrl := c.Scheme() + "://" + c.Request().Host + c.Request().URL.String()

			contributors, err := getContributorsFromGithub()

			if err != nil {

				app.Logger().Error("Error getting contributors from Github", "cacheKey", cacheKey, "error", err)

				contributors, err = readStoredContributors()

				if err != nil {
					app.Logger().Error("Error reading stored contributors", "cacheKey", cacheKey, "error", err)
					return apis.NewApiError(500, err.Error(), err)
				}
			}

			content := pages.ContributorsPageDTO{
				Contributors: contributors,
			}

			ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Contributors")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "The people who have contributed to the Web Gallery of Art.")
			ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, fullUrl)

			c.Response().Header().Set("HX-Push-Url", fullUrl)
			err = pages.ContributorsPage(content).Render(ctx, c.Response().Writer)

			if err != nil {
				app.Logger().Error("Error rendering artwork page", "error", err.Error())
				return c.String(http.StatusInternalServerError, "failed to render response template")
			}

			return nil

		})

		return nil
	})
}
