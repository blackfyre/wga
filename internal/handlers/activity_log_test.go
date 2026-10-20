package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestFailureOnlyActivityLogPersistsOnlyFailedRequests(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		status    int
		unbounded bool
		wantLog   bool
	}{
		{name: "successful request without the middleware", url: "/ok", status: http.StatusOK, unbounded: true, wantLog: true},
		{name: "successful request", url: "/ok", status: http.StatusOK},
		{name: "short-circuited rejection", url: "/rejected", status: http.StatusServiceUnavailable},
		{name: "failed request", url: "/missing", status: http.StatusNotFound, wantLog: true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var captured func() []*core.Log
			scenario := tests.ApiScenario{
				Name:            test.name,
				Method:          http.MethodGet,
				URL:             test.url,
				ExpectedStatus:  test.status,
				ExpectedContent: []string{""},
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := testutils.NewTestApp(t)
					captured = testutils.CaptureLogs(app)
					if !test.unbounded {
						registerFailureOnlyActivityLog(app)
					}
					app.OnServe().BindFunc(func(se *core.ServeEvent) error {
						se.Router.BindFunc(func(e *core.RequestEvent) error {
							if e.Request.URL.Path == "/rejected" {
								return e.String(http.StatusServiceUnavailable, "Service Unavailable\n")
							}
							return e.Next()
						})
						se.Router.GET("/ok", func(e *core.RequestEvent) error {
							return e.String(http.StatusOK, "ok")
						})
						return se.Next()
					})
					return app
				},
				AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
					want := 0
					if test.wantLog {
						want = 1
					}
					// PocketBase records request activity asynchronously, so wait
					// long enough for an expected entry before asserting absence.
					requestLogs := 0
					for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
						testutils.FlushLogs(t, app)
						requestLogs = 0
						for _, entry := range captured() {
							if entry.Data["type"] == "request" {
								requestLogs++
							}
						}
						if requestLogs >= 1 {
							break
						}
					}
					if requestLogs != want {
						t.Fatalf("persisted request logs = %d; want %d", requestLogs, want)
					}
				},
			}
			scenario.Test(t)
		})
	}
}
