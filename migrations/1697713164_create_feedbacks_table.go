package migrations

import (
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {

	tId := "feedbacks"
	tName := "Feedbacks"

	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = tName
		collection.Id = tId
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()

		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          tId + "_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
				Required:    true,
			},
			&schema.SchemaField{
				Id:       tId + "_email",
				Name:     "email",
				Type:     schema.FieldTypeEmail,
				Options:  &schema.EmailOptions{},
				Required: true,
			},
			&schema.SchemaField{
				Id:   tId + "_refer_to",
				Name: "refer_to",
				Type: schema.FieldTypeUrl,
				Options: &schema.UrlOptions{
					OnlyDomains: []string{os.Getenv("WGA_HOSTNAME")},
				},
				Required: true,
			},
			&schema.SchemaField{
				Id:   tId + "_message",
				Name: "message",
				Type: schema.FieldTypeEditor,
				Options: &schema.EditorOptions{
					ConvertUrls: true,
				},
				Required: true,
			},
			&schema.SchemaField{
				Id:          tId + "_handled",
				Name:        "handled",
				Type:        schema.FieldTypeBool,
				Options:     &schema.BoolOptions{},
				Presentable: true,
			},
		)

		return dao.SaveCollection(collection)

	}, func(db dbx.Builder) error {
		q := db.DropTable(tId)
		_, err := q.Execute()

		return err
	})
}
