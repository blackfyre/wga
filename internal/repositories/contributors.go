package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
)

const (
	defaultContributorsAPIURL   = "https://api.github.com/repos/blackfyre/wga/contributors"
	defaultContributorsCacheKey = "gh_contributors"
	defaultContributorsCacheTTL = 6 * time.Hour
	defaultContributorsFilePath = "contributors.json"
)

type ContributorsRepository struct {
	app       *pocketbase.PocketBase
	client    *http.Client
	apiURL    string
	cacheKey  string
	cacheTTL  time.Duration
	cacheFile string
}

type ContributorsSource string

const (
	ContributorsSourceCache        ContributorsSource = "cache"
	ContributorsSourceAPI          ContributorsSource = "api"
	ContributorsSourceFileFallback ContributorsSource = "file_fallback"
)

func NewContributorsRepository(app *pocketbase.PocketBase) *ContributorsRepository {
	return &ContributorsRepository{
		app:       app,
		client:    &http.Client{Timeout: 10 * time.Second},
		apiURL:    defaultContributorsAPIURL,
		cacheKey:  defaultContributorsCacheKey,
		cacheTTL:  defaultContributorsCacheTTL,
		cacheFile: defaultContributorsFilePath,
	}
}

func newContributorsRepositoryWithConfig(app *pocketbase.PocketBase, client *http.Client, apiURL string, cacheFile string, cacheKey string, cacheTTL time.Duration) *ContributorsRepository {
	return &ContributorsRepository{
		app:       app,
		client:    client,
		apiURL:    apiURL,
		cacheKey:  cacheKey,
		cacheTTL:  cacheTTL,
		cacheFile: cacheFile,
	}
}

func (r *ContributorsRepository) GetContributors() ([]pages.GithubContributor, error) {
	contributors, _, err := r.GetContributorsWithSource(context.Background())
	return contributors, err
}

func (r *ContributorsRepository) GetContributorsWithSource(ctx context.Context) ([]pages.GithubContributor, ContributorsSource, error) {
	if cached, ok := utils.GetCachedValue[[]pages.GithubContributor](r.app, r.cacheKey); ok {
		return cached, ContributorsSourceCache, nil
	}

	contributors, err := r.fetchContributorsFromAPI(ctx)
	if err == nil {
		if err := r.persistContributors(contributors); err != nil {
			return nil, "", err
		}

		utils.SetCachedValue(r.app, r.cacheKey, contributors, r.cacheTTL)
		return contributors, ContributorsSourceAPI, nil
	}
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}

	logging.ContextLogger(r.app, ctx).Warn("Contributors API fetch failed; trying local fallback",
		"event", "contributors.fetch.fallback",
		"outcome", "fallback",
		"error_type", logging.ErrorType(err),
		"error", logging.Redact(err),
	)

	fallbackContributors, fallbackErr := r.readStoredContributors()
	if fallbackErr != nil {
		return nil, "", fmt.Errorf("failed to fetch contributors from api and fallback file: api=%v fallback=%v", err, fallbackErr)
	}

	utils.SetCachedValue(r.app, r.cacheKey, fallbackContributors, r.cacheTTL)

	return fallbackContributors, ContributorsSourceFileFallback, nil
}

func (r *ContributorsRepository) fetchContributorsFromAPI(ctx context.Context) ([]pages.GithubContributor, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "blackfyre/wga")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("contributors api returned status %d", resp.StatusCode)
	}

	var contributors []pages.GithubContributor
	if err := json.NewDecoder(resp.Body).Decode(&contributors); err != nil {
		return nil, err
	}

	return contributors, nil
}

func (r *ContributorsRepository) readStoredContributors() ([]pages.GithubContributor, error) {
	f, err := os.Open(r.cacheFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var contributors []pages.GithubContributor
	if err := json.NewDecoder(f).Decode(&contributors); err != nil {
		return nil, err
	}

	return contributors, nil
}

func (r *ContributorsRepository) persistContributors(contributors []pages.GithubContributor) error {
	f, err := os.Create(r.cacheFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	if err := json.NewEncoder(f).Encode(contributors); err != nil {
		return err
	}

	return nil
}
