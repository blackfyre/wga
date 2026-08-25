package keyboard

import (
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestValidQueryRequiresTwoRunes(t *testing.T) {
	if validQuery("a") {
		t.Fatal("one rune query must be rejected")
	}
	if !validQuery("éx") {
		t.Fatal("two rune query must be accepted")
	}
}

func TestRequestLimiterResetsAfterOneMinute(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	limiter := newRequestLimiter()
	limiter.now = func() time.Time { return now }

	for range requestsPerMinute {
		if !limiter.allow("127.0.0.1") {
			t.Fatal("request within limit was rejected")
		}
	}
	if limiter.allow("127.0.0.1") {
		t.Fatal("request above limit was accepted")
	}

	now = now.Add(time.Minute)
	if !limiter.allow("127.0.0.1") {
		t.Fatal("request after rate window reset was rejected")
	}
}

func TestRequestLimiterRemovesExpiredClients(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	limiter := newRequestLimiter()
	limiter.now = func() time.Time { return now }
	limiter.clients["expired"] = requestWindow{started: now.Add(-time.Minute)}

	if !limiter.allow("current") {
		t.Fatal("request within limit was rejected")
	}
	if _, ok := limiter.clients["expired"]; ok {
		t.Fatal("expired client was not removed")
	}
}

func TestSuggestionLimitForBoundsRequestedCapacity(t *testing.T) {
	if got := suggestionLimitFor("3"); got != 3 {
		t.Fatalf("limit = %d, want 3", got)
	}
	if got := suggestionLimitFor("0"); got != suggestionLimit {
		t.Fatalf("zero limit = %d, want %d", got, suggestionLimit)
	}
	if got := suggestionLimitFor("99"); got != suggestionLimit {
		t.Fatalf("capped limit = %d, want %d", got, suggestionLimit)
	}
}

func TestArtistLabelIncludesKnownLifespan(t *testing.T) {
	if got := artistLabel("Vermeer", 1632, 1675); got != "Vermeer · 1632–1675" {
		t.Fatalf("artist label = %q", got)
	}
}

func TestSuggestionsStopsWhenArtistsFillBudget(t *testing.T) {
	app := newKeyboardTestApp(t)
	for _, name := range []string{"Art Alpha", "Art Bravo", "Art Charlie", "Art Delta", "Art Echo", "Art Foxtrot", "Art Golf"} {
		saveSuggestionArtist(t, app, name)
	}
	artist := saveSuggestionArtist(t, app, "Author")
	for _, title := range []string{"Art Work A", "Art Work B", "Art Work C"} {
		saveSuggestionArtwork(t, app, artist.Id, title)
	}

	rows, err := suggestions(app, "art", 7)
	if err != nil {
		t.Fatalf("get suggestions: %v", err)
	}
	if len(rows) > 7 {
		t.Fatalf("suggestion count = %d, must not exceed 7", len(rows))
	}
	if len(rows) != 7 {
		t.Fatalf("suggestion count = %d, want 7", len(rows))
	}
	for _, row := range rows {
		if row.Kind != "ARTIST" {
			t.Fatalf("suggestion kind = %q, want ARTIST", row.Kind)
		}
	}
}

func TestSuggestionsFillsRemainingBudgetWithOrderedArtworks(t *testing.T) {
	app := newKeyboardTestApp(t)
	artist := saveSuggestionArtist(t, app, "Artisan")
	for _, title := range []string{"Art Work C", "Art Work A", "Art Work B"} {
		saveSuggestionArtwork(t, app, artist.Id, title)
	}

	rows, err := suggestions(app, "art", 3)
	if err != nil {
		t.Fatalf("get suggestions: %v", err)
	}
	if len(rows) > 3 {
		t.Fatalf("suggestion count = %d, must not exceed 3", len(rows))
	}
	if len(rows) != 3 {
		t.Fatalf("suggestion count = %d, want 3", len(rows))
	}
	if rows[0].Kind != "ARTIST" || rows[0].Label != "Artisan" {
		t.Fatalf("first suggestion = %#v, want artist Artisan", rows[0])
	}
	if rows[1].Kind != "WORK" || rows[1].Label != "Art Work A · Artisan" {
		t.Fatalf("second suggestion = %#v, want Art Work A", rows[1])
	}
	if rows[2].Kind != "WORK" || rows[2].Label != "Art Work B · Artisan" {
		t.Fatalf("third suggestion = %#v, want Art Work B", rows[2])
	}
}

func newKeyboardTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset test app: %v", err)
		}
	})

	artists := core.NewBaseCollection(constants.CollectionArtists)
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("create artists collection: %v", err)
	}

	artworks := core.NewBaseCollection(constants.CollectionArtworks)
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.BoolField{Name: "published"},
		&core.RelationField{Name: "author", CollectionId: artists.Id, MaxSelect: 1},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks collection: %v", err)
	}

	return app
}

func saveSuggestionArtist(t *testing.T, app core.App, name string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtists)
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("published", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist: %v", err)
	}

	return record
}

func saveSuggestionArtwork(t *testing.T, app core.App, artistID string, title string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("author", artistID)
	record.Set("published", true)
	record.Set("title", title)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
}
