package migrations

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// maxReportedToneKeywordIDs bounds how many affected artwork ids are surfaced
// when the migration refuses to retire tone_keywords. The total count is still
// reported in full; the id list exists to make the refusal actionable without
// echoing source content or dumping an unbounded id list into the error.
const maxReportedToneKeywordIDs = 10

func init() {
	m.Register(retireArtworkToneKeywords, restoreArtworkToneKeywords)
}

// authoritativeToneKeywordsFilter matches artwork rows whose tone_keywords
// value must be retained. SQL NULL and the canonical empty JSON forms — null,
// "", [], {} — plus a blank column are safe to discard; every other stored
// value, including malformed or whitespace-padded JSON, is authoritative.
const authoritativeToneKeywordsFilter = `
	tone_keywords IS NOT NULL
	AND CAST(tone_keywords AS TEXT) NOT IN ('null', '""', '[]', '{}', '')
`

// retireArtworkToneKeywords completes the approved deferral of tone-keyword
// exploration. It only removes an empty field: source-backed values must be
// retained until a separately approved migration defines their future use.
func retireArtworkToneKeywords(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}
	if artworks.Fields.GetByName("tone_keywords") == nil {
		return nil
	}

	count, err := countAuthoritativeToneKeywords(app)
	if err != nil {
		return err
	}
	if count > 0 {
		ids, err := affectedToneKeywordIDs(app, maxReportedToneKeywordIDs)
		if err != nil {
			return err
		}
		return refuseToneKeywordRetirement(count, ids)
	}

	artworks.Fields.RemoveByName("tone_keywords")
	return app.Save(artworks)
}

// countAuthoritativeToneKeywords reports how many artwork rows hold a
// non-empty tone_keywords value and therefore block field removal.
func countAuthoritativeToneKeywords(app core.App) (int, error) {
	var count int
	if err := app.DB().NewQuery(
		"SELECT COUNT(*) FROM Artworks WHERE" + authoritativeToneKeywordsFilter,
	).Row(&count); err != nil {
		return 0, fmt.Errorf("inspect artwork tone keywords: %w", err)
	}
	return count, nil
}

// affectedToneKeywordIDs returns a bounded, deterministically ordered sample of
// the artwork ids that hold authoritative tone_keywords values. Source content
// is deliberately not selected, so the refusal never echoes stored values.
func affectedToneKeywordIDs(app core.App, limit int) ([]string, error) {
	var ids []string
	if err := app.DB().NewQuery(
		"SELECT id FROM Artworks WHERE" + authoritativeToneKeywordsFilter +
			" ORDER BY id LIMIT {:limit}",
	).Bind(dbx.Params{"limit": limit}).Column(&ids); err != nil {
		return nil, fmt.Errorf("collect artwork tone keyword ids: %w", err)
	}
	return ids, nil
}

// refuseToneKeywordRetirement formats a fail-closed error carrying the full
// count of authoritative records and a capped list of their ids.
func refuseToneKeywordRetirement(count int, ids []string) error {
	sample := strings.Join(ids, ", ")
	if count > len(ids) {
		sample += ", ..."
	}
	return fmt.Errorf(
		"cannot retire tone_keywords: %d artwork records contain authoritative values (ids: %s)",
		count,
		sample,
	)
}

// restoreArtworkToneKeywords supports migration down/up verification. The
// field remains empty because the forward migration refuses to discard values.
func restoreArtworkToneKeywords(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}
	if artworks.Fields.GetByName("tone_keywords") != nil {
		return nil
	}

	artworks.Fields.Add(jsonField("tone_keywords", false))
	return app.Save(artworks)
}
