package contributors

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type providerFunc func(context.Context) ([]Contributor, error)

func (f providerFunc) Fetch(ctx context.Context) ([]Contributor, error) {
	return f(ctx)
}

func TestGitHubProvider(t *testing.T) {
	t.Run("returns contributors from a successful response", func(t *testing.T) {
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "2" {
				_, _ = w.Write([]byte(`[{"login":"hubot","avatar_url":"avatar","html_url":"profile","contributions":2}]`))
				return
			}
			w.Header().Set("Link", "<"+server.URL+"?page=2>; rel=\"next\"")
			_, _ = w.Write([]byte(`[{"login":"octocat","avatar_url":"avatar","html_url":"profile","contributions":3}]`))
		}))
		defer server.Close()

		contributors, err := newGitHubProvider(server.Client(), server.URL).Fetch(context.Background())
		if err != nil {
			t.Fatalf("fetch contributors: %v", err)
		}
		if len(contributors) != 2 || contributors[0].Login != "octocat" || contributors[1].Login != "hubot" {
			t.Fatalf("contributors = %+v", contributors)
		}
	})

	for _, test := range []struct {
		name string
		http.Handler
		clientTimeout time.Duration
		wantKind      ProviderErrorKind
	}{
		{
			name: "maps malformed responses to contract errors",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`not json`))
			}),
			wantKind: ProviderErrorContract,
		},
		{
			name: "maps provider failures to retryable errors",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}),
			wantKind: ProviderErrorUnavailable,
		},
		{
			name: "maps timeouts without provider details",
			Handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				time.Sleep(100 * time.Millisecond)
			}),
			clientTimeout: 10 * time.Millisecond,
			wantKind:      ProviderErrorTimeout,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.Handler)
			defer server.Close()
			client := server.Client()
			client.Timeout = test.clientTimeout

			_, err := newGitHubProvider(client, server.URL).Fetch(context.Background())
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) {
				t.Fatalf("error = %v, want provider error", err)
			}
			if providerErr.Kind != test.wantKind {
				t.Fatalf("error kind = %q, want %q", providerErr.Kind, test.wantKind)
			}
		})
	}
}

func TestStoreUsesFallbackAndKeepsPreviousSnapshotOnFailure(t *testing.T) {
	app := newContributorTestApp(t)
	store, err := NewStore(app)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	fallback, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("read fallback: %v", err)
	}
	if fallback.Source != SnapshotSourceFileFallback || len(fallback.Contributors) == 0 {
		t.Fatalf("fallback = %+v", fallback)
	}

	previous := []Contributor{{Login: "stored", Contributions: 10}}
	if err := store.Replace(app, previous); err != nil {
		t.Fatalf("store snapshot: %v", err)
	}

	job := newRefreshJob(app, providerFunc(func(context.Context) ([]Contributor, error) {
		return nil, &ProviderError{Kind: ProviderErrorContract}
	}), store, func(context.Context, time.Duration) error { return nil })
	if err := job.Run(context.Background(), "run-failure"); err == nil {
		t.Fatal("expected refresh failure")
	}

	snapshot, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("read stored snapshot: %v", err)
	}
	if snapshot.Source != SnapshotSourceCache || len(snapshot.Contributors) != 1 || snapshot.Contributors[0].Login != "stored" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestRefreshRecordsAttemptsBeforeFetchingAndRetries(t *testing.T) {
	app := newContributorTestApp(t)
	store, err := NewStore(app)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	calls := 0
	job := newRefreshJob(app, providerFunc(func(context.Context) ([]Contributor, error) {
		calls++
		executions, err := app.FindRecordsByFilter(constants.CollectionContributorRefreshExecutions, "status = 'processing'", "", 0, 0)
		if err != nil {
			t.Fatalf("find active execution: %v", err)
		}
		if len(executions) != 1 {
			t.Fatalf("active executions = %d, want 1", len(executions))
		}
		if calls == 1 {
			return nil, &ProviderError{Kind: ProviderErrorUnavailable, Retryable: true}
		}
		return []Contributor{{Login: "refreshed", Contributions: 1}}, nil
	}), store, func(context.Context, time.Duration) error { return nil })

	if err := job.Run(context.Background(), "run-success"); err != nil {
		t.Fatalf("run refresh: %v", err)
	}

	executions, err := app.FindRecordsByFilter(constants.CollectionContributorRefreshExecutions, "run_id = 'run-success'", "attempt", 0, 0)
	if err != nil {
		t.Fatalf("find executions: %v", err)
	}
	if len(executions) != 2 || executions[0].GetString("status") != "failed" || executions[1].GetString("status") != "succeeded" {
		t.Fatalf("executions = %+v", executions)
	}
}

