package artworks

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbsearch "github.com/pocketbase/pocketbase/tools/search"
)

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

// listArtworkRecords returns the bounded page of artwork records matching the
// filters, in deterministic sort order.
func listArtworkRecords(app *pocketbase.PocketBase, f *filters, limit int, offset int) ([]*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return nil, err
	}

	query := app.RecordQuery(collection)
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

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// countArtworkRecords returns the number of artwork records matching the
// filters. The count ignores sorting and reuses the same condition set so the
// page total always matches the list query.
func countArtworkRecords(app *pocketbase.PocketBase, f *filters) (int, error) {
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return 0, err
	}

	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")

	query := app.RecordQuery(collection).Select("COUNT(DISTINCT " + baseID + ")")
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)

	if err := attachArtworkConditions(query, resolver, f); err != nil {
		return 0, err
	}
	if err := resolver.UpdateQuery(query); err != nil {
		return 0, err
	}

	var count int
	if err := query.Row(&count); err != nil {
		return 0, err
	}

	return count, nil
}
