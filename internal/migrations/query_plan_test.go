package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestBaselineIndexesSupportCatalogueScaleQueries(t *testing.T) {
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

	for _, query := range []string{
		`WITH RECURSIVE sequence(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM sequence WHERE n < 5000) INSERT INTO Artists (id, name, slug, published) SELECT printf('artist%09d', n), printf('Artist %d', n), printf('artist-%d', n), true FROM sequence`,
		`WITH RECURSIVE sequence(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM sequence WHERE n < 56000) INSERT INTO Artworks (id, title, published) SELECT printf('artwork%08d', n), printf('Artwork %d', n), true FROM sequence`,
	} {
		if _, err := app.DB().NewQuery(query).Execute(); err != nil {
			t.Fatalf("insert catalogue-scale records: %v", err)
		}
	}

	assertQueryUsesIndex(t, app, `EXPLAIN QUERY PLAN SELECT id FROM Artists WHERE published = true ORDER BY name, id LIMIT 30`, "pbx_artist_published_name")
	assertQueryUsesIndex(t, app, `EXPLAIN QUERY PLAN SELECT id FROM Artworks WHERE published = true ORDER BY title, id LIMIT 16`, "pbx_artwork_published_title")
	assertQueryUsesIndex(t, app, `EXPLAIN QUERY PLAN SELECT id FROM postcard_delivery_attempts WHERE status = 'queued' AND available_at <= CURRENT_TIMESTAMP ORDER BY available_at, id LIMIT 1`, "pbx_postcard_attempt_due")
}

func assertQueryUsesIndex(t *testing.T, app core.App, query string, index string) {
	t.Helper()

	var plan []struct {
		Detail string `db:"detail"`
	}
	if err := app.DB().NewQuery(query).All(&plan); err != nil {
		t.Fatalf("explain query plan: %v", err)
	}
	for _, row := range plan {
		if strings.Contains(row.Detail, index) {
			return
		}
	}
	t.Fatalf("expected query plan to use %s, got %#v", index, plan)
}
