package contributors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	contributorworkflow "github.com/blackfyre/wga/internal/contributors"
	"github.com/blackfyre/wga/internal/testutils"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type readerFunc func(context.Context) (contributorworkflow.Snapshot, error)

func (f readerFunc) Current(ctx context.Context) (contributorworkflow.Snapshot, error) {
	return f(ctx)
}

func TestContributorServerErrorIsClientSafe(t *testing.T) {
	const sensitiveDetail = "upstream credential token=secret-value"
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:           "contributor failure omits internal detail",
		Method:         http.MethodGet,
		URL:            "/contributors-error",
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedContent: []string{
			"The archive could not complete that request.",
			"Please try again shortly",
		},
		NotExpectedContent: []string{
			sensitiveDetail,
			"Unable to load contributors.",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			app.Settings().Logs.MaxDays = 1
			captured = testutils.CaptureLogs(app)

			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/contributors-error", func(e *core.RequestEvent) error {
					return contributorServerError(app, e, "fetch_error", errors.New(sensitiveDetail))
				})

				return se.Next()
			})

			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			testutils.FlushLogs(t, app)
			entry := testutils.LogWithEvent(captured(), "contributors.request.failed")
			if entry == nil {
				t.Fatal("expected a contributor failure log")
			}
			if strings.Contains(fmt.Sprint(testutils.LogData(captured())), sensitiveDetail) {
				t.Fatalf("captured log contains %q", sensitiveDetail)
			}
		},
	}

	scenario.Test(t)
}

func TestContributorRouteReadsOnlyFromItsReader(t *testing.T) {
	var readerCalls int

	scenario := tests.ApiScenario{
		Name:            "contributors route renders stored snapshot",
		Method:          http.MethodGet,
		URL:             "/contributors",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{"stored-contributor"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			RegisterHandlers(app, readerFunc(func(context.Context) (contributorworkflow.Snapshot, error) {
				readerCalls++
				return contributorworkflow.Snapshot{
					Contributors: []contributorworkflow.Contributor{{Login: "stored-contributor", Contributions: 1}},
					Source:       contributorworkflow.SnapshotSourceCache,
				}, nil
			}))
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
			if got := response.Header.Get("X-WGA-Contributors-Source"); got != "cache" {
				t.Fatalf("source header = %q, want cache", got)
			}
			if readerCalls != 1 {
				t.Fatalf("reader called %d times, want 1", readerCalls)
			}
		},
	}

	scenario.Test(t)
}

func TestContributorRouteServesFallbackSourceHeaderAndObservability(t *testing.T) {
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:            "contributors fallback source header and observability",
		Method:          http.MethodGet,
		URL:             "/contributors",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{"fallback-contributor"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			app.Settings().Logs.MaxDays = 1
			captured = testutils.CaptureLogs(app)
			RegisterHandlers(app, readerFunc(func(context.Context) (contributorworkflow.Snapshot, error) {
				return contributorworkflow.Snapshot{
					Contributors: []contributorworkflow.Contributor{{Login: "fallback-contributor", Contributions: 4}},
					Source:       contributorworkflow.SnapshotSourceFileFallback,
				}, nil
			}))
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, response *http.Response) {
			if got := response.Header.Get("X-WGA-Contributors-Source"); got != "file_fallback" {
				t.Fatalf("source header = %q, want file_fallback", got)
			}
			testutils.FlushLogs(t, app)
			entry := testutils.LogWithEvent(captured(), "contributors.request.completed")
			if entry == nil {
				t.Fatal("expected a contributor fallback log")
			}
			if entry.Data["outcome"] != "fallback" || entry.Data["source"] != contributorworkflow.SnapshotSourceFileFallback {
				t.Fatalf("fallback log data = %#v, want outcome=fallback source=file_fallback", entry.Data)
			}
		},
	}

	scenario.Test(t)
}

