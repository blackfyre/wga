package repositories

import "testing"

func TestArtistSchoolVocabularyIsCompleteAndAlphabetical(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexSchool(t, app, "schoolzulu00001", "zulu", "Zulu")
	saveArtistIndexSchool(t, app, "schoolalpha0001", "alpha", "Alpha")

	schools, err := ListArtistSchools(app)
	if err != nil {
		t.Fatal(err)
	}
	if len(schools) != 2 || schools[0].Name != "Alpha" || schools[1].Name != "Zulu" {
		t.Fatalf("school roster = %#v, want complete alphabetical vocabulary", schools)
	}
}

func TestArtistPeriodVocabularyRequiresPublishedArtistAssociation(t *testing.T) {
	app := newArtistIndexTestApp(t)
	saveArtistIndexPeriod(t, app, "periodzulu00001", "Zulu", 1700, 1799)
	saveArtistIndexPeriod(t, app, "periodalpha0001", "Alpha", 1500, 1599)
	saveArtistIndexPeriod(t, app, "periodunused001", "Unused", 1200, 1299)
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artistpub100000", name: "Public", birth: 1750, published: true})
	saveArtistIndexArtist(t, app, artistIndexArtistSeed{id: "artisthid100000", name: "Hidden", birth: 1550, published: false})

	periods, err := ListArtistPeriods(app)
	if err != nil {
		t.Fatal(err)
	}
	if len(periods) != 1 || periods[0].Name != "Zulu" {
		t.Fatalf("artist periods = %#v, want only the published-artist association", periods)
	}
}

func TestArtistPeriodVocabularyFailsClosedWithoutPublicIdentityFields(t *testing.T) {
	app := newArtistIndexLegacyTestApp(t, false)

	periods, err := ListArtistPeriods(app)
	if err != nil {
		t.Fatal(err)
	}
	if periods != nil {
		t.Fatalf("artist periods = %#v, want nil without public identity fields", periods)
	}
}
