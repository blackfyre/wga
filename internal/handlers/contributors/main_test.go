package contributors

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
)

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
			captured = captureContributorLogs(app)

			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/contributors-error", func(e *core.RequestEvent) error {
					return contributorServerError(app, e, "fetch_error", errors.New(sensitiveDetail))
				})

				return se.Next()
			})

			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			flushContributorLogs(t, app)
			entry := contributorLogWithEvent(captured(), "contributors.request.failed")
			if entry == nil {
				t.Fatal("expected a contributor failure log")
			}
			if strings.Contains(fmt.Sprint(contributorLogData(captured())), sensitiveDetail) {
				t.Fatalf("captured log contains %q", sensitiveDetail)
			}
		},
	}

	scenario.Test(t)
}

func captureContributorLogs(app *tests.TestApp) func() []*core.Log {
	var captured []*core.Log
	app.OnModelCreate(core.LogsTableName).BindFunc(func(e *core.ModelEvent) error {
		log, ok := e.Model.(*core.Log)
		if ok {
			entry := *log
			entry.Data = maps.Clone(log.Data)
			captured = append(captured, &entry)
		}

		return e.Next()
	})

	return func() []*core.Log {
		return captured
	}
}

func flushContributorLogs(t testing.TB, app *tests.TestApp) {
	t.Helper()

	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	if err := handler.WriteAll(context.Background()); err != nil {
		t.Fatalf("write logs: %v", err)
	}
}

func contributorLogWithEvent(logs []*core.Log, event string) *core.Log {
	for _, entry := range logs {
		if entry.Data["event"] == event {
			return entry
		}
	}

	return nil
}

func contributorLogData(logs []*core.Log) []any {
	data := make([]any, len(logs))
	for index, entry := range logs {
		data[index] = entry.Data
	}

	return data
}
