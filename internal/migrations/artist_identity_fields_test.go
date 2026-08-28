package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// TestArtistIdentityFieldsFreshMigration verifies that a fresh bootstrap runs
// the identity-fields migration before the synthetic seed, so seeded artists
// carry the producer-supplied filing and short forms verbatim.
func TestArtistIdentityFieldsFreshMigration(t *testing.T) {
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

	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	for _, fieldName := range []string{"filing_name", "short_name"} {
		if _, ok := artists.Fields.GetByName(fieldName).(*core.TextField); !ok {
			t.Fatalf("expected artists.%s text field", fieldName)
		}
	}

	// Synthetic source artist 02: source_display_name is the filing form and
	// display_name is the supplied short form; both must survive verbatim.
	artist, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find seeded artist: %v", err)
	}
	if got, want := artist.GetString("filing_name"), "SYNTHETIC ARTIST 02"; got != want {
		t.Fatalf("filing_name = %q, want %q", got, want)
	}
	if got, want := artist.GetString("short_name"), "Synthetic Artist 02"; got != want {
		t.Fatalf("short_name = %q, want %q", got, want)
	}
}

// TestArtistIdentityFieldsPriorBootstrapMigration verifies that applying the
// identity-fields migration to an already-seeded prior-bootstrap database adds
// the columns non-destructively without backfilling or fabricating values.
func TestArtistIdentityFieldsPriorBootstrapMigration(t *testing.T) {
	configureMigrations(t)

	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	if artists.Fields.GetByName("filing_name") != nil || artists.Fields.GetByName("short_name") != nil {
		t.Fatal("prior-bootstrap artists collection should not yet carry identity fields")
	}

	existing := core.NewRecord(artists)
	existing.Set("id", "2236bdd57f7492e")
	existing.Set("name", "SYNTHETIC ARTIST 02")
	existing.Set("slug", "synthetic-artist-02")
	existing.Set("known_place_of_birth", "n/a")
	existing.Set("known_place_of_death", "n/a")
	existing.Set("published", true)
	if err := app.Save(existing); err != nil {
		t.Fatalf("save prior-bootstrap artist: %v", err)
	}

	if err := addArtistIdentityFields(app); err != nil {
		t.Fatalf("apply artist identity fields migration: %v", err)
	}

	artists, err = app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("re-find artists collection: %v", err)
	}
	for _, fieldName := range []string{"filing_name", "short_name"} {
		if _, ok := artists.Fields.GetByName(fieldName).(*core.TextField); !ok {
			t.Fatalf("expected artists.%s text field after migration", fieldName)
		}
	}

	preserved, err := app.FindRecordById("artists", "2236bdd57f7492e")
	if err != nil {
		t.Fatalf("find preserved artist: %v", err)
	}
	if got := preserved.GetString("name"); got != "SYNTHETIC ARTIST 02" {
		t.Fatalf("preserved artist name = %q, want intact", got)
	}
	// The identity fields exist but were not backfilled: the prior bootstrap
	// seeded before they existed, and no name may be reconstructed or invented.
	if got := preserved.GetString("filing_name"); got != "" {
		t.Fatalf("prior-bootstrap filing_name = %q, want empty (not fabricated)", got)
	}
	if got := preserved.GetString("short_name"); got != "" {
		t.Fatalf("prior-bootstrap short_name = %q, want empty (not fabricated)", got)
	}
}
