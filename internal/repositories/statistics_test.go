package repositories

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestGetArtFormDistributionCountsArtworkOncePerForm(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	defer app.Cleanup()

	forms := core.NewBaseCollection("Art_forms")
	forms.Id = "test_art_forms"
	forms.MarkAsNew()
	forms.Fields.Add(&core.TextField{Id: "test_form_name", Name: "name", Required: true})
	if err := app.Save(forms); err != nil {
		t.Fatalf("save art forms collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "test_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.RelationField{
			Id:           "test_artwork_form",
			Name:         "form",
			CollectionId: forms.Id,
			MinSelect:    1,
			MaxSelect:    20,
		},
		&core.BoolField{Id: "test_artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	form := core.NewRecord(forms)
	form.Set("id", "testformrecord1")
	form.Set("name", "Painting")
	if err := app.Save(form); err != nil {
		t.Fatalf("save art form: %v", err)
	}

	artwork := core.NewRecord(artworks)
	artwork.Set("id", "testartworkrec1")
	artwork.Set("form", []string{form.Id})
	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork: %v", err)
	}

	if _, err := app.DB().NewQuery(`UPDATE Artworks SET form = '["testformrecord1","testformrecord1"]' WHERE id = 'testartworkrec1'`).Execute(); err != nil {
		t.Fatalf("duplicate art form relation: %v", err)
	}

	rows, err := NewStatisticsRepository(app).GetArtFormDistribution()
	if err != nil {
		t.Fatalf("get art form distribution: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one art form row, got %d", len(rows))
	}
	if rows[0].Name != "Painting" || rows[0].Count != 1 {
		t.Errorf("expected Painting count 1, got %#v", rows[0])
	}
}
