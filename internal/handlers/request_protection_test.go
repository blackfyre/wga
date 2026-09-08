package handlers

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const protectionTestSecret = "configured-origin-secret"

func TestProtectedReadMiddlewareRejectsBeforeHandlerWork(t *testing.T) {
	cases := []struct {
		name           string
		url            string
		headers        map[string]string
		status         int
		wantRetryAfter string
		prepare        func(*requestprotection.Policy) requestprotection.Admission
	}{
		{
			name:    "deployment host",
			url:     "https://wga-production.up.railway.app/artists",
			headers: trustedProtectionHeaders("198.51.100.7", protectionTestSecret),
			status:  http.StatusMisdirectedRequest,
		},
		{
			name:    "missing origin authentication",
			url:     "https://beta.wga.hu/artists",
			headers: trustedProtectionHeaders("198.51.100.7", ""),
			status:  http.StatusForbidden,
		},
		{
			name:    "invalid origin authentication",
			url:     "https://beta.wga.hu/artists",
			headers: trustedProtectionHeaders("198.51.100.7", "visitor-secret"),
			status:  http.StatusForbidden,
		},
		{
			name:           "client rate exhausted",
			url:            "https://beta.wga.hu/artists",
			headers:        trustedProtectionHeaders("198.51.100.7", protectionTestSecret),
			status:         http.StatusTooManyRequests,
			wantRetryAfter: "2",
			prepare: func(policy *requestprotection.Policy) requestprotection.Admission {
				admission := policy.Admit(context.Background(), requestprotection.ProfileSearch, "198.51.100.7", true)
				admission.Release()
				return requestprotection.Admission{}
			},
		},
		{
			name:           "global capacity exhausted",
			url:            "https://beta.wga.hu/artists",
			headers:        trustedProtectionHeaders("198.51.100.8", protectionTestSecret),
			status:         http.StatusServiceUnavailable,
			wantRetryAfter: "2",
			prepare: func(policy *requestprotection.Policy) requestprotection.Admission {
				return policy.Admit(context.Background(), requestprotection.ProfileSearch, "198.51.100.7", true)
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var handlerCalls atomic.Int32
			var held requestprotection.Admission

			scenario := tests.ApiScenario{
				Name:            test.name,
				Method:          http.MethodGet,
				URL:             test.url,
				Headers:         test.headers,
				ExpectedStatus:  test.status,
				ExpectedContent: []string{http.StatusText(test.status)},
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := testutils.NewTestApp(t)
					logging.RegisterRequestIDMiddleware(app)
					policy := newHTTPProtectionPolicy(t)
					if test.prepare != nil {
						held = test.prepare(policy)
					}
					resolver := requesttrust.New(
						requesttrust.SourceCloudflareRailway,
						requesttrust.NewCloudflareOriginSecrets(protectionTestSecret, ""),
					)
					if err := registerProtectedReadMiddleware(app, "https://beta.wga.hu", resolver, policy); err != nil {
						t.Fatalf("register middleware: %v", err)
					}
					app.OnServe().BindFunc(func(se *core.ServeEvent) error {
						se.Router.GET("/artists", func(e *core.RequestEvent) error {
							handlerCalls.Add(1)
							return e.String(http.StatusOK, "protected handler invoked")
						})
						return se.Next()
					})
					return app
				},
				AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
					held.Release()
					if got := handlerCalls.Load(); got != 0 {
						t.Fatalf("protected handler invoked %d times", got)
					}
					if got := response.Header.Get("Retry-After"); got != test.wantRetryAfter {
						t.Fatalf("Retry-After = %q; want %q", got, test.wantRetryAfter)
					}
				},
			}

			scenario.Test(t)
		})
	}
}

