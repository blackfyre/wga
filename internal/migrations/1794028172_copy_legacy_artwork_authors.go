package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(copyLegacyArtworkAuthors, func(core.App) error {
		// The copy is non-destructive: the legacy author relation is untouched,
		// so there is nothing to undo, and clearing primary_author could discard
		// a curator's later assignment.
		return nil
	})
}

// legacyAuthorReviewFilter is the PocketBase dashboard filter that lists the
// artworks the legacy-author migration could not assign: no primary author yet
// and several legacy authors. Curators use it to review the same set that the
// migration reports.
const legacyAuthorReviewFilter = "primary_author = '' && author:length > 1"

// legacyAuthorMigrationReport summarises one run of copyLegacyArtworkAuthors.
// Copied counts the artworks whose single legacy author became the primary
// author; ReviewIDs lists, in id order, the artworks with several legacy
// authors that retain their legacy assignment for curator review.
type legacyAuthorMigrationReport struct {
	Copied    int64
	ReviewIDs []string
}

// copyLegacyArtworkAuthors copies each artwork's single legacy author to
// primary_author and reports artworks with several legacy authors.
func copyLegacyArtworkAuthors(app core.App) error {
	report, err := migrateLegacyArtworkAuthors(app)
	if err != nil {
		return err
	}

	app.Logger().Info("Legacy artwork authors copied to primary_author",
		"event", "migration.legacy_author.copied",
		"copied", report.Copied,
		"review_count", len(report.ReviewIDs),
	)
	if len(report.ReviewIDs) > 0 {
		app.Logger().Warn("Artworks with several legacy authors need a curated primary author",
			"event", "migration.legacy_author.review",
			"review_count", len(report.ReviewIDs),
			"artwork_ids", report.ReviewIDs,
			"dashboard_filter", legacyAuthorReviewFilter,
		)
	}

	return nil
}

// migrateLegacyArtworkAuthors performs the copy. It never overwrites an
// existing primary author, never chooses between several legacy authors, and
// never modifies the legacy author relation, so it is safe to repeat.
func migrateLegacyArtworkAuthors(app core.App) (legacyAuthorMigrationReport, error) {
	report := legacyAuthorMigrationReport{}

	result, err := app.DB().NewQuery(`
		UPDATE Artworks
		SET primary_author = json_extract(author, '$[0]')
		WHERE COALESCE(primary_author, '') = ''
			AND json_valid(author)
			AND json_array_length(author) = 1
			AND COALESCE(json_extract(author, '$[0]'), '') != ''
	`).Execute()
	if err != nil {
		return report, err
	}
	if report.Copied, err = result.RowsAffected(); err != nil {
		return report, err
	}

	rows := []struct {
		ID string `db:"id"`
	}{}
	if err := app.DB().NewQuery(`
		SELECT id
		FROM Artworks
		WHERE COALESCE(primary_author, '') = ''
			AND json_valid(author)
			AND json_array_length(author) > 1
		ORDER BY id
	`).All(&rows); err != nil {
		return report, err
	}
	for _, row := range rows {
		report.ReviewIDs = append(report.ReviewIDs, row.ID)
	}

	return report, nil
}
