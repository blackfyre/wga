package handlers

import (
	"net/http"

	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// pocketBaseSkipSuccessActivityLogKey is the request-event key that
// apis.SkipSuccessActivityLog sets. PocketBase reads it after the handler chain
// returns; TestFailureOnlyActivityLogPersistsOnlyFailedRequests fails if a
// PocketBase upgrade changes it.
const pocketBaseSkipSuccessActivityLogKey = "__skipSuccessActivityLogger"

// registerFailureOnlyActivityLog keeps PocketBase's request activity log for
// failed requests only. A request is skipped when it returns no error and its
// response is below 500 or an expected protection outcome, so a flood of
// successful or admission-rejected requests no longer costs one persisted row
// each, while server faults rendered without an error are still recorded. The
// handler wraps the whole chain, including protection middleware that
// short-circuits it.
func registerFailureOnlyActivityLog(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Bind(&hook.Handler[*core.RequestEvent]{
			Id:       "wgaFailureOnlyActivityLog",
			Priority: apis.DefaultActivityLoggerMiddlewarePriority + 1,
			Func: func(e *core.RequestEvent) error {
				err := e.Next()
				if err == nil && (e.Status() < http.StatusInternalServerError || requestfailure.IsExpectedResponse(e)) {
					e.Set(pocketBaseSkipSuccessActivityLogKey, true)
				}
				return err
			},
		})
		return se.Next()
	})
}
