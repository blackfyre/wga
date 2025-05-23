package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func getContributorsFromGithub(app *pocketbase.PocketBase) ([]pages.GithubContributor, error) {

	ghContribCacheKey := "gh_contributors"

	if app.Store().Has(ghContribCacheKey) {
		return app.Store().Get(ghContribCacheKey).([]pages.GithubContributor), nil
	}

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
			app.Logger().Error("Error closing response body", "error", err)
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
			app.Logger().Error("Error closing file", "error", err)
		}
	}(f)

	err = json.NewEncoder(f).Encode(contributors)

	if err != nil {
		return nil, err
	}

	app.Store().Set(ghContribCacheKey, contributors)

	return contributors, nil
}

func readStoredContributors(app *pocketbase.PocketBase) ([]pages.GithubContributor, error) {
	f, err := os.Open("contributors.json")

	if err != nil {
		return nil, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			app.Logger().Error("Error closing file", "error", err)
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
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/contributors", func(c *core.RequestEvent) error {

			cacheKey := "contributors"
			fullUrl := tmplUtils.AssetUrl("/contributors")

			contributors, err := getContributorsFromGithub(app)

			if err != nil {

				app.Logger().Error("Error getting contributors from Github", "cacheKey", cacheKey, "error", err)

				contributors, err = readStoredContributors(app)

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

			c.Response.Header().Set("HX-Push-Url", fullUrl)

			// Create a bytes buffer to write the response to
			var buf bytes.Buffer

			err = pages.ContributorsPage(content).Render(ctx, &buf)

			if err != nil {
				app.Logger().Error("Error rendering artwork page", "error", err.Error())
				return c.Error(http.StatusInternalServerError, "failed to render response template", err)
			}

			return c.HTML(http.StatusOK, buf.String())

		})

		return se.Next()
	})
}
