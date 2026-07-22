package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestMinimalSeedDataSkipsExistingApplicationData(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("existing_data")
	collection.Fields.Add(&core.TextField{Name: "value"})
	if err := app.Save(collection); err != nil {
		t.Fatalf("create existing data collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("value", "existing")
	if err := app.Save(record); err != nil {
		t.Fatalf("create existing data record: %v", err)
	}

	if err := seedMinimalSiteData(app); err != nil {
		t.Fatalf("seed minimal site data: %v", err)
	}

	if _, err := app.FindCollectionByNameOrId("artists"); err == nil {
		t.Fatal("expected existing data to prevent seed collection creation")
	}
}

func TestMinimalSeedDataSkipsExistingSuperuser(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("find superusers collection: %v", err)
	}

	superuser := core.NewRecord(superusers)
	superuser.Set("email", "admin@example.com")
	superuser.Set("password", "password123456")
	if err := app.Save(superuser); err != nil {
		t.Fatalf("create superuser: %v", err)
	}

	if err := seedMinimalSiteData(app); err != nil {
		t.Fatalf("seed minimal site data: %v", err)
	}

	if _, err := app.FindCollectionByNameOrId("artists"); err == nil {
		t.Fatal("expected existing superuser to prevent seed collection creation")
	}
}