func TestProtectedReadMiddlewareAllowsExemptRoutesThroughDeploymentHost(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		pattern string
		body    string
	}{
		{name: "health", path: "/health", pattern: "/health", body: "healthy"},
		{name: "static asset", path: "/assets/css/style.css", pattern: "/assets/{path...}", body: "asset"},
		{name: "sitemap index", path: "/sitemap.xml", pattern: "/sitemap.xml", body: "sitemap"},
		{name: "sitemap child", path: "/sitemap/artists.xml", pattern: "/sitemap/{path...}", body: "sitemap child"},
		{name: "robots", path: "/robots.txt", pattern: "/robots.txt", body: "robots"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			scenario := tests.ApiScenario{
				Name:            test.name,
				Method:          http.MethodGet,
				URL:             "https://wga-production.up.railway.app" + test.path,
				Headers:         map[string]string{"X-Railway-Edge": "edge-a", "X-Real-IP": "198.51.100.7"},
				ExpectedStatus:  http.StatusOK,
				ExpectedContent: []string{test.body},
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := newProtectionContractApp(t)
					app.OnServe().BindFunc(func(se *core.ServeEvent) error {
						se.Router.GET(test.pattern, func(e *core.RequestEvent) error {
							return e.String(http.StatusOK, test.body)
						})
						return se.Next()
					})
					return app
				},
			}

			scenario.Test(t)
		})
	}
}

func TestProtectedReadMiddlewarePreservesFullPageAndHTMXContracts(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		htmx      bool
		want      []string
		doNotWant []string
	}{
		{
			name:      "canonical full page",
			path:      "/artists",
			want:      []string{"<html", "catalogue page"},
			doNotWant: []string{`id="dual-area"`},
		},
		{
			name:      "canonical HTMX fragment",
			path:      "/dual-mode",
			htmx:      true,
			want:      []string{`id="dual-area"`, "catalogue fragment"},
			doNotWant: []string{"<html"},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			headers := trustedProtectionHeaders("198.51.100.7", protectionTestSecret)
			if test.htmx {
				headers["HX-Request"] = "true"
			}

			scenario := tests.ApiScenario{
				Name:               test.name,
				Method:             http.MethodGet,
				URL:                "https://beta.wga.hu" + test.path,
				Headers:            headers,
				ExpectedStatus:     http.StatusOK,
				ExpectedContent:    test.want,
				NotExpectedContent: test.doNotWant,
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := newProtectionContractApp(t)
					app.OnServe().BindFunc(func(se *core.ServeEvent) error {
						se.Router.GET(test.path, func(e *core.RequestEvent) error {
							if e.Request.Header.Get("HX-Request") == "true" {
								return e.HTML(http.StatusOK, `<section id="dual-area">catalogue fragment</section>`)
							}
							return e.HTML(http.StatusOK, `<html><body>catalogue page</body></html>`)
						})
						return se.Next()
					})
					return app
				},
			}

			scenario.Test(t)
		})
	}
}

func trustedProtectionHeaders(identity string, secret string) map[string]string {
	headers := map[string]string{
		"X-Railway-Edge":   "edge-a",
		"CF-Connecting-IP": identity,
	}
	if secret != "" {
		headers["X-WGA-Edge-Secret"] = secret
	}
	return headers
}

func newHTTPProtectionPolicy(t testing.TB) *requestprotection.Policy {
	t.Helper()
	policy, err := requestprotection.NewPolicy(config.PublicRequestProtection{
		Mode:                      config.ProtectionModeEnforce,
		MaxConcurrentReads:        1,
		SearchRequestsPerMinute:   1,
		FragmentRequestsPerMinute: 1,
		DetailRequestsPerMinute:   1,
		LimiterEntryCapacity:      8,
		RetryAfter:                1500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	return policy
}

func newProtectionContractApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app := testutils.NewTestApp(t)
	logging.RegisterRequestIDMiddleware(app)
	resolver := requesttrust.New(
		requesttrust.SourceCloudflareRailway,
		requesttrust.NewCloudflareOriginSecrets(protectionTestSecret, ""),
	)
	if err := registerProtectedReadMiddleware(app, "https://beta.wga.hu", resolver, newHTTPProtectionPolicy(t)); err != nil {
		t.Fatalf("register middleware: %v", err)
	}
	return app
}
