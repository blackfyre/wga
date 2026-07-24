package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
)

func TestContributorsRepositoryGetContributors(t *testing.T) {
	t.Run("returns cached contributors without calling api", func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		cacheKey := "contributors:test:cache"
		cached := []pages.GithubContributor{{Login: "cached-user", Contributions: 10}}
		utils.SetCachedValue(app, cacheKey, cached, time.Hour)

		repo := newContributorsRepositoryWithConfig(
			app,
			&http.Client{Timeout: 100 * time.Millisecond},
			"http://127.0.0.1:1",
			filepath.Join(t.TempDir(), "contributors.json"),
			cacheKey,
			time.Hour,
		)

		contributors, source, err := repo.GetContributorsWithSource(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if source != ContributorsSourceCache {
			t.Fatalf("expected source %q, got %q", ContributorsSourceCache, source)
		}

		if len(contributors) != 1 || contributors[0].Login != "cached-user" {
			t.Fatalf("expected cached contributor result, got %+v", contributors)
		}
	})

	t.Run("fetches from api and persists to file", func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		apiContributors := []pages.GithubContributor{{Login: "api-user", Contributions: 42}}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(apiContributors)
		}))
		defer ts.Close()

		cacheFile := filepath.Join(t.TempDir(), "contributors.json")
		repo := newContributorsRepositoryWithConfig(
			app,
			ts.Client(),
			ts.URL,
			cacheFile,
			"contributors:test:api",
			time.Hour,
		)

		contributors, source, err := repo.GetContributorsWithSource(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if source != ContributorsSourceAPI {
			t.Fatalf("expected source %q, got %q", ContributorsSourceAPI, source)
		}

		if len(contributors) != 1 || contributors[0].Login != "api-user" {
			t.Fatalf("expected api contributor result, got %+v", contributors)
		}

		if _, err := os.Stat(cacheFile); err != nil {
			t.Fatalf("expected contributors cache file to be created, got %v", err)
		}
	})

	t.Run("falls back to stored file when api fails", func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		fallbackContributors := []pages.GithubContributor{{Login: "file-user", Contributions: 5}}

		tmpDir := t.TempDir()
		cacheFile := filepath.Join(tmpDir, "contributors.json")
		file, err := os.Create(cacheFile)
		if err != nil {
			t.Fatalf("failed to create fallback file: %v", err)
		}
		if err := json.NewEncoder(file).Encode(fallbackContributors); err != nil {
			_ = file.Close()
			t.Fatalf("failed to write fallback file: %v", err)
		}
		_ = file.Close()

		repo := newContributorsRepositoryWithConfig(
			app,
			&http.Client{Timeout: 100 * time.Millisecond},
			"http://127.0.0.1:1",
			cacheFile,
			"contributors:test:fallback",
			time.Hour,
		)

		contributors, source, err := repo.GetContributorsWithSource(context.Background())
		if err != nil {
			t.Fatalf("expected fallback to succeed, got %v", err)
		}

		if source != ContributorsSourceFileFallback {
			t.Fatalf("expected source %q, got %q", ContributorsSourceFileFallback, source)
		}

		if len(contributors) != 1 || contributors[0].Login != "file-user" {
			t.Fatalf("expected fallback contributor result, got %+v", contributors)
		}
	})
}

func TestFetchContributorsFromAPIHonoursCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer ts.Close()

	repo := newContributorsRepositoryWithConfig(
		pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"}),
		ts.Client(),
		ts.URL,
		filepath.Join(t.TempDir(), "contributors.json"),
		"contributors:test:cancellation",
		time.Hour,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := repo.fetchContributorsFromAPI(ctx)
		result <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("contributors request did not reach the upstream server")
	}
	cancel()

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("upstream request context was not cancelled")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("fetch error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("contributors request did not return after cancellation")
	}
}
