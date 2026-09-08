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
