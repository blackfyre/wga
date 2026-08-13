package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addFeedbackCategories, removeFeedbackCategories)
}

func addFeedbackCategories(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("feedbacks")
	if err != nil {
		return err
	}

	name, ok := collection.Fields.GetByName("name").(*core.TextField)
	if !ok {
		return fmt.Errorf("feedbacks.name is not a text field")
	}
	name.Required = false

	email, ok := collection.Fields.GetByName("email").(*core.EmailField)
	if !ok {
		return fmt.Errorf("feedbacks.email is not an email field")
	}
	email.Required = false

	collection.Fields.Add(selectField("category", []string{"general", "correction", "technical", "suggestion"}, true))

	return app.Save(collection)
}

func removeFeedbackCategories(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("feedbacks")
	if err != nil {
		return err
	}

	collection.Fields.RemoveByName("category")

	name, ok := collection.Fields.GetByName("name").(*core.TextField)
	if !ok {
		return fmt.Errorf("feedbacks.name is not a text field")
	}
	name.Required = true

	email, ok := collection.Fields.GetByName("email").(*core.EmailField)
	if !ok {
		return fmt.Errorf("feedbacks.email is not an email field")
	}
	email.Required = true

	return app.Save(collection)
}
