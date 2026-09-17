package hooks

import (
	"testing"

	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestDualModeReferenceHooksInvalidateAfterMutations(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	schools := core.NewBaseCollection("schools")
	schools.Id = "dual_ref_schools"
	schools.MarkAsNew()
	schools.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug", Required: true},
	)
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools collection: %v", err)
	}

	periods := core.NewBaseCollection("art_periods")
	periods.Id = "dual_ref_periods"
	periods.MarkAsNew()
	periods.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.NumberField{Name: "start", Required: true},
		&core.NumberField{Name: "end", Required: true},
	)
	if err := app.Save(periods); err != nil {
		t.Fatalf("save periods collection: %v", err)
	}

	artists := core.NewBaseCollection("artists")
	artists.Id = "dual_ref_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "filing_name", Required: true},
		&core.TextField{Name: "short_name", Required: true},
		&core.NumberField{Name: "year_of_birth"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	collectionHoldingsCacheHook(app)
	dualModeReferenceCacheHook(app)

	school := core.NewRecord(schools)
	school.Id = "dualschool00001"
	school.Set("name", "Original School")
	school.Set("slug", "original-school")
	if err := app.Save(school); err != nil {
		t.Fatalf("create school: %v", err)
	}
	assertDualReference(t, app, "Original School", "", 0)
	school.Set("name", "Renamed School")
	if err := app.Save(school); err != nil {
		t.Fatalf("update school: %v", err)
	}
	assertDualReference(t, app, "Renamed School", "", 0)
	if err := app.Delete(school); err != nil {
		t.Fatalf("delete school: %v", err)
	}
	assertDualReference(t, app, "", "", 0)

	artist := core.NewRecord(artists)
	artist.Id = "dualartist00001"
	artist.Set("name", "Reference Artist")
	artist.Set("filing_name", "Reference, Artist")
	artist.Set("short_name", "Reference")
	artist.Set("year_of_birth", 1500)
	artist.Set("published", true)
	if err := app.Save(artist); err != nil {
		t.Fatalf("create artist: %v", err)
	}
	assertDualReference(t, app, "", "", 1500)

	period := core.NewRecord(periods)
	period.Id = "dualperiod00001"
	period.Set("name", "Original Period")
	period.Set("start", 1400)
	period.Set("end", 1600)
	if err := app.Save(period); err != nil {
		t.Fatalf("create period: %v", err)
	}
	assertDualReference(t, app, "", "Original Period", 1500)
	period.Set("name", "Renamed Period")
	if err := app.Save(period); err != nil {
		t.Fatalf("update period: %v", err)
	}
	assertDualReference(t, app, "", "Renamed Period", 1500)
	if err := app.Delete(period); err != nil {
		t.Fatalf("delete period: %v", err)
	}
	assertDualReference(t, app, "", "", 1500)

	artist.Set("year_of_birth", 1600)
	if err := app.Save(artist); err != nil {
		t.Fatalf("update artist: %v", err)
	}
	assertDualReference(t, app, "", "", 1600)
	if err := app.Delete(artist); err != nil {
		t.Fatalf("delete artist: %v", err)
	}
	assertDualReference(t, app, "", "", 0)
}

func assertDualReference(t *testing.T, app core.App, schoolName, periodName string, bornMin int) {
	t.Helper()
	value, err := repositories.NewDualModeReferenceRepository(app).Load()
	if err != nil {
		t.Fatalf("load dual reference: %v", err)
	}
	if got := firstDualSchoolName(value); got != schoolName {
		t.Fatalf("school name = %q, want %q", got, schoolName)
	}
	if got := firstDualPeriodName(value); got != periodName {
		t.Fatalf("period name = %q, want %q", got, periodName)
	}
	if value.BornMin != bornMin || value.BornMax != bornMin {
		t.Fatalf("birth bounds = (%d, %d), want (%d, %d)", value.BornMin, value.BornMax, bornMin, bornMin)
	}
}

func firstDualSchoolName(value repositories.DualModeReference) string {
	if len(value.Schools) == 0 {
		return ""
	}
	return value.Schools[0].Name
}

func firstDualPeriodName(value repositories.DualModeReference) string {
	if len(value.Periods) == 0 {
		return ""
	}
	return value.Periods[0].Name
}
