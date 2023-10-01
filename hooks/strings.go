package hooks

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerStringsUpdate(app *pocketbase.PocketBase) {
	log.Println("registering strings update hook")
	// fires only for "users" and "members"
	app.OnModelBeforeUpdate("strings").Add(func(e *core.ModelEvent) error {
		fmt.Println(e.Model.TableName())
		log.Println(e.Model.GetId())
		return nil
	})
}
