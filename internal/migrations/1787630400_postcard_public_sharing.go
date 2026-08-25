package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addPostcardPublicSharing, removePostcardPublicSharing)
}

func addPostcardPublicSharing(app core.App) error {
	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		return err
	}
	addFieldIfMissing(postcards, &core.BoolField{Name: "include_music"})
	addFieldIfMissing(postcards, &core.DateField{Name: "retention_until"})
	addFieldIfMissing(postcards, &core.DateField{Name: "sender_email_purged_at"})
	addFieldIfMissing(postcards, &core.DateField{Name: "content_purged_at"})
	if err := app.Save(postcards); err != nil {
		return err
	}

	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		return err
	}
	addFieldIfMissing(deliveries, &core.TextField{Name: "view_token_envelope", Max: 256})
	addFieldIfMissing(deliveries, &core.TextField{Name: "view_token_hash", Max: 64})
	addFieldIfMissing(deliveries, &core.DateField{Name: "view_expires_at"})
	addFieldIfMissing(deliveries, &core.DateField{Name: "recipient_purged_at"})
	deliveries.Indexes = appendIndexIfMissing(deliveries.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_delivery_view_token` ON `postcard_deliveries` (view_token_hash) WHERE view_token_hash != ''",
		"CREATE INDEX `pbx_postcard_delivery_view_expiry` ON `postcard_deliveries` (view_expires_at, id)",
	)
	if err := app.Save(deliveries); err != nil {
		return err
	}

	attempts, err := app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if err != nil {
		return err
	}
	addFieldIfMissing(attempts, &core.SelectField{Name: "direction", Values: []string{"outbound", "inbound"}, MaxSelect: 1})
	addFieldIfMissing(attempts, &core.TextField{Name: "integration_key", Max: 100})
	addFieldIfMissing(attempts, &core.TextField{Name: "message_type", Max: 100})
	addFieldIfMissing(attempts, &core.TextField{Name: "external_message_id", Max: 255})
	addFieldIfMissing(attempts, &core.TextField{Name: "deduplication_key", Max: 255})
	addFieldIfMissing(attempts, &core.RelationField{Name: "causation_message_id", CollectionId: attempts.Id, MaxSelect: 1})
	addFieldIfMissing(attempts, &core.TextField{Name: "payload_reference", Max: 255})
	addFieldIfMissing(attempts, &core.TextField{Name: "payload_format", Max: 50})
	addFieldIfMissing(attempts, &core.DateField{Name: "payload_retention_until"})
	addFieldIfMissing(attempts, &core.DateField{Name: "payload_purged_at"})
	addFieldIfMissing(attempts, &core.JSONField{Name: "transport_metadata", MaxSize: 2048})
	addFieldIfMissing(attempts, &core.TextField{Name: "last_error_code", Max: 100})
	addFieldIfMissing(attempts, &core.TextField{Name: "result_summary", Max: 500})
	attempts.Indexes = appendIndexIfMissing(attempts.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_attempt_dedup` ON `postcard_delivery_attempts` (integration_key, deduplication_key) WHERE deduplication_key != ''",
		"CREATE INDEX `pbx_postcard_attempt_correlation` ON `postcard_delivery_attempts` (correlation_id)",
		"CREATE INDEX `pbx_postcard_attempt_causation` ON `postcard_delivery_attempts` (causation_message_id)",
		"CREATE INDEX `pbx_postcard_attempt_unresolved` ON `postcard_delivery_attempts` (status, resolved_at, id)",
	)
	return app.Save(attempts)
}

func removePostcardPublicSharing(app core.App) error {
	attempts, err := app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if err != nil {
		return err
	}
	attempts.Indexes = removeNamedIndexes(attempts.Indexes, "pbx_postcard_attempt_dedup", "pbx_postcard_attempt_correlation", "pbx_postcard_attempt_causation", "pbx_postcard_attempt_unresolved")
	if err := app.Save(attempts); err != nil {
		return err
	}

	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		return err
	}
	deliveries.Indexes = removeNamedIndexes(deliveries.Indexes, "pbx_postcard_delivery_view_token", "pbx_postcard_delivery_view_expiry")
	if err := app.Save(deliveries); err != nil {
		return err
	}

	// Postcard content, recipient-access hashes, and integration-message
	// lifecycle fields are authoritative data. A code rollback disables the
	// public routes and workers but deliberately preserves that data for a
	// forward-compatible redeploy and operator review.
	return nil
}

func addFieldIfMissing(collection *core.Collection, field core.Field) {
	if collection.Fields.GetByName(field.GetName()) == nil {
		collection.Fields.Add(field)
	}
}

func appendIndexIfMissing(indexes []string, additions ...string) []string {
	for _, addition := range additions {
		found := false
		for _, existing := range indexes {
			if existing == addition {
				found = true
				break
			}
		}
		if !found {
			indexes = append(indexes, addition)
		}
	}
	return indexes
}

func removeNamedIndexes(indexes []string, names ...string) []string {
	kept := indexes[:0]
	for _, index := range indexes {
		remove := false
		for _, name := range names {
			if strings.Contains(index, name) {
				remove = true
				break
			}
		}
		if !remove {
			kept = append(kept, index)
		}
	}
	return kept
}
