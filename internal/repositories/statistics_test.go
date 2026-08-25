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

func TestGetArtworksBySchoolAndPeriodExcludesUnpublishedArtists(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	defer app.Cleanup()

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.BoolField{Id: "test_artist_published", Name: "published"},
		&core.NumberField{Id: "test_artist_yob", Name: "year_of_birth"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	schools := core.NewBaseCollection("Schools")
	schools.Id = "test_schools"
	schools.MarkAsNew()
	schools.Fields.Add(&core.TextField{Id: "test_school_name", Name: "name", Required: true})
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "test_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.BoolField{Id: "test_artwork_published", Name: "published"},
		&core.RelationField{Id: "test_artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Id: "test_artwork_school", Name: "school", CollectionId: schools.Id, MinSelect: 1, MaxSelect: 10},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	publishedArtist := core.NewRecord(artists)
	publishedArtist.Set("id", "testartistpub01")
	publishedArtist.Set("published", true)
	publishedArtist.Set("year_of_birth", 1500)
	if err := app.Save(publishedArtist); err != nil {
		t.Fatalf("save published artist: %v", err)
	}

	unpublishedArtist := core.NewRecord(artists)
	unpublishedArtist.Set("id", "testartistunp01")
	unpublishedArtist.Set("published", false)
	unpublishedArtist.Set("year_of_birth", 1500)
	if err := app.Save(unpublishedArtist); err != nil {
		t.Fatalf("save unpublished artist: %v", err)
	}

	school := core.NewRecord(schools)
	school.Set("id", "testschoolit001")
	school.Set("name", "Italian")
	if err := app.Save(school); err != nil {
		t.Fatalf("save school: %v", err)
	}

	publishedWork := core.NewRecord(artworks)
	publishedWork.Set("id", "testworkpub0001")
	publishedWork.Set("published", true)
	publishedWork.Set("author", []string{publishedArtist.Id})
	publishedWork.Set("school", []string{school.Id})
	if err := app.Save(publishedWork); err != nil {
		t.Fatalf("save published artwork: %v", err)
	}

	leakyWork := core.NewRecord(artworks)
	leakyWork.Set("id", "testworkleak001")
	leakyWork.Set("published", true)
	leakyWork.Set("author", []string{unpublishedArtist.Id})
	leakyWork.Set("school", []string{school.Id})
	if err := app.Save(leakyWork); err != nil {
		t.Fatalf("save leaky artwork: %v", err)
	}

	rows, err := NewStatisticsRepository(app).GetArtworksBySchoolAndPeriod()
	if err != nil {
		t.Fatalf("get artworks by school and period: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one school/period row, got %d: %#v", len(rows), rows)
	}
	if rows[0].PeriodStart != 1500 || rows[0].School != "Italian" || rows[0].Count != 1 {
		t.Errorf("expected (1500, Italian, 1), got %#v", rows[0])
	}
}
