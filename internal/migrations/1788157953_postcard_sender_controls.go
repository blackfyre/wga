package migrations

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addPostcardSenderControls, removePostcardSenderControls)
}

func addPostcardSenderControls(app core.App) error {
	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		return err
	}
	addFieldIfMissing(postcards, &core.TextField{Name: "submission_key_hash", Max: 64})
	if status, ok := postcards.Fields.GetByName("status").(*core.SelectField); ok && !slices.Contains(status.Values, "failed") {
		status.Values = append(status.Values, "failed")
	}
	addFieldIfMissing(postcards, &core.DateField{Name: "failed_at"})
	postcards.Indexes = appendIndexIfMissing(postcards.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_submission_key` ON `Postcards` (submission_key_hash) WHERE submission_key_hash != ''",
	)
	if err := app.Save(postcards); err != nil {
		return err
	}

	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		return err
	}
	if status, ok := deliveries.Fields.GetByName("status").(*core.SelectField); ok && !slices.Contains(status.Values, "failed") {
		status.Values = append(status.Values, "failed")
	}
	addFieldIfMissing(deliveries, &core.DateField{Name: "failed_at"})
	if err := app.Save(deliveries); err != nil {
		return err
	}

	if _, err := app.FindCollectionByNameOrId("tracking_postcard_sender_controls"); err == nil {
		return nil
	}
	controls := currentCollection("postcard_sender_controls", "tracking_postcard_sender_controls",
		relationField("postcard", "postcards", 1, 1, true, true),
		&core.TextField{Name: "token_hash", Required: true, Max: 64},
		&core.TextField{Name: "token_envelope", Max: 256},
		&core.DateField{Name: "expires_at", Required: true},
		&core.DateField{Name: "revoked_at"},
		&core.DateField{Name: "purged_at"},
	)
	controls.Indexes = append(controls.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_sender_control_postcard` ON `postcard_sender_controls` (postcard)",
		"CREATE UNIQUE INDEX `pbx_postcard_sender_control_token` ON `postcard_sender_controls` (token_hash) WHERE token_hash != ''",
		"CREATE INDEX `pbx_postcard_sender_control_expiry` ON `postcard_sender_controls` (expires_at, id)",
	)
	return app.Save(controls)
}

func removePostcardSenderControls(app core.App) error {
	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		return err
	}
	postcards.Indexes = removeNamedIndexes(postcards.Indexes, "pbx_postcard_submission_key")
	return app.Save(postcards)
}
