package migrations

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The remembered grid/list view now applies to every listing, not only to
// artwork search, so the privacy policy names it alongside the search toolbar.
const (
	privacyPreferenceSentenceBeforeListingView = "With your permission, this device remembers your artwork search toolbar, colour scheme, palette, reading aid and Study Board, in this browser's local storage and, for the search toolbar, a cookie."
	privacyPreferenceSentenceWithListingView   = "With your permission, this device remembers your artwork search toolbar, the grid or list view you last chose on any listing, your colour scheme, palette, reading aid and Study Board, in this browser's local storage and, for the search toolbar and listing view, a cookie."
)

func init() {
	m.Register(describeListingViewInPrivacyPolicy, func(core.App) error {
		return errors.New("the privacy policy listing-view wording cannot be restored")
	})
}

// describeListingViewInPrivacyPolicy rewrites only the shipped preference
// sentence. A page without that exact sentence, such as one an editor has
// changed, is left as it is.
func describeListingViewInPrivacyPolicy(app core.App) error {
	record, err := app.FindFirstRecordByData(constants.CollectionStaticPages, "slug", privacyPolicySlug)
	if errors.Is(err, sql.ErrNoRows) {
		app.Logger().Info("privacy policy listing view wording left unchanged: no privacy policy page",
			"event", "migration.privacy_listing_view.skipped")
		return nil
	}
	if err != nil {
		return err
	}

	content := record.GetString("content")
	if !strings.Contains(content, privacyPreferenceSentenceBeforeListingView) {
		app.Logger().Info("privacy policy listing view wording left unchanged: shipped sentence not found",
			"event", "migration.privacy_listing_view.skipped")
		return nil
	}
	record.Set("content", strings.Replace(content, privacyPreferenceSentenceBeforeListingView, privacyPreferenceSentenceWithListingView, 1))
	return app.Save(record)
}
