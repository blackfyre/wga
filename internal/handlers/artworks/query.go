package artworks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbsearch "github.com/pocketbase/pocketbase/tools/search"
)

type artworkPageRow struct {
	ID    string `db:"id"`
	Total int    `db:"total"`
}

var errArtworkPageChanged = errors.New("artwork page changed during hydration")

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
// distinct-artwork count in one pass.
func listArtworkPageRowsForCollection(app core.App, collection *core.Collection, f *filters, limit int, offset int) ([]artworkPageRow, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("list artwork page rows: limit must be positive")
	}
	if offset < 0 {
		offset = 0
	}

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
	query.AndGroupBy(baseID)
	query.Offset(int64(offset))
	query.Limit(int64(limit))

	rows := []artworkPageRow{}
	if err := query.All(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// listCanonicalArtworkPageRows recovers the total and canonical last page after
// a requested offset returns no rows. It materialises the filtered IDs once, so
// recovery does not issue separate first- and last-page window scans.
func listCanonicalArtworkPageRows(app core.App, collection *core.Collection, f *filters, limit int) ([]artworkPageRow, error) {
	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	query := app.RecordQuery(collection).Select(baseID + " AS id")
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
	orderBy := strings.Join(query.Info().OrderBy, ", ")
	if orderBy == "" {
		return nil, fmt.Errorf("list canonical artwork page rows: missing deterministic order")
	}
	query.AndSelect("ROW_NUMBER() OVER (ORDER BY " + orderBy + ") AS position")
	query.AndGroupBy(baseID)

	built := query.Build()
	params := built.Params()
	params["artwork_page_limit"] = limit
	pageQuery := app.DB().NewQuery(`
		WITH matches AS MATERIALIZED (` + built.SQL() + `),
		tally AS (
			SELECT COUNT(*) AS total
			FROM matches
		),
		bounds AS (
			SELECT total, ((total - 1) / {:artwork_page_limit}) * {:artwork_page_limit} AS page_offset
			FROM tally
		)
		SELECT matches.id, bounds.total
		FROM matches
		CROSS JOIN bounds
		WHERE matches.position > bounds.page_offset
			AND matches.position <= bounds.page_offset + {:artwork_page_limit}
		ORDER BY matches.position
	`).Bind(params)

	rows := []artworkPageRow{}
	if err := pageQuery.All(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// listArtworkRecordsByPageRowsForCollection revalidates the bounded page
// against the active filters while hydrating it. A mismatch tells the caller to
// repeat selection rather than rendering a stale or deleted record.
func listArtworkRecordsByPageRowsForCollection(app core.App, collection *core.Collection, f *filters, rows []artworkPageRow) ([]*core.Record, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	baseID := app.DB().QuoteSimpleTableName(collection.Name) + "." + app.DB().QuoteSimpleColumnName("id")
	ids := make([]any, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	query := app.RecordQuery(collection).AndWhere(dbx.In(baseID, ids...))
	resolver := core.NewRecordFieldResolver(app, collection, nil, true)
	if err := attachArtworkConditions(query, resolver, f); err != nil {
		return nil, err
	}
	if err := resolver.UpdateQuery(query); err != nil {
		return nil, err
	}

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}
	if len(records) != len(rows) {
		return nil, errArtworkPageChanged
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
	return ordered, nil
}
