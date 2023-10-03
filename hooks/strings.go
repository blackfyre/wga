package hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerStringsUpdate(app *pocketbase.PocketBase) {
	app.OnModelBeforeUpdate("strings").Add(func(e *core.ModelEvent) error {

		// record, _ := e.Model.(*models.Record)
		// content := record.Get("content").(string)

		// content = content + "<p>!!!</p>"
		// record.Set("content", content)

		return nil
	})
}
