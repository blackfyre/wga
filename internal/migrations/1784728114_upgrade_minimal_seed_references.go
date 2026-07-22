package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	minimalSeedSchoolName     = "Demonstration School"
	minimalSeedSchoolSlug     = "demonstration-school"
	minimalSeedWelcomeContent = "<p>Welcome to this demonstration collection. It contains fictional starter content for exploring artists, artworks, search, and comparison.</p>"
	minimalSeedPrivacyTitle   = "Privacy policy"
	minimalSeedPrivacyContent = "<p>This development instance contains fictional starter content only.</p>"
	referenceSchoolName       = "American"
	referenceSchoolSlug       = "american"
	referencePrivacyPageTitle = "Privacy Policy"
	seedPrivacyPageContent    = "<p>Web Gallery of Art respects visitor privacy. This starter database contains fictional data for testing only; configure the production privacy policy before collecting visitor information.</p>"
	referenceWelcomeContent   = `<p>
The Web Gallery of Art is a searchable collection of European painting, sculpture, decorative arts, and architecture from the 3rd century to the early 20th century. It began as a Renaissance-focused project and grew into a broader archive built for students, teachers, and curious visitors who want images and context in the same place.
</p>
<p>
This version of the site is organized around a few strong routes. Start with the <a href="/artists">artists</a> index for biographies and related works, move into <a href="/artworks">artwork search</a> when you know what you want to filter, open <a href="/dual-mode" target="_blank">Dual Mode</a> to compare two pages side by side, or use <a href="/inspire">Inspiration</a> when you want the collection to surprise you.
</p>
<p>
The project remains an independent public resource: open, interconnected, and designed for study as much as enjoyment. If the collection helps you, leave a note in the <a href="/guestbook">guestbook</a> or explore how the site is maintained by its <a href="/contributors">contributors</a>.
</p>`
)

func init() {
	m.Register(upgradeMinimalSeedReferences, func(app core.App) error {
		// Seed records may have been edited after migration; preserve them on rollback.
		return nil
	})
}

func upgradeMinimalSeedReferences(app core.App) error {
	return app.RunInTransaction(func(txApp core.App) error {
		school, err := findSeedRecord(txApp, "schools", "slug", minimalSeedSchoolSlug)
		if err != nil {
			return err
		}
		if school != nil && school.GetString("name") == minimalSeedSchoolName {
			school.Set("name", referenceSchoolName)
			school.Set("slug", referenceSchoolSlug)
			if err := txApp.Save(school); err != nil {
				return err
			}
		}

		welcome, err := findSeedRecord(txApp, "strings", "name", "welcome")
		if err != nil {
			return err
		}
		if welcome != nil && welcome.GetString("content") == minimalSeedWelcomeContent {
			welcome.Set("content", referenceWelcomeContent)
			if err := txApp.Save(welcome); err != nil {
				return err
			}
		}

		privacyPage, err := findSeedRecord(txApp, "static_pages", "slug", "privacy-policy")
		if err != nil {
			return err
		}
		if privacyPage != nil && privacyPage.GetString("title") == minimalSeedPrivacyTitle && privacyPage.GetString("content") == minimalSeedPrivacyContent {
			privacyPage.Set("title", referencePrivacyPageTitle)
			privacyPage.Set("content", seedPrivacyPageContent)
			return txApp.Save(privacyPage)
		}

		return nil
	})
}

func findSeedRecord(app core.App, collection string, field string, value string) (*core.Record, error) {
	record, err := app.FindFirstRecordByData(collection, field, value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return record, err
}
