package repositories

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/list"
)

// artistSelectionsLimit bounds the published selections listed for one artist.
// The relation itself caps at a few dozen per artist in the source data, but
// the read-model never issues an unbounded query regardless.
const artistSelectionsLimit = 100

// ArtistSelectionsRepository is the bounded read-model for source-backed curated
// artist selections. It only exposes published selections that belong to the
// given artist and their ordered artwork membership.
type ArtistSelectionsRepository struct {
	app core.App
}

func NewArtistSelectionsRepository(app core.App) *ArtistSelectionsRepository {
	return &ArtistSelectionsRepository{app: app}
}

// CountPublishedSelections returns the number of published selections owned by
// the artist.
func (r *ArtistSelectionsRepository) CountPublishedSelections(artistID string) (int, error) {
	query := r.app.RecordQuery(constants.CollectionSelections)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(selectionArtistFilter(artistID))
	query.Select("COUNT(DISTINCT id)")

	var count int
	if err := query.Row(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// ListPublishedSelections returns a bounded, deterministic list of the artist's
// published selections ordered by display title then id.
func (r *ArtistSelectionsRepository) ListPublishedSelections(artistID string, limit int) ([]*core.Record, error) {
	if limit <= 0 {
		limit = artistSelectionsLimit
	}

	query := r.app.RecordQuery(constants.CollectionSelections)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(selectionArtistFilter(artistID))
	query.OrderBy("display_title ASC", "id ASC")
	query.Limit(int64(limit))

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// FindPublishedSelection returns the published selection with the given id and
// artist, or sql.ErrNoRows when the selection is missing, unpublished, or owned
// by a different artist.
func (r *ArtistSelectionsRepository) FindPublishedSelection(artistID string, selectionID string) (*core.Record, error) {
	return r.app.FindRecordById(constants.CollectionSelections, selectionID, func(q *dbx.SelectQuery) error {
		q.AndWhere(dbx.NewExp("published = true"))
		q.AndWhere(selectionArtistFilter(artistID))
		return nil
	})
}

// ListSelectionArtworks returns the selection's ordered artwork records in the
// exact order of the stored artworks relation. A work is returned only when it
// is published and its author relation includes the selection artist, so a
// malformed selection that references another artist's published work never
// renders it.
func (r *ArtistSelectionsRepository) ListSelectionArtworks(artistID string, selection *core.Record) ([]*core.Record, error) {
	ids := selection.GetStringSlice("artworks")
	if len(ids) == 0 {
		return nil, nil
	}

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(artworkAuthorFilter(artistID))
	query.AndWhere(dbx.In("Artworks.id", list.ToInterfaceSlice(ids)...))
	query.Limit(int64(len(ids)))

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	byID := make(map[string]*core.Record, len(records))
	for _, record := range records {
		byID[record.Id] = record
	}

	ordered := make([]*core.Record, 0, len(ids))
	for _, id := range ids {
		if record, ok := byID[id]; ok {
			ordered = append(ordered, record)
		}
	}

	return ordered, nil
}

// selectionArtistFilter scopes a query to the required single-artist relation.
// The artist relation is stored as a plain TEXT column (MaxSelect 1), so it is
// matched directly rather than through the JSON membership used by multi-select
// relations.
func selectionArtistFilter(artistID string) dbx.Expression {
	return dbx.NewExp("Art_selections.artist = {:artist_id}", dbx.Params{"artist_id": artistID})
}

// artworkAuthorFilter scopes an artwork query to records whose author relation
// includes the given artist. The artworks.author relation is stored as a JSON
// array, so it is matched through json_each like the artwork record read-model.
func artworkAuthorFilter(artistID string) dbx.Expression {
	return dbx.NewExp(
		"EXISTS (SELECT 1 FROM json_each(Artworks.author) je WHERE je.value = {:artist_id})",
		dbx.Params{"artist_id": artistID},
	)
}
