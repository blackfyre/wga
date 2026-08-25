package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestParticipationPublicationMigrationInFreshChain(t *testing.T) {
	configureMigrations(t)
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	songs, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find music_song: %v", err)
	}
	if _, ok := songs.Fields.GetByName("published").(*core.BoolField); !ok {
		t.Fatal("music_song.published must be a bool field")
	}
	if !hasIndex(songs.Indexes, musicSongPublishedIndex) {
		t.Fatalf("missing index %s", musicSongPublishedIndex)
	}

	composers, err := app.FindCollectionByNameOrId("music_composer")
	if err != nil {
		t.Fatalf("find music_composer: %v", err)
	}
	if _, ok := composers.Fields.GetByName("published").(*core.BoolField); !ok {
		t.Fatal("music_composer.published must be a bool field")
	}
}

func TestParticipationPublicationBackfillsExistingAndDefaultsNew(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	composers, songs := newParticipationCollections(t, app)

	legacyComposer := core.NewRecord(composers)
	legacyComposer.Set("name", "Sweelinck")
	if err := app.Save(legacyComposer); err != nil {
		t.Fatalf("save legacy composer: %v", err)
	}
	legacySong := core.NewRecord(songs)
	legacySong.Set("title", "Fantasia")
	legacySong.Set("composer", []string{legacyComposer.Id})
	legacySong.Set("source", "fantasia.mp3")
	if err := app.Save(legacySong); err != nil {
		t.Fatalf("save legacy song: %v", err)
	}

	if err := addParticipationPublication(app); err != nil {
		t.Fatalf("add participation publication: %v", err)
	}

	songsAfter, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find updated music_song: %v", err)
	}
	if _, ok := songsAfter.Fields.GetByName("published").(*core.BoolField); !ok {
		t.Fatal("music_song.published must be a bool field")
	}
	if !hasIndex(songsAfter.Indexes, musicSongPublishedIndex) {
		t.Fatalf("missing index %s", musicSongPublishedIndex)
	}

	composersAfter, err := app.FindCollectionByNameOrId("music_composer")
	if err != nil {
		t.Fatalf("find updated music_composer: %v", err)
	}
	if _, ok := composersAfter.Fields.GetByName("published").(*core.BoolField); !ok {
		t.Fatal("music_composer.published must be a bool field")
	}
	if hasIndex(composersAfter.Indexes, "published") {
		t.Fatal("music_composer must not gain an unqueried published index")
	}

	for collection, id := range map[string]string{
		"music_song":     legacySong.Id,
		"music_composer": legacyComposer.Id,
	} {
		record, err := app.FindRecordById(collection, id)
		if err != nil {
			t.Fatalf("find legacy %s record: %v", collection, err)
		}
		if !record.GetBool("published") {
			t.Fatalf("legacy %s record was not backfilled as published", collection)
		}
	}

	privateSong := core.NewRecord(songsAfter)
	privateSong.Set("title", "Unpublished study")
	privateSong.Set("composer", []string{legacyComposer.Id})
	privateSong.Set("source", "study.mp3")
	if err := app.Save(privateSong); err != nil {
		t.Fatalf("save new private song: %v", err)
	}
	if privateSong.GetBool("published") {
		t.Fatal("new song must default to unpublished")
	}
	privateComposer := core.NewRecord(composersAfter)
	privateComposer.Set("name", "Private composer")
	if err := app.Save(privateComposer); err != nil {
		t.Fatalf("save new private composer: %v", err)
	}
	if privateComposer.GetBool("published") {
		t.Fatal("new composer must default to unpublished")
	}

	if err := addParticipationPublication(app); err != nil {
		t.Fatalf("repeat participation publication: %v", err)
	}
	repeated, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find repeated music_song: %v", err)
	}
	if count := strings.Count(strings.Join(repeated.Indexes, "\n"), musicSongPublishedIndex); count != 1 {
		t.Fatalf("repeat migration duplicated index %s (%d occurrences)", musicSongPublishedIndex, count)
	}
	backfilled, err := app.FindRecordById("music_song", legacySong.Id)
	if err != nil {
		t.Fatalf("find backfilled song after repeat: %v", err)
	}
	if !backfilled.GetBool("published") {
		t.Fatal("repeat migration lost the legacy published state")
	}
}

func TestParticipationPublicationRollbackRetainsFieldsAndData(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	composers, songs := newParticipationCollections(t, app)

	composer := core.NewRecord(composers)
	composer.Set("name", "Sweelinck")
	if err := app.Save(composer); err != nil {
		t.Fatalf("save composer: %v", err)
	}
	song := core.NewRecord(songs)
	song.Set("title", "Fantasia")
	song.Set("composer", []string{composer.Id})
	song.Set("source", "fantasia.mp3")
	if err := app.Save(song); err != nil {
		t.Fatalf("save song: %v", err)
	}

	if err := addParticipationPublication(app); err != nil {
		t.Fatalf("add participation publication: %v", err)
	}
	if err := removeParticipationPublication(app); err != nil {
		t.Fatalf("remove participation publication: %v", err)
	}

	for _, collection := range []string{"music_song", "music_composer"} {
		rolledBack, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatalf("find rolled-back %s: %v", collection, err)
		}
		if rolledBack.Fields.GetByName("published") == nil {
			t.Fatalf("authoritative field %s.published was removed by rollback", collection)
		}
	}
	songsAfter, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		t.Fatalf("find rolled-back music_song: %v", err)
	}
	if hasIndex(songsAfter.Indexes, musicSongPublishedIndex) {
		t.Fatalf("rollback retained feature index %s", musicSongPublishedIndex)
	}
	record, err := app.FindRecordById("music_song", song.Id)
	if err != nil {
		t.Fatalf("find rolled-back song record: %v", err)
	}
	if !record.GetBool("published") {
		t.Fatal("rollback lost the authoritative published state")
	}
}

func newParticipationCollections(t *testing.T, app *core.BaseApp) (*core.Collection, *core.Collection) {
	t.Helper()

	composers := core.NewBaseCollection("Music_composer")
	composers.Id = "music_composer"
	composers.MarkAsNew()
	composers.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.SelectField{Name: "century", Values: []string{"17"}, MaxSelect: 1},
	)
	if err := app.Save(composers); err != nil {
		t.Fatalf("save music_composer collection: %v", err)
	}

	songs := core.NewBaseCollection("Music_song")
	songs.Id = "music_song"
	songs.MarkAsNew()
	songs.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "composer", CollectionId: composers.Id, MinSelect: 1, MaxSelect: 20},
		&core.TextField{Name: "source"},
	)
	if err := app.Save(songs); err != nil {
		t.Fatalf("save music_song collection: %v", err)
	}

	return composers, songs
}
