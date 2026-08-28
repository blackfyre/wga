package repositories

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/list"
)

// artistRecordWorksPreviewLimit bounds the published-works preview rendered on
// an artist record. The full catalogue remains reachable through the wider
// /artworks?artist= route.
const artistRecordWorksPreviewLimit = 12

// ArtistRecordRepository is the bounded read-model for a public artist record.
// It only exposes published artists and the published works that belong to
// them, plus a deterministic period-music match derived from stored data.
type ArtistRecordRepository struct {
	app core.App
}

func NewArtistRecordRepository(app core.App) *ArtistRecordRepository {
	return &ArtistRecordRepository{app: app}
}

// FindPublishedArtist returns the published artist with the given id, or
// sql.ErrNoRows when the artist is missing, unpublished, or lacks either
// authoritative identity field (filing or short form).
func (r *ArtistRecordRepository) FindPublishedArtist(id string) (*core.Record, error) {
	present, err := artistsIdentityFieldsPresent(r.app)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, sql.ErrNoRows
	}

	return r.app.FindRecordById(constants.CollectionArtists, id, func(q *dbx.SelectQuery) error {
		q.AndWhere(dbx.NewExp("published = true"))
		q.AndWhere(dbx.NewExp(publishedArtistIdentity))
		return nil
	})
}

// CountPublishedWorks returns the number of published artworks whose author
// relation includes the given artist.
func (r *ArtistRecordRepository) CountPublishedWorks(artistID string) (int, error) {
	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp(
		"EXISTS (SELECT 1 FROM json_each(Artworks.author) je WHERE je.value = {:artist_id})",
		dbx.Params{"artist_id": artistID},
	))
	query.Select("COUNT(DISTINCT id)")

	var count int
	if err := query.Row(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// ListPublishedWorks returns a bounded, deterministic preview of the artist's
// published works ordered by title then id.
func (r *ArtistRecordRepository) ListPublishedWorks(artistID string, limit int) ([]*core.Record, error) {
	if limit <= 0 {
		limit = artistRecordWorksPreviewLimit
	}

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp(
		"EXISTS (SELECT 1 FROM json_each(Artworks.author) je WHERE je.value = {:artist_id})",
		dbx.Params{"artist_id": artistID},
	))
	query.OrderBy("title ASC", "id ASC")
	query.Limit(int64(limit))

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// PeriodSong is a deterministic period-music projection: the matching song
// record and the name of its first composer.
type PeriodSong struct {
	Record   *core.Record
	Composer string
}

// ListSchoolNames resolves the display names of the given school ids in a
// single bounded query and returns them in the supplied (relation) order. The
// artists.school relation caps at ten ids, so the query limit is bounded by
// the input size rather than loading every school.
func (r *ArtistRecordRepository) ListSchoolNames(schoolIDs []string) ([]string, error) {
	if len(schoolIDs) == 0 {
		return nil, nil
	}

	query := r.app.RecordQuery(constants.CollectionSchools)
	query.AndWhere(dbx.In("Schools.id", list.ToInterfaceSlice(schoolIDs)...))
	query.Limit(int64(len(schoolIDs)))

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	byID := make(map[string]string, len(records))
	for _, record := range records {
		if name := record.GetString("name"); name != "" {
			byID[record.Id] = name
		}
	}

	names := make([]string, 0, len(schoolIDs))
	for _, id := range schoolIDs {
		if name, ok := byID[id]; ok {
			names = append(names, name)
		}
	}

	return names, nil
}

// ListMatchingArtPeriods returns the art periods whose inclusive [start, end]
// range contains the birth year, bounded to two rows. Two rows are enough to
// distinguish none, one unambiguous, and ambiguous matches without loading the
// full period table.
func (r *ArtistRecordRepository) ListMatchingArtPeriods(birthYear int) ([]*core.Record, error) {
	if birthYear <= 0 {
		return nil, nil
	}

	query := r.app.RecordQuery("art_periods")
	query.AndWhere(dbx.NewExp("start > 0 AND end > 0 AND start <= end"))
	query.AndWhere(dbx.NewExp("start <= {:birth} AND end >= {:birth}", dbx.Params{"birth": birthYear}))
	query.OrderBy("name ASC", "id ASC")
	query.Limit(2)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// MatchPeriodSong returns one deterministic period-appropriate song whose
// composer century matches the artist's birth century, or nil when the birth
// year is unknown or no composer carries a matching century. The stored
// composer century is the sole source evidence for the match. Both the song and
// its matching composer must be explicitly published; unpublished recordings
// are never surfaced by the period-music card.
func (r *ArtistRecordRepository) MatchPeriodSong(birthYear int) (*PeriodSong, error) {
	century := centuryForYear(birthYear)
	if century == "" {
		return nil, nil
	}

	song := &core.Record{}
	err := r.app.RecordQuery("music_song").
		AndWhere(dbx.NewExp("published = true")).
		AndWhere(dbx.NewExp(
			"EXISTS (SELECT 1 FROM json_each(Music_song.composer) je JOIN Music_composer mc ON mc.id = je.value WHERE mc.century = {:century} AND mc.published = true)",
			dbx.Params{"century": century},
		)).
		OrderBy("title ASC", "id ASC").
		Limit(1).
		One(song)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	composer := ""
	if ids := song.GetStringSlice("composer"); len(ids) > 0 {
		if record, err := r.app.FindRecordById("music_composer", ids[0]); err == nil {
			composer = record.GetString("name")
		}
	}

	return &PeriodSong{Record: song, Composer: composer}, nil
}

// centuryForYear returns the one-based century containing the year, or an empty
// string when the year is unknown. 1606 -> "17", 1900 -> "19", 1901 -> "20".
func centuryForYear(year int) string {
	if year <= 0 {
		return ""
	}

	return strconv.Itoa((year-1)/100 + 1)
}
