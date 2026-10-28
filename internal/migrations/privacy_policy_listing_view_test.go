package migrations

import (
	"strings"
	"testing"
)

func TestPrivacyListingViewRewordsTheShippedSentence(t *testing.T) {
	app := newPrivacyPolicyApp(t, privacyBefore+privacyLegacy+privacyAfter)
	if err := replacePrivacyCookieSection(app); err != nil {
		t.Fatalf("replace cookie section: %v", err)
	}

	if err := describeListingViewInPrivacyPolicy(app); err != nil {
		t.Fatalf("describe listing view: %v", err)
	}
	content := privacyPolicyContent(t, app)
	if !strings.Contains(content, privacyPreferenceSentenceWithListingView) || strings.Contains(content, privacyPreferenceSentenceBeforeListingView) {
		t.Fatalf("unexpected privacy policy content:\n%s", content)
	}

	if err := describeListingViewInPrivacyPolicy(app); err != nil {
		t.Fatalf("describe listing view again: %v", err)
	}
	if again := privacyPolicyContent(t, app); again != content {
		t.Fatalf("expected a second run to change nothing, got:\n%s", again)
	}
}

func TestPrivacyListingViewLeavesAnEditedPage(t *testing.T) {
	edited := privacyBefore + "<h2>Cookies</h2>\n\n<p>Edited by an editor.</p>\n\n" + privacyAfter
	app := newPrivacyPolicyApp(t, edited)
	if err := describeListingViewInPrivacyPolicy(app); err != nil {
		t.Fatalf("describe listing view: %v", err)
	}
	if content := privacyPolicyContent(t, app); content != edited {
		t.Fatalf("expected an edited page to stay unchanged, got:\n%s", content)
	}
}

func TestPrivacyListingViewSkipsAMissingPage(t *testing.T) {
	app := newPrivacyPolicyApp(t, "")
	if err := describeListingViewInPrivacyPolicy(app); err != nil {
		t.Fatalf("expected a missing privacy policy page to be skipped: %v", err)
	}
}
