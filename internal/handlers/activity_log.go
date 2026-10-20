package handlers

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// registerFailureOnlyActivityLog keeps PocketBase's request activity log for
// requests that fail with an error. Successful and admission-rejected requests
// return no error, so a flood of them no longer costs one persisted log row
// each. The handler runs before the activity logger reads the flag and before
// protection middleware can short-circuit the chain.
func registerFailureOnlyActivityLog(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		skip := apis.SkipSuccessActivityLog()
		skip.Priority = apis.DefaultActivityLoggerMiddlewarePriority + 1
		se.Router.Bind(skip)
		return se.Next()
	})
}
