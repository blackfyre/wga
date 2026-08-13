package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(createCurrentSchema, func(core.App) error {
		return fmt.Errorf("current schema cannot be rolled back safely")
	})
}

func createCurrentSchema(app core.App) error {
	if err := saveCurrentCollection(app, currentCollection("Schools", "schools",
		textField("name", true, true),
		textField("slug", true, false),
	)); err != nil {
		return err
	}
	schools, err := app.FindCollectionByNameOrId("schools")
	if err != nil {
		return err
	}
	schools.Indexes = append(schools.Indexes, "CREATE UNIQUE INDEX `idx_unique_school` ON `Schools` (name, slug)")
	if err := app.Save(schools); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Art_forms", "art_forms",
		textField("name", false, true),
		textField("slug", false, false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Art_types", "art_types",
		textField("name", false, true),
		textField("slug", false, false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Art_periods", "art_periods",
		textField("name", false, true),
		textField("slug", false, false),
		numberField("start", false),
		numberField("end", false),
		textField("description", false, false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Music_composer", "music_composer",
		textField("name", true, true),
		selectField("century", []string{"12", "13", "14", "15", "16", "17", "18", "19", "20", "21"}, true),
		textField("language", false, true),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Artists", "artists",
		textField("name", true, true),
		textField("slug", true, false),
		editorField("bio", false),
		numberField("year_of_birth", false),
		numberField("year_of_death", false),
		textField("place_of_birth", false, false),
		textField("place_of_death", false, false),
		boolField("exact_year_of_birth", false),
		boolField("exact_year_of_death", false),
		textField("profession", false, false),
		selectField("known_place_of_birth", []string{"yes", "no", "n/a"}, true),
		selectField("known_place_of_death", []string{"yes", "no", "n/a"}, true),
		relationField("school", "schools", 1, 10, false, false),
		boolField("published", true),
	)); err != nil {
		return err
	}
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}
	artists.Fields.Add(relationField("also_known_as", "artists", 0, 0, false, false))
	artists.Indexes = append(artists.Indexes,
		"CREATE UNIQUE INDEX `pbx_artist_slug` ON `Artists` (slug)",
		"CREATE INDEX `pbx_artist_published_name` ON `Artists` (published, name, id)",
	)
	if err := app.Save(artists); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Artworks", "artworks",
		textField("title", true, true),
		relationField("author", "artists", 1, 10, false, false),
		relationField("form", "art_forms", 1, 20, false, false),
		relationField("type", "art_types", 1, 20, false, false),
		textField("technique", false, false),
		relationField("school", "schools", 1, 10, false, false),
		editorField("comment", false),
		boolField("published", false),
		fileField("image", []string{"image/jpeg", "image/png"}, 5*1024*1024, []string{"100x100", "320x240"}, false),
	)); err != nil {
		return err
	}
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}
	artworks.Indexes = append(artworks.Indexes, "CREATE INDEX `pbx_artwork_published_title` ON `Artworks` (published, title, id)")
	if err := app.Save(artworks); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Glossary", "glossary",
		textField("expression", true, true),
		textFieldWithMax("definition", true, false, 10000),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Guestbook", "guestbook",
		textField("message", false, false),
		textField("name", false, false),
		emailField("email", false),
		textField("location", false, false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Music_song", "music_song",
		textField("title", true, true),
		relationField("composer", "music_composer", 1, 20, false, false),
		fileField("source", []string{"audio/mpeg", "audio/mp3"}, 64*1024*1024, nil, true),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Strings", "strings",
		textField("name", true, true),
		editorField("content", false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Static_pages", "static_pages",
		textField("title", true, true),
		textField("slug", false, false),
		convertingEditorField("content", true),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Feedbacks", "feedbacks",
		textField("name", true, true),
		emailField("email", true),
		urlField("refer_to", true),
		convertingEditorField("message", true),
		boolField("handled", false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("Postcards", "postcards",
		textField("sender_name", true, true),
		emailField("sender_email", true),
		textField("recipients", true, false),
		editorField("message", true),
		relationField("image_id", "artworks", 1, 1, false, false),
		boolField("notify_sender", false),
		selectField("status", []string{"queued", "sent", "received", "cancelled"}, true),
		dateField("sent_at", false),
		textField("correlation_id", false, false),
		dateField("received_at", false),
	)); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("postcard_deliveries", "tracking_postcard_deliveries",
		relationField("postcard", "postcards", 0, 0, true, true),
		textField("recipient", true, false),
		selectField("status", []string{"pending", "sent", "cancelled"}, true),
		dateField("sent_at", false),
		dateField("cancelled_at", false),
	)); err != nil {
		return err
	}
	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		return err
	}
	deliveries.Indexes = append(deliveries.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_delivery_recipient` ON `postcard_deliveries` (postcard, recipient)",
		"CREATE INDEX `pbx_postcard_delivery_status` ON `postcard_deliveries` (postcard, status)",
	)
	if err := app.Save(deliveries); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("postcard_delivery_attempts", "tracking_postcard_delivery_attempts",
		relationField("delivery", "tracking_postcard_deliveries", 0, 0, true, true),
		numberField("sequence", true),
		selectField("status", []string{"queued", "processing", "processed", "dead_lettered", "cancelled"}, true),
		textField("correlation_id", true, false),
		textField("message_id", true, false),
		numberField("attempt_count", false),
		numberField("max_attempts", true),
		dateField("available_at", true),
		textField("claim_token", false, false),
		dateField("claim_expires_at", false),
		dateField("transport_started_at", false),
		dateField("last_attempt_at", false),
		dateField("processed_at", false),
		dateField("dead_lettered_at", false),
		textField("result_code", false, false),
		textField("last_error_class", false, false),
		boolField("last_error_retryable", false),
		textField("last_error_summary", false, false),
		selectField("resolution_code", []string{"replayed_unmodified", "resolved_manually", "closed_without_replay", "ignored_duplicate"}, false),
		textField("resolution_summary", false, false),
		dateField("resolved_at", false),
	)); err != nil {
		return err
	}
	attempts, err := app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if err != nil {
		return err
	}
	attempts.Fields.Add(relationField("replay_of", "tracking_postcard_delivery_attempts", 0, 0, false, false))
	attempts.Indexes = append(attempts.Indexes,
		"CREATE UNIQUE INDEX `pbx_postcard_attempt_sequence` ON `postcard_delivery_attempts` (delivery, sequence)",
		"CREATE UNIQUE INDEX `pbx_postcard_attempt_message_id` ON `postcard_delivery_attempts` (message_id)",
		"CREATE INDEX `pbx_postcard_attempt_due` ON `postcard_delivery_attempts` (status, available_at, id)",
		"CREATE INDEX `pbx_postcard_attempt_expired` ON `postcard_delivery_attempts` (status, claim_expires_at)",
	)
	if err := app.Save(attempts); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("contributor_snapshots", "tracking_contributor_snapshots",
		textField("key", true, false),
		jsonField("payload", true),
	)); err != nil {
		return err
	}
	snapshots, err := app.FindCollectionByNameOrId("tracking_contributor_snapshots")
	if err != nil {
		return err
	}
	snapshots.Indexes = append(snapshots.Indexes, "CREATE UNIQUE INDEX `pbx_contributor_snapshot_key` ON `contributor_snapshots` (key)")
	if err := app.Save(snapshots); err != nil {
		return err
	}
	if err := saveCurrentCollection(app, currentCollection("contributor_refresh_executions", "tracking_contributor_refresh_executions",
		textField("run_id", true, false),
		numberField("attempt", true),
		numberField("max_attempts", true),
		selectField("status", []string{"processing", "succeeded", "failed"}, true),
		dateField("claim_expires_at", true),
		dateField("completed_at", false),
		numberField("snapshot_count", false),
		textField("error_class", false, false),
		boolField("error_retryable", false),
	)); err != nil {
		return err
	}
	executions, err := app.FindCollectionByNameOrId("tracking_contributor_refresh_executions")
	if err != nil {
		return err
	}
	executions.Indexes = append(executions.Indexes,
		"CREATE UNIQUE INDEX `pbx_contributor_refresh_attempt` ON `contributor_refresh_executions` (run_id, attempt)",
		"CREATE UNIQUE INDEX `pbx_contributor_refresh_active` ON `contributor_refresh_executions` (status) WHERE status = 'processing'",
	)

	return app.Save(executions)
}

func currentCollection(name string, id string, fields ...core.Field) *core.Collection {
	collection := core.NewBaseCollection(name)
	collection.Id = id
	collection.MarkAsNew()
	collection.Fields.Add(fields...)
	collection.Fields.Add(
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)

	return collection
}

func saveCurrentCollection(app core.App, collection *core.Collection) error {
	return app.Save(collection)
}

func textField(name string, required bool, presentable bool) *core.TextField {
	return &core.TextField{Name: name, Required: required, Presentable: presentable}
}

func textFieldWithMax(name string, required bool, presentable bool, max int) *core.TextField {
	return &core.TextField{Name: name, Required: required, Presentable: presentable, Max: max}
}

func numberField(name string, required bool) *core.NumberField {
	return &core.NumberField{Name: name, Required: required}
}

func boolField(name string, required bool) *core.BoolField {
	return &core.BoolField{Name: name, Required: required}
}

func dateField(name string, required bool) *core.DateField {
	return &core.DateField{Name: name, Required: required}
}

func emailField(name string, required bool) *core.EmailField {
	return &core.EmailField{Name: name, Required: required}
}

func urlField(name string, required bool) *core.URLField {
	return &core.URLField{Name: name, Required: required}
}

func editorField(name string, required bool) *core.EditorField {
	return &core.EditorField{Name: name, Required: required}
}

func convertingEditorField(name string, required bool) *core.EditorField {
	return &core.EditorField{Name: name, Required: required, ConvertURLs: true}
}

func jsonField(name string, required bool) *core.JSONField {
	return &core.JSONField{Name: name, Required: required}
}

func selectField(name string, values []string, required bool) *core.SelectField {
	return &core.SelectField{Name: name, Values: values, Required: required, MaxSelect: 1}
}

func relationField(name string, collectionID string, minSelect int, maxSelect int, required bool, cascadeDelete bool) *core.RelationField {
	return &core.RelationField{
		Name:          name,
		CollectionId:  collectionID,
		MinSelect:     minSelect,
		MaxSelect:     maxSelect,
		Required:      required,
		CascadeDelete: cascadeDelete,
	}
}

func fileField(name string, mimeTypes []string, maxSize int64, thumbs []string, required bool) *core.FileField {
	return &core.FileField{
		Name:      name,
		MimeTypes: mimeTypes,
		MaxSize:   maxSize,
		Thumbs:    thumbs,
		Required:  required,
	}
}
