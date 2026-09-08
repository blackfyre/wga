package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
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
				NotExpectedContent: []string{
					"<!DOCTYPE", "<html", "protected handler invoked", protectionTestSecret,
					"CF-Connecting-IP", "X-WGA-Edge-Secret", "198.51.100",
				},
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

func TestPlainProtectionResponsesAreMinimalExpectedOutcomes(t *testing.T) {
	for _, test := range []struct {
		status         int
		retryAfter     time.Duration
		wantRetryAfter string
	}{
		{status: http.StatusForbidden},
		{status: http.StatusMisdirectedRequest},
		{status: http.StatusTooManyRequests, retryAfter: 1500 * time.Millisecond, wantRetryAfter: "2"},
		{status: http.StatusServiceUnavailable, retryAfter: 1500 * time.Millisecond, wantRetryAfter: "2"},
	} {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			response := httptest.NewRecorder()
			event := &core.RequestEvent{Event: router.Event{
				Request:  httptest.NewRequest(http.MethodGet, "/artists/private-slug", nil),
				Response: response,
			}}

			if err := plainProtectionResponse(event, test.status, test.retryAfter); err != nil {
				t.Fatalf("render response: %v", err)
			}
			if response.Code != test.status {
				t.Fatalf("status = %d; want %d", response.Code, test.status)
			}
			if got, want := response.Body.String(), http.StatusText(test.status)+"\n"; got != want {
				t.Fatalf("body = %q; want %q", got, want)
			}
			if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
				t.Fatalf("Content-Type = %q; want plain text", got)
			}
			if got := response.Header().Get("Retry-After"); got != test.wantRetryAfter {
				t.Fatalf("Retry-After = %q; want %q", got, test.wantRetryAfter)
			}
			if !requestfailure.IsExpectedResponse(event) {
				t.Fatal("protection response was not marked as an expected outcome")
			}
		})
	}
}

