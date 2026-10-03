package migrations

import (
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

const (
	privacyBefore = "<h1>Privacy Policy</h1>\n\n<h2>Log Files</h2>\n\n<p>Standard log files.</p>\n\n"
	privacyLegacy = "<h2>Cookies and Web Beacons</h2>\n\n<p>Like any other website, Web Gallery of Art uses \"cookies\".</p>\n\n"
	privacyAfter  = "<h2>Advertising Partners Privacy Policies</h2>\n\n<p>Partners.</p>\n"
)

func TestPrivacyCookieSectionReplacesOnlyTheLegacySection(t *testing.T) {
	app := newPrivacyPolicyApp(t, privacyBefore+privacyLegacy+privacyAfter)

	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("replace cookie section: %v", err)
	}

	content := privacyPolicyContent(t, app)
	if content != privacyBefore+privacyCookieSection+privacyAfter {
		t.Fatalf("unexpected privacy policy content:\n%s", content)
	}
	for _, expected := range []string{
		"hold your session and the record of your cookie choice",
		"Reading preferences are optional",
		"No advertising or cross-site tracking cookies are set",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected privacy policy to contain %q", expected)
		}
	}
	for _, removed := range []string{"Web Beacons", "period music"} {
		if strings.Contains(content, removed) {
			t.Fatalf("expected privacy policy not to contain %q", removed)
		}
	}
}

func TestPrivacyCookieSectionReplacesAFinalSection(t *testing.T) {
	app := newPrivacyPolicyApp(t, privacyBefore+privacyLegacy)

	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("replace final cookie section: %v", err)
	}
	if content := privacyPolicyContent(t, app); content != privacyBefore+privacyCookieSection {
		t.Fatalf("unexpected privacy policy content:\n%s", content)
	}
}

func TestPrivacyCookieSectionLeavesAPageWithoutTheLegacySection(t *testing.T) {
	edited := privacyBefore + "<h2>Cookies</h2>\n\n<p>Edited by an editor.</p>\n\n" + privacyAfter
	app := newPrivacyPolicyApp(t, edited)

	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("replace cookie section: %v", err)
	}
	if content := privacyPolicyContent(t, app); content != edited {
		t.Fatalf("expected an edited page to stay unchanged, got:\n%s", content)
	}
}

func TestPrivacyCookieSectionIsStableWhenRunAgain(t *testing.T) {
	app := newPrivacyPolicyApp(t, privacyBefore+privacyLegacy+privacyAfter)

	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("replace cookie section: %v", err)
	}
	first := privacyPolicyContent(t, app)
	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("replace cookie section again: %v", err)
	}
	if again := privacyPolicyContent(t, app); again != first {
		t.Fatalf("expected a second run to change nothing, got:\n%s", again)
	}
}

func TestPrivacyCookieSectionSkipsAMissingPage(t *testing.T) {
	app := newPrivacyPolicyApp(t, "")

	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("expected a missing privacy policy page to be skipped: %v", err)
	}
}

// newPrivacyPolicyApp creates the current schema and, unless content is empty,
// a privacy policy page holding content.
func newPrivacyPolicyApp(t *testing.T, content string) *core.BaseApp {
	t.Helper()
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ClearBootstrap(); err != nil {
			t.Error(err)
		}
	})
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create baseline schema: %v", err)
	}
	if content == "" {
		return app
	}
	collection, err := app.FindCollectionByNameOrId(constants.CollectionStaticPages)
	if err != nil {
		t.Fatalf("find static pages: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("title", "Privacy Policy")
	record.Set("slug", privacyPolicySlug)
	record.Set("content", content)
	if err := app.Save(record); err != nil {
		t.Fatalf("save privacy policy: %v", err)
	}
	return app
}

func privacyPolicyContent(t *testing.T, app core.App) string {
	t.Helper()
	record, err := app.FindFirstRecordByData(constants.CollectionStaticPages, "slug", privacyPolicySlug)
	if err != nil {
		t.Fatalf("find privacy policy: %v", err)
	}
	return record.GetString("content")
}