func TestRefreshRecordsPersistenceFailureAndAllowsRetry(t *testing.T) {
	app := newContributorTestApp(t)
	store, err := NewStore(app)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	snapshots, err := app.FindCollectionByNameOrId(constants.CollectionContributorSnapshots)
	if err != nil {
		t.Fatalf("find snapshot collection: %v", err)
	}
	if err := app.Delete(snapshots); err != nil {
		t.Fatalf("delete snapshot collection: %v", err)
	}

	job := newRefreshJob(app, providerFunc(func(context.Context) ([]Contributor, error) {
		return []Contributor{{Login: "refreshed", Contributions: 1}}, nil
	}), store, func(context.Context, time.Duration) error { return nil })
	if err := job.Run(context.Background(), "run-persistence-one"); err == nil {
		t.Fatal("expected persistence failure")
	}
	if err := job.Run(context.Background(), "run-persistence-two"); err == nil {
		t.Fatal("expected second persistence failure")
	}

	executions, err := app.FindRecordsByFilter(constants.CollectionContributorRefreshExecutions, "status = 'failed'", "", 0, 0)
	if err != nil {
		t.Fatalf("find failed executions: %v", err)
	}
	if len(executions) != 2 {
		t.Fatalf("failed executions = %d, want 2", len(executions))
	}
	for _, execution := range executions {
		if execution.GetString("error_class") != "persistence" {
			t.Fatalf("error class = %q, want persistence", execution.GetString("error_class"))
		}
	}
}

func TestRefreshRecoversAbandonedExecution(t *testing.T) {
	app := newContributorTestApp(t)
	store, err := NewStore(app)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	collection, err := app.FindCollectionByNameOrId(constants.CollectionContributorRefreshExecutions)
	if err != nil {
		t.Fatalf("find refresh collection: %v", err)
	}
	abandoned := core.NewRecord(collection)
	abandoned.Set("run_id", "abandoned")
	abandoned.Set("attempt", 1)
	abandoned.Set("max_attempts", refreshMaxAttempts)
	abandoned.Set("status", "processing")
	abandoned.Set("claim_expires_at", time.Now().Add(-time.Minute))
	if err := app.Save(abandoned); err != nil {
		t.Fatalf("save abandoned execution: %v", err)
	}

	job := newRefreshJob(app, providerFunc(func(context.Context) ([]Contributor, error) {
		return []Contributor{{Login: "refreshed", Contributions: 1}}, nil
	}), store, func(context.Context, time.Duration) error { return nil })
	if err := job.Run(context.Background(), "run-after-abandoned"); err != nil {
		t.Fatalf("run refresh: %v", err)
	}

	abandoned, err = app.FindRecordById(constants.CollectionContributorRefreshExecutions, abandoned.Id)
	if err != nil {
		t.Fatalf("find abandoned execution: %v", err)
	}
	if abandoned.GetString("status") != "failed" || abandoned.GetString("error_class") != "abandoned" {
		t.Fatalf("abandoned execution = %+v", abandoned)
	}
}

func newContributorTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	snapshots := core.NewBaseCollection(constants.CollectionContributorSnapshots)
	snapshots.Fields.Add(
		&core.TextField{Name: "key", Required: true},
		&core.JSONField{Name: "payload", Required: true},
	)
	snapshots.AddIndex("snapshot_key", true, "key", "")
	if err := app.Save(snapshots); err != nil {
		t.Fatalf("create snapshot collection: %v", err)
	}

	executions := core.NewBaseCollection(constants.CollectionContributorRefreshExecutions)
	executions.Fields.Add(
		&core.TextField{Name: "run_id", Required: true},
		&core.NumberField{Name: "attempt", Required: true},
		&core.NumberField{Name: "max_attempts", Required: true},
		&core.SelectField{Name: "status", Values: []string{"processing", "succeeded", "failed"}, MaxSelect: 1, Required: true},
		&core.DateField{Name: "claim_expires_at", Required: true},
		&core.DateField{Name: "completed_at"},
		&core.NumberField{Name: "snapshot_count"},
		&core.TextField{Name: "error_class"},
		&core.BoolField{Name: "error_retryable"},
	)
	executions.AddIndex("refresh_attempt", true, "run_id, attempt", "")
	executions.AddIndex("refresh_active", true, "status", "status = 'processing'")
	if err := app.Save(executions); err != nil {
		t.Fatalf("create refresh collection: %v", err)
	}

	return app
}
