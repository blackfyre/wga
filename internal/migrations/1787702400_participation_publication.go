package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const musicSongPublishedIndex = "pbx_music_song_published"

func init() {
	m.Register(addParticipationPublication, removeParticipationPublication)
}

// addParticipationPublication introduces the explicit `published` flag for the
// period-music participation records. Songs and composers that already exist
// when the migration runs are backfilled as published so the release does not
// hide previously visible recordings; records created afterwards inherit the
// boolean field's false default and stay private until a writer publishes them
// explicitly. The only index added is the one the public card query uses to
// select published songs ordered by title and id; composers are reached by
// primary-key lookup and need no published index.
func addParticipationPublication(app core.App) error {
	songs, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		return err
	}
	if songs.Fields.GetByName("published") == nil {
		songs.Fields.Add(&core.BoolField{Name: "published"})
	}
	songs.Indexes = appendIndex(songs.Indexes,
		"CREATE INDEX `"+musicSongPublishedIndex+"` ON `Music_song` (`published`, `title`, `id`)",
	)
	if err := app.Save(songs); err != nil {
		return err
	}

	composers, err := app.FindCollectionByNameOrId("music_composer")
	if err != nil {
		return err
	}
	if composers.Fields.GetByName("published") == nil {
		composers.Fields.Add(&core.BoolField{Name: "published"})
	}
	if err := app.Save(composers); err != nil {
		return err
	}

	for _, collection := range []string{"music_song", "music_composer"} {
		if err := backfillPublished(app, collection); err != nil {
			return err
		}
	}

	return nil
}

// backfillPublished marks every still-private record as published. It targets
// only rows that still carry the field's false default, so the migration-time
// snapshot is published once while re-running the migration is a no-op.
func backfillPublished(app core.App, collection string) error {
	records, err := app.FindRecordsByFilter(collection, "published = false", "", 0, 0)
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set("published", true)
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}

// removeParticipationPublication drops the period-music query index but keeps
// the `published` fields and their values. Publication state is authoritative
// data: a code rollback disables the routes that read it but must not destroy
// the recorded outcome, so a forward-compatible redeploy restores the index and
// keeps the same visibility.
func removeParticipationPublication(app core.App) error {
	songs, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		return err
	}

	songs.Indexes = removeIndex(songs.Indexes, musicSongPublishedIndex)

	return app.Save(songs)
}
