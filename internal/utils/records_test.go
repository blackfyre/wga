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

	collection := core.NewBaseCollection("Items")
	collection.Id = "items"
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.BoolField{Name: "published"},
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
}
