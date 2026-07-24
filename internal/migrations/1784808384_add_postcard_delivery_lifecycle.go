package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		postcards, err := app.FindCollectionByNameOrId("postcards")
		if err != nil {
			return err
		}

		postcards.Fields.Add(
			&core.TextField{Id: "postcard_correlation_id", Name: "correlation_id"},
			&core.DateField{Id: "postcard_received_at", Name: "received_at"},
		)
		if err := app.Save(postcards); err != nil {
			return err
		}

		deliveries := core.NewBaseCollection("postcard_deliveries")
		deliveries.Id = "postcard_deliveries"
		deliveries.Name = "postcardDeliveries"
		deliveries.MarkAsNew()
		deliveries.Fields.Add(
			&core.RelationField{Id: "postcard_delivery_postcard", Name: "postcard", CollectionId: postcards.Id, Required: true, CascadeDelete: true},
			&core.TextField{Id: "postcard_delivery_recipient", Name: "recipient", Required: true},
			&core.SelectField{Id: "postcard_delivery_status", Name: "status", Values: []string{"pending", "sent", "cancelled"}, MaxSelect: 1, Required: true},
			&core.DateField{Id: "postcard_delivery_sent_at", Name: "sent_at"},
			&core.DateField{Id: "postcard_delivery_cancelled_at", Name: "cancelled_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		deliveries.AddIndex("pbx_postcard_delivery_recipient", true, "postcard, recipient", "")
		deliveries.AddIndex("pbx_postcard_delivery_status", false, "postcard, status", "")
		if err := app.Save(deliveries); err != nil {
			return err
		}

		attempts := core.NewBaseCollection("postcard_delivery_attempts")
		attempts.Id = "postcard_delivery_attempts"
		attempts.Name = "postcardDeliveryAttempts"
		attempts.MarkAsNew()
		attempts.Fields.Add(
			&core.RelationField{Id: "postcard_attempt_delivery", Name: "delivery", CollectionId: deliveries.Id, Required: true, CascadeDelete: true},
			&core.NumberField{Id: "postcard_attempt_sequence", Name: "sequence", Required: true},
			&core.SelectField{Id: "postcard_attempt_status", Name: "status", Values: []string{"queued", "processing", "processed", "dead_lettered", "cancelled"}, MaxSelect: 1, Required: true},
			&core.TextField{Id: "postcard_attempt_correlation_id", Name: "correlation_id", Required: true},
			&core.TextField{Id: "postcard_attempt_message_id", Name: "message_id", Required: true},
			&core.NumberField{Id: "postcard_attempt_count", Name: "attempt_count"},
			&core.NumberField{Id: "postcard_attempt_max_attempts", Name: "max_attempts", Required: true},
			&core.DateField{Id: "postcard_attempt_available_at", Name: "available_at", Required: true},
			&core.TextField{Id: "postcard_attempt_claim_token", Name: "claim_token"},
			&core.DateField{Id: "postcard_attempt_claim_expires_at", Name: "claim_expires_at"},
			&core.DateField{Id: "postcard_attempt_transport_started_at", Name: "transport_started_at"},
			&core.DateField{Id: "postcard_attempt_last_attempt_at", Name: "last_attempt_at"},
			&core.DateField{Id: "postcard_attempt_processed_at", Name: "processed_at"},
			&core.DateField{Id: "postcard_attempt_dead_lettered_at", Name: "dead_lettered_at"},
			&core.TextField{Id: "postcard_attempt_result_code", Name: "result_code"},
			&core.TextField{Id: "postcard_attempt_error_class", Name: "last_error_class"},
			&core.BoolField{Id: "postcard_attempt_error_retryable", Name: "last_error_retryable"},
			&core.TextField{Id: "postcard_attempt_error_summary", Name: "last_error_summary", Max: 255},
			&core.SelectField{Id: "postcard_attempt_resolution_code", Name: "resolution_code", Values: []string{"replayed_unmodified", "resolved_manually", "closed_without_replay", "ignored_duplicate"}, MaxSelect: 1},
			&core.TextField{Id: "postcard_attempt_resolution_summary", Name: "resolution_summary", Max: 255},
			&core.DateField{Id: "postcard_attempt_resolved_at", Name: "resolved_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		attempts.AddIndex("pbx_postcard_attempt_sequence", true, "delivery, sequence", "")
		attempts.AddIndex("pbx_postcard_attempt_message_id", true, "message_id", "")
		attempts.AddIndex("pbx_postcard_attempt_due", false, "status, available_at", "")
		attempts.AddIndex("pbx_postcard_attempt_expired", false, "status, claim_expires_at", "")
		if err := app.Save(attempts); err != nil {
			return err
		}
		attempts.Fields.Add(
			&core.RelationField{Id: "postcard_attempt_replay_of", Name: "replay_of", CollectionId: attempts.Id},
		)
		if err := app.Save(attempts); err != nil {
			return err
		}

		return nil
	}, nil)
}
