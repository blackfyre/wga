package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// init registers the durable contributor snapshot and refresh execution schema.
func init() {
	m.Register(func(app core.App) error {
		snapshots := core.NewBaseCollection("contributor_snapshots")
		snapshots.Id = "contributor_snapshots"
		snapshots.Name = "contributorSnapshots"
		snapshots.MarkAsNew()
		snapshots.Fields.Add(
			&core.TextField{Id: "contributor_snapshot_key", Name: "key", Required: true},
			&core.JSONField{Id: "contributor_snapshot_payload", Name: "payload", Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		snapshots.AddIndex("pbx_contributor_snapshot_key", true, "key", "")
		if err := app.Save(snapshots); err != nil {
			return err
		}

		executions := core.NewBaseCollection("contributor_refresh_executions")
		executions.Id = "contributor_refresh_executions"
		executions.Name = "contributorRefreshExecutions"
		executions.MarkAsNew()
		executions.Fields.Add(
			&core.TextField{Id: "contributor_refresh_run_id", Name: "run_id", Required: true},
			&core.NumberField{Id: "contributor_refresh_attempt", Name: "attempt", Required: true},
			&core.NumberField{Id: "contributor_refresh_max_attempts", Name: "max_attempts", Required: true},
			&core.SelectField{Id: "contributor_refresh_status", Name: "status", Values: []string{"processing", "succeeded", "failed"}, MaxSelect: 1, Required: true},
			&core.DateField{Id: "contributor_refresh_completed_at", Name: "completed_at"},
			&core.NumberField{Id: "contributor_refresh_snapshot_count", Name: "snapshot_count"},
			&core.TextField{Id: "contributor_refresh_error_class", Name: "error_class"},
			&core.BoolField{Id: "contributor_refresh_error_retryable", Name: "error_retryable"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		executions.AddIndex("pbx_contributor_refresh_attempt", true, "run_id, attempt", "")
		executions.AddIndex("pbx_contributor_refresh_active", true, "status", "status = 'processing'")
		if err := app.Save(executions); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		if err := deleteCollection(app, "contributor_refresh_executions"); err != nil {
			return err
		}

		return deleteCollection(app, "contributor_snapshots")
	})
}