func TestContributorRouteRendersExactValuesAndCounts(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "contributors route renders exact values and counts",
		Method:         http.MethodGet,
		URL:            "/contributors",
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			">alice</a>", "7 commits",
			">bob</a>", "12 commits",
			"2 PEOPLE",
			`src="https://avatars.githubusercontent.com/u/1?v=4"`,
			`href="https://github.com/alice"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			RegisterHandlers(app, readerFunc(func(context.Context) (contributorworkflow.Snapshot, error) {
				return contributorworkflow.Snapshot{
					Contributors: []contributorworkflow.Contributor{
						{Login: "alice", AvatarURL: "https://avatars.githubusercontent.com/u/1?v=4", HTMLURL: "https://github.com/alice", Contributions: 7},
						{Login: "bob", AvatarURL: "", HTMLURL: "https://github.com/bob", Contributions: 12},
					},
					Source: contributorworkflow.SnapshotSourceCache,
				}, nil
			}))
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
			if got := response.Header.Get("HX-Push-Url"); got != "/contributors" {
				t.Fatalf("HX-Push-Url = %q, want /contributors", got)
			}
		},
	}

	scenario.Test(t)
}

func TestContributorRouteOmitsInvalidAndEmptyAvatarAndProfileURLs(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "contributors route omits invalid and empty avatar and profile URLs",
		Method:         http.MethodGet,
		URL:            "/contributors",
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			">valid</a>",
			`src="https://avatars.githubusercontent.com/u/9?v=4"`,
			`href="https://github.com/valid"`,
		},
		NotExpectedContent: []string{
			`javascript:alert(1)`,
			`src="//cdn.example.com/avatar.png"`,
			`href="mailto:invalid@example.com"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			RegisterHandlers(app, readerFunc(func(context.Context) (contributorworkflow.Snapshot, error) {
				return contributorworkflow.Snapshot{
					Contributors: []contributorworkflow.Contributor{
						{Login: "valid", AvatarURL: "https://avatars.githubusercontent.com/u/9?v=4", HTMLURL: "https://github.com/valid", Contributions: 2},
						{Login: "relative", AvatarURL: "/avatar.png", HTMLURL: "/profile", Contributions: 1},
						{Login: "js", AvatarURL: "javascript:alert(1)", HTMLURL: "javascript:alert(1)", Contributions: 1},
						{Login: "proto", AvatarURL: "//cdn.example.com/avatar.png", HTMLURL: "//github.com/proto", Contributions: 1},
						{Login: "mail", AvatarURL: "mailto:mail@example.com", HTMLURL: "mailto:invalid@example.com", Contributions: 1},
					},
					Source: contributorworkflow.SnapshotSourceCache,
				}, nil
			}))
			return app
		},
	}

	scenario.Test(t)
}

func TestContributorRoutePreservesCanonicalMetadata(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "contributors route canonical metadata",
		Method:         http.MethodGet,
		URL:            "/contributors",
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`<link rel="canonical" href="https://gallery.example/contributors"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			configureContributorPublicURL(t)
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			RegisterHandlers(app, readerFunc(func(context.Context) (contributorworkflow.Snapshot, error) {
				return contributorworkflow.Snapshot{
					Contributors: []contributorworkflow.Contributor{{Login: "canonical", Contributions: 1}},
					Source:       contributorworkflow.SnapshotSourceCache,
				}, nil
			}))
			return app
		},
	}

	scenario.Test(t)
}

func TestPageContributorsValidatesAbsoluteHTTPURLs(t *testing.T) {
	cases := []struct {
		name       string
		avatarURL  string
		htmlURL    string
		wantAvatar string
		wantHTML   string
	}{
		{
			name:       "valid https avatar and profile",
			avatarURL:  "https://avatars.githubusercontent.com/u/1?v=4",
			htmlURL:    "https://github.com/alice",
			wantAvatar: "https://avatars.githubusercontent.com/u/1?v=4",
			wantHTML:   "https://github.com/alice",
		},
		{
			name:       "valid http",
			avatarURL:  "http://example.com/a.png",
			htmlURL:    "http://example.com/u",
			wantAvatar: "http://example.com/a.png",
			wantHTML:   "http://example.com/u",
		},
		{
			name:       "empty values",
			avatarURL:  "",
			htmlURL:    "",
			wantAvatar: "",
			wantHTML:   "",
		},
		{
			name:       "relative paths",
			avatarURL:  "/avatars/1",
			htmlURL:    "/profile",
			wantAvatar: "",
			wantHTML:   "",
		},
		{
			name:       "protocol relative",
			avatarURL:  "//example.com/a.png",
			htmlURL:    "//example.com/u",
			wantAvatar: "",
			wantHTML:   "",
		},
		{
			name:       "non-http schemes",
			avatarURL:  "javascript:alert(1)",
			htmlURL:    "mailto:alice@example.com",
			wantAvatar: "",
			wantHTML:   "",
		},
		{
			name:       "scheme without host",
			avatarURL:  "https://",
			htmlURL:    "http:",
			wantAvatar: "",
			wantHTML:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pageContributors([]contributorworkflow.Contributor{
				{Login: "alice", AvatarURL: tc.avatarURL, HTMLURL: tc.htmlURL, Contributions: 3},
			})

			if len(got) != 1 {
				t.Fatalf("pageContributors returned %d entries, want 1", len(got))
			}
			if got[0].AvatarURL != tc.wantAvatar {
				t.Errorf("AvatarURL = %q, want %q", got[0].AvatarURL, tc.wantAvatar)
			}
			if got[0].HTMLURL != tc.wantHTML {
				t.Errorf("HTMLURL = %q, want %q", got[0].HTMLURL, tc.wantHTML)
			}
			if got[0].Login != "alice" || got[0].Contributions != 3 {
				t.Errorf("login and contribution count must be preserved exactly: %#v", got[0])
			}
		})
	}
}

func configureContributorPublicURL(t testing.TB) {
	t.Helper()

	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{
			"WGA_ENV":                "development",
			"WGA_PROTOCOL":           "https",
			"WGA_HOSTNAME":           "gallery.example",
			"WGA_SENDER_NAME":        "WGA",
			"WGA_SENDER_ADDRESS":     "sender@example.com",
			"WGA_POSTCARD_FREQUENCY": "*/1 * * * *",
		}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() { apputils.ConfigurePublicURL(config.PublicURL{}) })
}
