package repositories

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestGetRecordStringField(t *testing.T) {
	t.Run("returns string field value", func(t *testing.T) {
		collection := core.NewBaseCollection("strings")
		record := core.NewRecord(collection)
		record.Set("content", "welcome")

		value, err := getRecordStringField(record, "content")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if value != "welcome" {
			t.Fatalf("expected value 'welcome', got %q", value)
		}
	})

	t.Run("returns error for non-string field", func(t *testing.T) {
		collection := core.NewBaseCollection("strings")
		record := core.NewRecord(collection)
		record.Set("content", 123)

		_, err := getRecordStringField(record, "content")
		if err == nil {
			t.Fatalf("expected type error for non-string field")
		}
	})
}
