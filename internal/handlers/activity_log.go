package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// pocketBaseSkipSuccessActivityLogKey is the request-event key that
// apis.SkipSuccessActivityLog sets. PocketBase reads it after the handler chain
// returns; TestFailureOnlyActivityLogPersistsOnlyFailedRequests fails if a
// PocketBase upgrade changes it.
const pocketBaseSkipSuccessActivityLogKey = "__skipSuccessActivityLogger"

// protectionRejectionKey marks a response written by public-read admission.
// Unlike the general expected-response marker, it is never set for other
// rendered 5xx failures, which keep their activity row.
const protectionRejectionKey = "wga.protection_rejection"

func isProtectionRejection(e *core.RequestEvent) bool {
	rejected, _ := e.Get(protectionRejectionKey).(bool)
	return rejected
}

// registerFailureOnlyActivityLog keeps PocketBase's request activity log for
// failed requests only. A request is skipped when it returns no error and its
// response is below 500 or an admission rejection, so a flood of
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
				if err == nil && (e.Status() < http.StatusInternalServerError || isProtectionRejection(e)) {
					e.Set(pocketBaseSkipSuccessActivityLogKey, true)
				}
				return err
			},
		})
		return se.Next()
	})
}
