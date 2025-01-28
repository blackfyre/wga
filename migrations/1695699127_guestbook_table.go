package migrations

import (
	"encoding/json"
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type GuestbookRecord struct {
	Message  string `json:"message"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Location string `json:"location"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("Guestbook")

		collection.Name = "Guestbook"
		collection.Id = "guestbook"
		collection.System = false
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:   "guestbooks_message",
				Name: "message",
			},
			&core.TextField{
				Id:   "guestbooks_name",
				Name: "name",
			},
			&core.TextField{
				Id:   "guestbooks_email",
				Name: "email",
			},
			&core.TextField{
				Id:   "guestbooks_location",
				Name: "location",
			},
		)

		err := app.Save(collection)

		if err != nil {
			return err
		}

		data, err := os.ReadFile("./guestbook.json")

		if err != nil {
			// no data to import
			return nil
		} else {
			var c []GuestbookRecord

			err = json.Unmarshal(data, &c)

			if err != nil {
				return err
			}

			for _, g := range c {

				r := core.NewRecord(collection)

				r.Set("message", g.Message)
				r.Set("name", g.Name)
				r.Set("email", g.Email)
				r.Set("location", g.Location)
				r.Set("created", g.Created)
				r.Set("updated", g.Updated)

				err = app.Save(r)

				if err != nil {
					return err
				}

			}

			return nil
		}
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("guestbook")

		if err != nil {
			return err
		}

		return app.Delete(c)
	})
}
