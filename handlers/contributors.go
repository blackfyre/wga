package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type GithubContributor struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
	Contributions     int    `json:"contributions"`
}

func getContributorsFromGithub() ([]GithubContributor, error) {
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

	defer resp.Body.Close()

	var contributors []GithubContributor

	err = json.NewDecoder(resp.Body).Decode(&contributors)

	if err != nil {
		return nil, err
	}

	// write to file
	f, err := os.Create("contributors.json")

	if err != nil {
		return nil, err
	}

	defer f.Close()

	err = json.NewEncoder(f).Encode(contributors)

	if err != nil {
		return nil, err
	}

	return contributors, nil
}

func readStoredContributors() ([]GithubContributor, error) {
	f, err := os.Open("contributors.json")

	if err != nil {
		return nil, err
	}

	defer f.Close()

	var contributors []GithubContributor

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
			htmx := utils.IsHtmxRequest(c)

			if htmx {
				cacheKey = cacheKey + "-htmx"
			}

			html := ""

			if app.Store().Has(cacheKey) {
				storedValue, ok := app.Store().Get(cacheKey).(string)
				if !ok {
				    // Handle the case where the value is not a string
				    app.Logger().Error("Expected string value in store for key:", cacheKey)
				    return apis.NewApiError(500, "Internal server error", nil)
				}
				html = storedValue
			} else {
				contributors, err := getContributorsFromGithub()

				if err != nil {
					app.Logger().Error("Error getting contributors from Github, HTTP status code:", resp.StatusCode, err)
					contributors, err = readStoredContributors()

					if err != nil {
						app.Logger().Error("Error reading stored contributors", err)
						return apis.NewApiError(500, err.Error(), err)
					}
				}

				data := assets.NewRenderData(app)
				data["Contributors"] = contributors

				html, err = assets.Render(assets.Renderable{
					IsHtmx: htmx,
					Block:  "contributors:content",
					Data:   data,
				})

				if err != nil {
					app.Logger().Error("Error rendering contributors", err)
					return apis.NewApiError(500, err.Error(), err)
				}

				app.Store().Set(cacheKey, html)
			}

			c.Response().Header().Set("HX-Push-Url", "/contributors")

			return c.HTML(http.StatusOK, html)

		})

		return nil
	})
}
