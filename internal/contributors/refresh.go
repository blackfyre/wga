package contributors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

const refreshMaxAttempts = 3

var ErrRefreshInProgress = errors.New("contributor refresh already in progress")

type refreshJob struct {
	app      core.App
	provider Provider
	store    *Store
	wait     func(context.Context, time.Duration) error
}

func NewRefreshJob(app core.App, provider Provider, store *Store) RefreshJob {
	return newRefreshJob(app, provider, store, waitForRetry)
}

func newRefreshJob(app core.App, provider Provider, store *Store, wait func(context.Context, time.Duration) error) RefreshJob {
	return &refreshJob{app: app, provider: provider, store: store, wait: wait}
}

func (j *refreshJob) Run(ctx context.Context, runID string) error {
	for attempt := 1; attempt <= refreshMaxAttempts; attempt++ {
		execution, err := j.startExecution(runID, attempt)
		if err != nil {
			return err
		}

		contributors, fetchErr := j.provider.Fetch(ctx)
		if fetchErr == nil {
			err = j.app.RunInTransaction(func(txApp core.App) error {
				if err := j.store.Replace(txApp, contributors); err != nil {
					return err
				}

				execution.Set("status", "succeeded")
				execution.Set("completed_at", time.Now().UTC())
				execution.Set("snapshot_count", len(contributors))
				return txApp.Save(execution)
			})
			if err != nil {
				execution.Set("status", "failed")
				execution.Set("completed_at", time.Now().UTC())
				execution.Set("error_class", "persistence")
				execution.Set("error_retryable", false)
				if recordErr := j.app.Save(execution); recordErr != nil {
					return fmt.Errorf("store contributor refresh: %w", errors.Join(err, recordErr))
				}
				return fmt.Errorf("store contributor refresh: %w", err)
			}

			return nil
		}

		retryable, errorClass := classifyProviderError(fetchErr)
		execution.Set("status", "failed")
		execution.Set("completed_at", time.Now().UTC())
		execution.Set("error_class", errorClass)
		execution.Set("error_retryable", retryable)
		if err := j.app.Save(execution); err != nil {
			return fmt.Errorf("record contributor refresh failure: %w", err)
		}

		if !retryable || attempt == refreshMaxAttempts {
			return fetchErr
		}
		if err := j.wait(ctx, time.Duration(attempt)*time.Second); err != nil {
			return err
		}
	}

	return nil
}

func (j *refreshJob) startExecution(runID string, attempt int) (*core.Record, error) {
	active, err := j.app.FindRecordsByFilter(constants.CollectionContributorRefreshExecutions, "status = 'processing'", "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(active) != 0 {
		return nil, ErrRefreshInProgress
	}

	collection, err := j.app.FindCollectionByNameOrId(constants.CollectionContributorRefreshExecutions)
	if err != nil {
		return nil, err
	}

	execution := core.NewRecord(collection)
	execution.Set("run_id", runID)
	execution.Set("attempt", attempt)
	execution.Set("max_attempts", refreshMaxAttempts)
	execution.Set("status", "processing")
	if err := j.app.Save(execution); err != nil {
		return nil, err
	}

	return execution, nil
}

func classifyProviderError(err error) (bool, string) {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Retryable, string(providerErr.Kind)
	}

	return false, "internal"
}

func waitForRetry(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
