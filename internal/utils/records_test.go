package utils

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestCountRecordsByFilter(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	authors := core.NewBaseCollection("Authors")
	authors.Id = "authors"
	authors.MarkAsNew()
	authors.Fields.Add(&core.TextField{Name: "name", Required: true})
	if err := app.Save(authors); err != nil {
		t.Fatalf("save authors collection: %v", err)
	}

	collection := core.NewBaseCollection("Items")
	collection.Id = "items"
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.BoolField{Name: "published"},
		&core.RelationField{Name: "authors", CollectionId: authors.Id, MaxSelect: 10},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save collection: %v", err)
	}
	for _, item := range []struct {
		title     string
		published bool
	}{
		{title: "First", published: true},
		{title: "Second", published: true},
		{title: "Draft", published: false},
	} {
		record := core.NewRecord(collection)
		record.Set("title", item.title)
		record.Set("published", item.published)
		if err := app.Save(record); err != nil {
			t.Fatalf("save record: %v", err)
		}
	}

	count, err := CountRecordsByFilter(app, "items", "published = {:published}", dbx.Params{"published": true})
	if err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 published records, got %d", count)
	}

	for range 2 {
		author := core.NewRecord(authors)
		author.Set("name", "Shared")
		if err := app.Save(author); err != nil {
			t.Fatalf("save author: %v", err)
		}
	}
	sharedAuthors, err := app.FindRecordsByFilter("authors", `name = 'Shared'`, "", 0, 0)
	if err != nil {
		t.Fatalf("find shared authors: %v", err)
	}
	collaborative := core.NewRecord(collection)
	collaborative.Set("title", "Collaborative")
	collaborative.Set("published", true)
	collaborative.Set("authors", []string{sharedAuthors[0].Id, sharedAuthors[1].Id})
	if err := app.Save(collaborative); err != nil {
		t.Fatalf("save collaborative record: %v", err)
	}

	count, err = CountRecordsByFilter(app, "items", "authors.name = {:name}", dbx.Params{"name": "Shared"})
	if err != nil {
		t.Fatalf("count records by relation: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 collaborative record, got %d", count)
	}
}
