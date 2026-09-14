package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addPostcardSharedRecipientAccess, removePostcardSharedRecipientAccess)
}

func addPostcardSharedRecipientAccess(app core.App) error {
	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		return err
	}
	addFieldIfMissing(postcards, &core.TextField{Name: "view_token_hash", Max: 64})
	addFieldIfMissing(postcards, &core.DateField{Name: "view_expires_at"})
	postcards.Indexes = appendIndexIfMissing(postcards.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_shared_view_token` ON `Postcards` (view_token_hash) WHERE view_token_hash != ''",
		"CREATE INDEX `pbx_postcard_shared_view_expiry` ON `Postcards` (view_expires_at, id)",
	)
	if err := app.Save(postcards); err != nil {
		return err
	}

	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		return err
	}
	addFieldIfMissing(deliveries, &core.TextField{Name: "recipient_mask", Max: 254})
	return app.Save(deliveries)
}

func removePostcardSharedRecipientAccess(app core.App) error {
	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		return err
	}
	postcards.Indexes = removeNamedIndexes(postcards.Indexes, "pbx_postcard_shared_view_token", "pbx_postcard_shared_view_expiry")
	// Shared access fields contain authoritative live bearer-token lookup state.
	// A rollback disables the new workflow but preserves data for forward recovery.
	return app.Save(postcards)
}
