package contributors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	contributorworkflow "github.com/blackfyre/wga/internal/contributors"
	"github.com/blackfyre/wga/internal/testutils"
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
			"Unable to load contributors.",
		},
		NotExpectedContent: []string{
			sensitiveDetail,
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
	scenario := tests.ApiScenario{
		Name:            "contributors route renders stored snapshot",
		Method:          http.MethodGet,
		URL:             "/contributors",
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{"@stored-contributor"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			RegisterHandlers(app, readerFunc(func(context.Context) (contributorworkflow.Snapshot, error) {
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
		},
	}

	scenario.Test(t)
}
