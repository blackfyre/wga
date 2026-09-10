package artworks

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbsearch "github.com/pocketbase/pocketbase/tools/search"
)

type artworkPageRow struct {
	ID    string `db:"id"`
	Total int    `db:"total"`
}

// attachArtworkConditions attaches the catalogue filter conditions to the query.
func attachArtworkConditions(query *dbx.SelectQuery, resolver *core.RecordFieldResolver, f *filters) error {
	filterString, params := f.BuildFilter()
	if filterString != "" {
		expression, err := pbsearch.FilterData(filterString).BuildExpr(resolver, params)
		if err != nil {
			return err
		}
		query.AndWhere(expression)
	}

	return nil
}

// attachArtworkSort attaches the deterministic sort. Unknown or missing values
// (source_row = 0 for catalogue, date_start = 0 for date) are kept after all
// authoritative values via a raw order-by prefix, then fall through to the
// resolver sort string for the criterion and id tie-break columns.
func attachArtworkSort(query *dbx.SelectQuery, resolver *core.RecordFieldResolver, f *filters) error {
	if prefix := f.sortPrefixOrderBy(); prefix != "" {
		query.AndOrderBy(prefix)
	}

	for _, sortField := range pbsearch.ParseSortFromString(f.sortString()) {
		expression, err := sortField.BuildExpr(resolver)
		if err != nil {
			return err
		}
		if expression != "" {
			query.AndOrderBy(expression)
		}
	}

	return nil
}

// listArtworkPageRows evaluates the filtered, ordered page and its complete
// result count in one pass. Record hydration is deliberately separate and
// bounded to these IDs so the expensive filter predicates are not repeated.
func listArtworkPageRowsForCollection(app *pocketbase.PocketBase, collection *core.Collection, f *filters, limit int, offset int) ([]artworkPageRow, error) {
	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	query := app.RecordQuery(collection).Select(baseID+" AS id", "COUNT(*) OVER() AS total")
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)
	if err := attachArtworkConditions(query, resolver, f); err != nil {
		return nil, err
	}
	if err := attachArtworkSort(query, resolver, f); err != nil {
		return nil, err
	}
	if err := resolver.UpdateQuery(query); err != nil {
		return nil, err
	}
	if offset > 0 {
		query.Offset(int64(offset))
	}
	if limit > 0 {
		query.Limit(int64(limit))
	}

	rows := []artworkPageRow{}
	if err := query.All(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// listArtworkRecordsByPageRowsForCollection hydrates the bounded page and
// restores the exact order selected by listArtworkPageRowsForCollection.
func listArtworkRecordsByPageRowsForCollection(app *pocketbase.PocketBase, collection *core.Collection, rows []artworkPageRow) ([]*core.Record, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	ids := make([]any, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}

	records := []*core.Record{}
	if err := app.RecordQuery(collection).AndWhere(dbx.In(baseID, ids...)).All(&records); err != nil {
		return nil, err
	}
	byID := make(map[string]*core.Record, len(records))
	for _, record := range records {
		byID[record.Id] = record
	}
	ordered := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if record := byID[row.ID]; record != nil {
			ordered = append(ordered, record)
		}
	}
	if len(ordered) != len(rows) {
		return nil, fmt.Errorf("hydrate artwork page: found %d of %d selected records", len(ordered), len(rows))
	}
	return ordered, nil
}
