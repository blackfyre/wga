package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func renderContributors(t *testing.T, content ContributorsPageDTO) string {
	t.Helper()

	var output bytes.Buffer
	if err := ContributorsBlock(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render contributors block: %v", err)
	}

	return output.String()
}

func TestContributorsRendersSingleHeadingAndListSemantics(t *testing.T) {
	rendered := renderContributors(t, ContributorsPageDTO{Contributors: []GithubContributor{
		{Login: "blackfyre", AvatarURL: "https://avatars.githubusercontent.com/u/1?v=4", HTMLURL: "https://github.com/blackfyre", Contributions: 585},
	}})

	for _, expected := range []string{
		"11 — CONTRIBUTORS",
		">Contributors</h1>",
		"The people who have contributed to the Web Gallery of Art.",
		"text-(length:--t-32)",
		"md:text-(length:--t-44)",
		"CODE CONTRIBUTORS",
		"1 PEOPLE",
		"@blackfyre",
		"585 commits",
		`src="https://avatars.githubusercontent.com/u/1?v=4"`,
		`href="https://github.com/blackfyre"`,
		`<ul role="list"`,
		`<article`,
		`<li`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered contributors page does not contain %q\ngot: %s", expected, rendered)
		}
	}

	if strings.Count(rendered, "<h1") != 1 {
		t.Errorf("h1 count = %d, want 1", strings.Count(rendered, "<h1"))
	}
}

func TestContributorsOmitsUnsupportedEditorial(t *testing.T) {
	rendered := renderContributors(t, ContributorsPageDTO{Contributors: []GithubContributor{
		{Login: "blackfyre", AvatarURL: "https://avatars.githubusercontent.com/u/1?v=4", HTMLURL: "https://github.com/blackfyre", Contributions: 1},
	}})

	for _, unsupported := range []string{
		"Emil Krén",
		"Daniel Marx",
		"created by",
		"Version 2",
		"SUPPORT THE PROJECT",
		"SPONSOR ON GITHUB",
		"github.com/sponsors",
		"REPOSITORY",
		"LICENCE",
		"STACK",
	} {
		if strings.Contains(rendered, unsupported) {
			t.Errorf("rendered contributors page must not contain unsupported editorial %q\ngot: %s", unsupported, rendered)
		}
	}
}

func TestContributorOmitsMissingAvatarAndProfileLink(t *testing.T) {
	rendered := renderContributors(t, ContributorsPageDTO{Contributors: []GithubContributor{
		{Login: "anon", Contributions: 3},
	}})

	if strings.Contains(rendered, "<img") {
		t.Error("contributor without an avatar must not render an image")
	}
	if strings.Contains(rendered, `aria-label="@anon on GitHub"`) {
		t.Error("contributor without a profile URL must not render a profile anchor")
	}
	if strings.Contains(rendered, `src=""`) || strings.Contains(rendered, `href=""`) {
		t.Error("contributor output must not emit empty src or href attributes")
	}
	if !strings.Contains(rendered, "@anon") || !strings.Contains(rendered, "3 commits") {
		t.Error("contributor identity and count must still render")
	}
}

func TestContributorOutputIsSanitisedWithoutUncheckedSafeURL(t *testing.T) {
	rendered := renderContributors(t, ContributorsPageDTO{Contributors: []GithubContributor{
		{Login: "blackfyre", AvatarURL: "https://avatars.githubusercontent.com/u/1?v=4", HTMLURL: "https://github.com/blackfyre", Contributions: 1},
	}})

	if strings.Contains(rendered, "SafeURL") {
		t.Fatal("contributor output must not leak unchecked SafeURL markup")
	}
	if !strings.Contains(rendered, `aria-label="@blackfyre on GitHub"`) || !strings.Contains(rendered, `rel="noopener"`) || !strings.Contains(rendered, `target="_blank"`) {
		t.Error("contributor profile link must carry an accessible name and safe external link attributes")
	}
}
