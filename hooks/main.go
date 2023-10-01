package hooks

import "github.com/pocketbase/pocketbase"

func RegisterHooks(app *pocketbase.PocketBase) {
	registerStringsUpdate(app)
}
