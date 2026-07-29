package utils

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/search"
)

// CountRecordsByFilter returns the number of records matching a PocketBase filter.
func CountRecordsByFilter(app core.App, collectionName string, filter string, params dbx.Params) (int, error) {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return 0, err
	}

	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	query := app.RecordQuery(collection).Select("COUNT(DISTINCT " + baseID + ")")
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)
	if filter != "" {
		expression, err := search.FilterData(filter).BuildExpr(resolver, params)
		if err != nil {
			return 0, err
		}
		query.AndWhere(expression)
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
