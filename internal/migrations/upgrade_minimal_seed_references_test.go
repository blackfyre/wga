package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestUpgradeMinimalSeedReferencesKeepsEditedRecords(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	schoolCollection := core.NewBaseCollection("schools")
	schoolCollection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	if err := app.Save(schoolCollection); err != nil {
		t.Fatalf("create schools collection: %v", err)
	}

	stringsCollection := core.NewBaseCollection("strings")
	stringsCollection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "content"},
	)
	if err := app.Save(stringsCollection); err != nil {
		t.Fatalf("create strings collection: %v", err)
	}

	staticPagesCollection := core.NewBaseCollection("static_pages")
	staticPagesCollection.Fields.Add(
		&core.TextField{Name: "title"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "content"},
	)
	if err := app.Save(staticPagesCollection); err != nil {
		t.Fatalf("create static pages collection: %v", err)
	}

	if err := upgradeMinimalSeedReferences(app); err != nil {
		t.Fatalf("upgrade missing seed records: %v", err)
	}

	school := core.NewRecord(schoolCollection)
	school.Set("name", "Curated School")
	school.Set("slug", minimalSeedSchoolSlug)
	if err := app.Save(school); err != nil {
		t.Fatalf("create edited school: %v", err)
	}

	welcome := core.NewRecord(stringsCollection)
	welcome.Set("name", "welcome")
	welcome.Set("content", "<p>Edited welcome content.</p>")
	if err := app.Save(welcome); err != nil {
		t.Fatalf("create edited welcome content: %v", err)
	}

	privacyPage := core.NewRecord(staticPagesCollection)
	privacyPage.Set("title", "Edited privacy policy")
	privacyPage.Set("slug", "privacy-policy")
	privacyPage.Set("content", minimalSeedPrivacyContent)
	if err := app.Save(privacyPage); err != nil {
		t.Fatalf("create edited privacy page: %v", err)
	}

	if err := upgradeMinimalSeedReferences(app); err != nil {
		t.Fatalf("upgrade edited seed records: %v", err)
	}

	updatedSchool, err := app.FindRecordById("schools", school.Id)
	if err != nil {
		t.Fatalf("find edited school: %v", err)
	}
	updatedWelcome, err := app.FindRecordById("strings", welcome.Id)
	if err != nil {
		t.Fatalf("find edited welcome content: %v", err)
	}
	updatedPrivacyPage, err := app.FindRecordById("static_pages", privacyPage.Id)
	if err != nil {
		t.Fatalf("find edited privacy page: %v", err)
	}

	if got, want := updatedSchool.GetString("name"), "Curated School"; got != want {
		t.Fatalf("expected school name %q, got %q", want, got)
	}
	if got, want := updatedWelcome.GetString("content"), "<p>Edited welcome content.</p>"; got != want {
		t.Fatalf("expected welcome content %q, got %q", want, got)
	}
	if got, want := updatedPrivacyPage.GetString("title"), "Edited privacy policy"; got != want {
		t.Fatalf("expected privacy page title %q, got %q", want, got)
	}
	if got, want := updatedPrivacyPage.GetString("content"), minimalSeedPrivacyContent; got != want {
		t.Fatalf("expected privacy page content %q, got %q", want, got)
	}
}
