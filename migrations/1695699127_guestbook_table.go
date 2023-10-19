package migrations

import (
	"encoding/json"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type GuestbookRecord struct {
	Message          string `json:"message"`
	Name        	 string `json:"name"`
	Email       	 string `json:"email"`
	Location         string `json:"location"`
	Created          string `json:"created"`
	Updated          string `json:"updated"`
}

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "guestbook"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:      "guestbooks_message",
				Name:    "message",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:          "guestbooks_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "guestbooks_email",
				Name:        "email",
				Type:        schema.FieldTypeEmail,
				Options:     &schema.EmailOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "guestbooks_location",
				Name:    "location",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
		)

		err := dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		data, err := os.ReadFile("./guestbook.json")
		if err != nil {
			return dao.SaveCollection(collection)
		} else {
			var c []GuestbookRecord

			err = json.Unmarshal(data, &c)
	
			if err != nil {
				return err
			}
	
			for _, g := range c {
				q := db.Insert("guestbook", dbx.Params{
					"message":      g.Message,
					"name":         g.Name,
					"email":        g.Email,
					"location": 	g.Location,
					"created":      g.Created,
					"updated":      g.Updated,
				})
	
				_, err = q.Execute()
	
				if err != nil {
					return err
				}
	
			}
	
			return nil
		}
	}, func(db dbx.Builder) error {
		// add down queries...

		q := db.DropTable("guestbook")
		_, err := q.Execute()

		return err
	})
}
