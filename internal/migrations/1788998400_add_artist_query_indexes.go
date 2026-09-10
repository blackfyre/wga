package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	artistIndexPublishedFiling     = "pbx_artist_published_filing"
	artistIndexPublishedFilingDesc = "pbx_artist_published_filing_desc"
	artistIndexPublishedBirth      = "pbx_artist_published_birth"
)

var artistQueryIndexes = []string{
	"CREATE INDEX `" + artistIndexPublishedFiling + "` ON `Artists` (`published`, `filing_name`, `id`)",
	"CREATE INDEX `" + artistIndexPublishedFilingDesc + "` ON `Artists` (`published`, `filing_name` DESC, `id` ASC)",
	"CREATE INDEX `" + artistIndexPublishedBirth + "` ON `Artists` (`published`, (`year_of_birth` = 0), `year_of_birth`, `filing_name`, `id`)",
}

func init() {
	m.Register(addArtistQueryIndexes, removeArtistQueryIndexes)
}

func addArtistQueryIndexes(app core.App) error {
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}
	for _, index := range artistQueryIndexes {
		artists.Indexes = appendIndex(artists.Indexes, index)
	}
	return app.Save(artists)
}

func removeArtistQueryIndexes(app core.App) error {
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}
	artists.Indexes = removeIndex(
		artists.Indexes,
		artistIndexPublishedFiling,
		artistIndexPublishedFilingDesc,
		artistIndexPublishedBirth,
	)
	return app.Save(artists)
}