func TestProtectedReadMiddlewareLogsStablePrivateDecisions(t *testing.T) {
	const (
		clientIdentity = "198.51.100.77"
		forwardedIP    = "203.0.113.88"
		privateSlug    = "private-record-slug"
		invalidSecret  = "visitor-origin-secret-value"
	)

	cases := []struct {
		name           string
		mode           config.ProtectionMode
		url            string
		headers        map[string]string
		status         int
		logStatus      int
		event          string
		decision       requestprotection.Decision
		capacityUsed   string
		prepare        func(*requestprotection.Policy) requestprotection.Admission
		wantHandlerRun bool
	}{
		{
			name:     "host rejection",
			mode:     config.ProtectionModeEnforce,
			url:      "https://wga-production.up.railway.app/artists/" + privateSlug,
			headers:  trustedProtectionHeaders(clientIdentity, protectionTestSecret),
			status:   http.StatusMisdirectedRequest,
			event:    "request_protection.host_rejected",
			decision: requestprotection.DecisionHost,
		},
		{
			name:     "origin authentication rejection",
			mode:     config.ProtectionModeEnforce,
			url:      "https://beta.wga.hu/artists/" + privateSlug,
			headers:  trustedProtectionHeaders(clientIdentity, invalidSecret),
			status:   http.StatusForbidden,
			event:    "request_protection.origin_authentication_rejected",
			decision: requestprotection.DecisionOriginAuth,
		},
		{
			name:     "client rate rejection",
			mode:     config.ProtectionModeEnforce,
			url:      "https://beta.wga.hu/artists/" + privateSlug,
			headers:  trustedProtectionHeaders(clientIdentity, protectionTestSecret),
			status:   http.StatusTooManyRequests,
			event:    "request_protection.client_rate_rejected",
			decision: requestprotection.DecisionClientRate,
			prepare: func(policy *requestprotection.Policy) requestprotection.Admission {
				admission := policy.Admit(context.Background(), requestprotection.ProfileDetail, clientIdentity, true)
				admission.Release()
				return requestprotection.Admission{}
			},
		},
		{
			name:         "capacity rejection",
			mode:         config.ProtectionModeEnforce,
			url:          "https://beta.wga.hu/artists/" + privateSlug,
			headers:      trustedProtectionHeaders(clientIdentity, protectionTestSecret),
			status:       http.StatusServiceUnavailable,
			event:        "request_protection.capacity_rejected",
			decision:     requestprotection.DecisionGlobalCapacity,
			capacityUsed: "1",
			prepare: func(policy *requestprotection.Policy) requestprotection.Admission {
				return policy.Admit(context.Background(), requestprotection.ProfileDetail, "198.51.100.99", true)
			},
		},
		{
			name:           "observe rate decision",
			mode:           config.ProtectionModeObserve,
			url:            "https://beta.wga.hu/artists/" + privateSlug,
			headers:        trustedProtectionHeaders(clientIdentity, protectionTestSecret),
			status:         http.StatusOK,
			logStatus:      http.StatusTooManyRequests,
			event:          "request_protection.admission_observed",
			decision:       requestprotection.DecisionClientRate,
			wantHandlerRun: true,
			prepare: func(policy *requestprotection.Policy) requestprotection.Admission {
				admission := policy.Admit(context.Background(), requestprotection.ProfileDetail, clientIdentity, true)
				admission.Release()
				return requestprotection.Admission{}
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			test.headers["X-Forwarded-For"] = forwardedIP
			test.headers["X-Real-IP"] = forwardedIP
			var captured func() []*core.Log
			var held requestprotection.Admission
			var handlerCalls atomic.Int32

			scenario := tests.ApiScenario{
				Name:           test.name,
				Method:         http.MethodGet,
				URL:            test.url,
				Headers:        test.headers,
				ExpectedStatus: test.status,
				ExpectedContent: func() []string {
					if test.wantHandlerRun {
						return []string{"handler completed"}
					}
					return []string{http.StatusText(test.status)}
				}(),
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := testutils.NewTestApp(t)
					captured = testutils.CaptureLogs(app)
					logging.RegisterRequestIDMiddleware(app)
					policy := newHTTPProtectionPolicyMode(t, test.mode)
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
						se.Router.GET("/artists/{name}", func(e *core.RequestEvent) error {
							handlerCalls.Add(1)
							return e.String(http.StatusOK, "handler completed")
						})
						return se.Next()
					})
					return app
				},
				AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
					held.Release()
					testutils.FlushLogs(t, app)
					entries := testutils.LogsWithEvent(captured(), test.event)
					if len(entries) != 1 {
						t.Fatalf("%s logs = %d; want 1", test.event, len(entries))
					}
					data := entries[0].Data
					logStatus := test.logStatus
					if logStatus == 0 {
						logStatus = test.status
					}
					wantFields := map[string]string{
						"profile":          "detail",
						"decision":         string(test.decision),
						"status":           fmt.Sprint(logStatus),
						"configured_limit": "1",
						"capacity_limit":   "1",
						"protection_mode":  string(test.mode),
					}
					for key, want := range wantFields {
						if got := fmt.Sprint(data[key]); got != want {
							t.Errorf("%s = %q; want %q", key, got, want)
						}
					}
					wantCapacity := test.capacityUsed
					if wantCapacity == "" {
						wantCapacity = "0"
					}
					if got := fmt.Sprint(data["capacity_in_use"]); got != wantCapacity {
						t.Errorf("capacity_in_use = %q; want %q", got, wantCapacity)
					}
					if fmt.Sprint(data["request_id"]) == "" {
						t.Error("request_id is empty")
					}
					wantCalls := int32(0)
					if test.wantHandlerRun {
						wantCalls = 1
					}
					if got := handlerCalls.Load(); got != wantCalls {
						t.Errorf("handler calls = %d; want %d", got, wantCalls)
					}

					formatted := fmt.Sprint(data)
					for _, forbiddenKey := range []string{
						"client_identity", "client_ip", "forwarded_headers", "limiter_key", "origin_secret", "requested_slug",
					} {
						if _, found := data[forbiddenKey]; found {
							t.Errorf("structured fields contain forbidden key %q", forbiddenKey)
						}
					}
					for _, sensitive := range []string{
						clientIdentity, forwardedIP, protectionTestSecret, invalidSecret, privateSlug,
						"CF-Connecting-IP", "X-Forwarded-For", "X-Real-IP", "X-WGA-Edge-Secret", "limiter_key",
					} {
						if strings.Contains(formatted, sensitive) {
							t.Errorf("structured fields exposed %q: %s", sensitive, formatted)
						}
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

func TestCancellationTelemetryRecordsOnlyProfileAndStage(t *testing.T) {
	const (
		clientIdentity = "198.51.100.91"
		privateSlug    = "private-record-slug"
		querySecret    = "request-query-secret"
	)
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:           "cancelled protected detail",
		Method:         http.MethodGet,
		URL:            "https://beta.wga.hu/artists/" + privateSlug + "?token=" + querySecret,
		Headers:        trustedProtectionHeaders(clientIdentity, protectionTestSecret),
		ExpectedStatus: http.StatusNoContent,
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := newProtectionContractApp(t)
			captured = testutils.CaptureLogs(app)
			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/artists/{slug}", func(e *core.RequestEvent) error {
					ctx, cancel := context.WithCancel(e.Request.Context())
					e.Request = e.Request.WithContext(ctx)
					cancel()
					_ = requestprotection.Checkpoint(e.Request.Context(), "artist.detail.projection")
					return e.NoContent(http.StatusNoContent)
				})
				return se.Next()
			})
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			testutils.FlushLogs(t, app)
			entries := testutils.LogsWithEvent(captured(), "request_protection.cancelled")
			if len(entries) != 1 {
				t.Fatalf("cancellation logs = %d, want 1", len(entries))
			}
			data := entries[0].Data
			if got := fmt.Sprint(data["profile"]); got != "detail" {
				t.Errorf("profile = %q, want detail", got)
			}
			if got := fmt.Sprint(data["stage"]); got != "artist.detail.projection" {
				t.Errorf("stage = %q, want artist.detail.projection", got)
			}
			if fmt.Sprint(data["request_id"]) == "" {
				t.Error("request_id is empty")
			}
			formatted := fmt.Sprint(data)
			for _, forbiddenKey := range []string{"client_identity", "client_ip", "path", "query", "requested_slug"} {
				if _, found := data[forbiddenKey]; found {
					t.Errorf("structured fields contain forbidden key %q", forbiddenKey)
				}
			}
			for _, sensitive := range []string{clientIdentity, privateSlug, querySecret, protectionTestSecret} {
				if strings.Contains(formatted, sensitive) {
					t.Errorf("structured fields exposed %q: %s", sensitive, formatted)
				}
			}
		},
	}

	scenario.Test(t)
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
	return newHTTPProtectionPolicyMode(t, config.ProtectionModeEnforce)
}

func newHTTPProtectionPolicyMode(t testing.TB, mode config.ProtectionMode) *requestprotection.Policy {
	t.Helper()
	policy, err := requestprotection.NewPolicy(config.PublicRequestProtection{
		Mode:                      mode,
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
