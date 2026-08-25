package guestbook

import (
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type repository struct {
	app core.App
}

func (r repository) createUnreviewed(input submissionInput, now time.Time) error {
	collection, err := r.app.FindCollectionByNameOrId(constants.CollectionGuestbook)
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("name", input.Name)
	record.Set("location", input.Location)
	record.Set("message", input.Message)
	record.Set("moderation_state", guestbookStateUnreviewed)
	record.Set("retention_until", formatPocketBaseTime(now.Add(guestbookPrivateRetention)))

	return r.app.Save(record)
}

func (r repository) deleteExpiredPrivate(now time.Time) (int, error) {
	records, err := r.app.FindRecordsByFilter(
		constants.CollectionGuestbook,
		"moderation_state != {:approved} && retention_until != '' && retention_until <= {:now}",
		"+retention_until",
		100,
		0,
		dbx.Params{"approved": guestbookStateApproved, "now": formatPocketBaseTime(now)},
	)
	if err != nil {
		return 0, err
	}

	for _, record := range records {
		if err := r.app.Delete(record); err != nil {
			return 0, err
		}
	}

	return len(records), nil
}

// PurgeExpiredPrivateEntries removes at most 100 expired, non-approved entries.
// The central lifecycle job owns scheduling and run-scoped logging and may call
// this repeatedly until the returned count is less than 100.
func PurgeExpiredPrivateEntries(app core.App, now time.Time) (int, error) {
	return (repository{app: app}).deleteExpiredPrivate(now)
}

func (r repository) publicCounts(selected filters) (int, int, error) {
	filter, params := publicFilter(selected)
	total, err := utils.CountRecordsByFilter(
		r.app,
		constants.CollectionGuestbook,
		"moderation_state = {:approved}",
		dbx.Params{"approved": guestbookStateApproved},
	)
	if err != nil {
		return 0, 0, err
	}

	scopeTotal, err := utils.CountRecordsByFilter(r.app, constants.CollectionGuestbook, filter, params)
	if err != nil {
		return 0, 0, err
	}

	return total, scopeTotal, nil
}

func (r repository) publicEntries(selected filters, limit int) ([]dto.GuestbookEntry, error) {
	filter, params := publicFilter(selected)
	records, err := r.app.FindRecordsByFilter(
		constants.CollectionGuestbook,
		filter,
		"-created,-id",
		limit,
		0,
		params,
	)
	if err != nil {
		return nil, err
	}

	entries := make([]dto.GuestbookEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, dto.GuestbookEntry{
			Name:     record.GetString("name"),
			Location: record.GetString("location"),
			Message:  record.GetString("message"),
			Created:  record.GetDateTime("created").Time().Format("2006-01-02"),
		})
	}

	return entries, nil
}

// approvedYears returns the distinct calendar years of approved entries,
// newest first, as a single database projection. Missing or empty created
// timestamps are excluded, so the result contains no empty year strings.
func (r repository) approvedYears() ([]string, error) {
	collection, err := r.app.FindCollectionByNameOrId(constants.CollectionGuestbook)
	if err != nil {
		return nil, err
	}

	table := r.app.DB().QuoteSimpleTableName(collection.Name)
	rows := []struct {
		Year string `db:"year"`
	}{}
	err = r.app.DB().NewQuery(
		"SELECT DISTINCT SUBSTR(created, 1, 4) AS year" +
			" FROM " + table +
			" WHERE moderation_state = {:approved} AND created IS NOT NULL AND created != ''" +
			" ORDER BY year DESC",
	).Bind(dbx.Params{"approved": guestbookStateApproved}).All(&rows)
	if err != nil {
		return nil, err
	}

	years := make([]string, 0, len(rows))
	for _, row := range rows {
		years = append(years, row.Year)
	}

	return years, nil
}

func formatPocketBaseTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000Z")
}

func publicFilter(filters filters) (string, dbx.Params) {
	parts := []string{"moderation_state = {:approved}"}
	params := dbx.Params{"approved": guestbookStateApproved}

	if filters.Year != "all" {
		parts = append(parts, "created ~ {:year}")
		params["year"] = filters.Year
	}
	if filters.Query != "" {
		parts = append(parts, "(name ~ {:query} || location ~ {:query} || message ~ {:query})")
		params["query"] = filters.Query
	}

	return strings.Join(parts, " && "), params
}
