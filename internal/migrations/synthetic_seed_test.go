package migrations

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSyntheticSeedMigrationImportsBaselineSchema(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	assertSyntheticCollectionCounts(t, app, map[string]int{
		"schools":        26,
		"art_forms":      14,
		"art_types":      3,
		"art_periods":    32,
		"artists":        10,
		"artworks":       27,
		"glossary":       5,
		"guestbook":      2,
		"music_composer": 2,
		"music_song":     3,
		"strings":        7,
		"static_pages":   1,
	})

	for _, collectionName := range []string{"professions", "biographies", "biography_links", "source_attributions"} {
		_, err := app.FindCollectionByNameOrId(collectionName)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no %s collection, got %v", collectionName, err)
		}
	}

	for _, collection := range []struct {
		id   string
		name string
	}{
		{id: "artists", name: "Artists"},
		{id: "artworks", name: "Artworks"},
		{id: "postcards", name: "Postcards"},
		{id: "tracking_postcard_deliveries", name: "postcard_deliveries"},
		{id: "tracking_postcard_delivery_attempts", name: "postcard_delivery_attempts"},
		{id: "tracking_contributor_snapshots", name: "contributor_snapshots"},
		{id: "tracking_contributor_refresh_executions", name: "contributor_refresh_executions"},
	} {
		item, err := app.FindCollectionByNameOrId(collection.id)
		if err != nil {
			t.Fatalf("find %s collection: %v", collection.id, err)
		}
		if got := item.Name; got != collection.name {
			t.Fatalf("expected %s collection name %q, got %q", collection.id, collection.name, got)
		}
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	if got, want := users.Id, "_pb_users_auth_"; got != want {
		t.Fatalf("expected users collection ID %q, got %q", want, got)
	}
	for _, rule := range []*string{users.ListRule, users.ViewRule, users.UpdateRule, users.DeleteRule} {
		if rule == nil || *rule != "id = @request.auth.id" {
			t.Fatal("expected users self-only access rules")
		}
	}

	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	for _, fieldName := range []string{"source_path", "source_hash", "debug_hash", "professions"} {
		if artists.Fields.GetByName(fieldName) != nil {
			t.Fatalf("expected no %s artist field", fieldName)
		}
	}
	assertIndex(t, artists, "pbx_artist_published_name")

	schools, err := app.FindCollectionByNameOrId("schools")
	if err != nil {
		t.Fatalf("find schools collection: %v", err)
	}
	assertIndex(t, schools, "idx_unique_school")

	for _, item := range []struct {
		collection string
		field      string
	}{
		{collection: "static_pages", field: "content"},
		{collection: "feedbacks", field: "message"},
	} {
		collection, err := app.FindCollectionByNameOrId(item.collection)
		if err != nil {
			t.Fatalf("find %s collection: %v", item.collection, err)
		}
		field, ok := collection.Fields.GetByName(item.field).(*core.EditorField)
		if !ok || !field.ConvertURLs {
			t.Fatalf("expected %s.%s to convert URLs", item.collection, item.field)
		}
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}
	for _, fieldName := range []string{"date_text", "source_url", "source_image_path"} {
		if artworks.Fields.GetByName(fieldName) != nil {
			t.Fatalf("expected no %s artwork field", fieldName)
		}
	}
	assertIndex(t, artworks, "pbx_artwork_published_title")

	attempts, err := app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if err != nil {
		t.Fatalf("find postcard attempts collection: %v", err)
	}
	assertIndex(t, attempts, "pbx_postcard_attempt_due")

	period, err := app.FindRecordById("art_periods", "3s9ubtzmmukygar")
	if err != nil {
		t.Fatalf("find art period: %v", err)
	}
	if period.GetString("slug") == "" || period.GetString("description") == "" {
		t.Fatal("expected art period slug and description")
	}

	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	if got, want := artist.GetString("name"), "Synthetic Artist 02"; got != want {
		t.Fatalf("expected artist name %q, got %q", want, got)
	}
	if !strings.Contains(artist.GetString("bio"), "Synthetic Artist 02") {
		t.Fatal("expected artist biography HTML to be imported")
	}
	if len(artist.GetStringSlice("school")) != 1 {
		t.Fatal("expected artist school relation")
	}

	artwork, err := app.FindRecordById("artworks", "07561d2efd0a6db")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	if artwork.GetString("image") == "" {
		t.Fatal("expected artwork image")
	}

	song, err := app.FindRecordById("music_song", "72d6bb922f76aea")
	if err != nil {
		t.Fatalf("find music track: %v", err)
	}
	if song.GetString("source") == "" {
		t.Fatal("expected music source")
	}
}

func TestSyntheticSeedMigrationSkipsPopulatedTarget(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	stringsCollection, err := app.FindCollectionByNameOrId("strings")
	if err != nil {
		t.Fatalf("find strings collection: %v", err)
	}
	custom := core.NewRecord(stringsCollection)
	custom.Set("name", "existing")
	custom.Set("content", "Existing content")
	if err := app.Save(custom); err != nil {
		t.Fatalf("create existing content: %v", err)
	}

	if err := seedCurrentSyntheticData(app); err != nil {
		t.Fatalf("skip populated target: %v", err)
	}
	if _, err := app.FindRecordById("strings", custom.Id); err != nil {
		t.Fatalf("find existing content: %v", err)
	}
}

func assertSyntheticCollectionCounts(t *testing.T, app core.App, expected map[string]int) {
	t.Helper()

	for collectionName, want := range expected {
		records, err := app.FindRecordsByFilter(collectionName, "", "", 0, 0)
		if err != nil {
			t.Fatalf("find %s records: %v", collectionName, err)
		}
		if got := len(records); got != want {
			t.Fatalf("expected %d %s records, got %d", want, collectionName, got)
		}
	}
}

func assertIndex(t *testing.T, collection *core.Collection, name string) {
	t.Helper()

	for _, index := range collection.Indexes {
		if strings.Contains(index, name) {
			return
		}
	}
	t.Fatalf("expected %s index", name)
}
