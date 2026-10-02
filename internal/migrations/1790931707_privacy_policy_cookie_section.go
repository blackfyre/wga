package migrations

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	privacyPolicySlug = "privacy-policy"
	// legacyPrivacyCookieHeading opens the generic boilerplate section that
	// claimed cookies record visited pages and customise content.
	legacyPrivacyCookieHeading = "<h2>Cookies and Web Beacons</h2>"
)

// privacyCookieSection describes cookie and storage use under the consent
// model: essential cookies for the session and the consent record, optional
// consent-gated preference storage, and no advertising or tracking cookies.
const privacyCookieSection = `<h2>Cookies and browser storage</h2>

<p>Essential cookies keep this site working. They hold your session and the record of your cookie choice, and they cannot be switched off.</p>

<p>Reading preferences are optional. With your permission, this device remembers your artwork search toolbar, colour scheme, palette, reading aid and Study Board, in this browser's local storage and, for the search toolbar, a cookie. Choosing DENY, or switching preference storage off in Cookie settings, deletes them, and your choices then last for the current visit only.</p>

<p>No advertising or cross-site tracking cookies are set, and analytics cookies are not in use.</p>

`

func init() {
	m.Register(replacePrivacyCookieSection, func(core.App) error {
		return errors.New("the replaced privacy policy cookie section cannot be restored")
	})
}

// replacePrivacyCookieSection replaces only the privacy policy's legacy cookie
// section, up to the next heading, and leaves every other section untouched.
// A page without that section is left as it is.
func replacePrivacyCookieSection(app core.App) error {
	record, err := app.FindFirstRecordByData(constants.CollectionStaticPages, "slug", privacyPolicySlug)
	if errors.Is(err, sql.ErrNoRows) {
		app.Logger().Info("privacy policy cookie section left unchanged: no privacy policy page",
			"event", "migration.privacy_cookie_section.skipped")
		return nil
	}
	if err != nil {
		return err
	}

	content, changed := withPrivacyCookieSection(record.GetString("content"))
	if !changed {
		app.Logger().Warn("privacy policy cookie section left unchanged: legacy section not found",
			"event", "migration.privacy_cookie_section.skipped")
		return nil
	}
	record.Set("content", content)
	return app.Save(record)
}

// withPrivacyCookieSection returns content with the legacy cookie section
// replaced, and whether it was found.
func withPrivacyCookieSection(content string) (string, bool) {
	start := strings.Index(content, legacyPrivacyCookieHeading)
	if start < 0 {
		return content, false
	}
	rest := content[start+len(legacyPrivacyCookieHeading):]
	end := len(content)
	if next := strings.Index(strings.ToLower(rest), "<h2"); next >= 0 {
		end = start + len(legacyPrivacyCookieHeading) + next
	}
	return content[:start] + privacyCookieSection + content[end:], true
}
