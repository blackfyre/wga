package seed

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	guestbookStateApproved   = "approved"
	guestbookStateUnreviewed = "unreviewed"
)

// PromoteProductionGuestbookEntries approves valid historical entries from an
// external producer database. It uses the producer's stable entry IDs, so it
// cannot change locally created guestbook records. Source email addresses are
// cleared for every matching row, including malformed entries left unreviewed.
func PromoteProductionGuestbookEntries(app core.App, sqlitePath string) error {
	if sqlitePath == "" {
		return nil
	}

	entries, err := loadProductionGuestbookEntries(sqlitePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		values := dbx.Params{
			"email": "",
		}
		if entry.isHistoricalArchiveEntry() {
			values["moderation_state"] = guestbookStateApproved
			values["retention_until"] = ""
		} else {
			values["moderation_state"] = guestbookStateUnreviewed
		}

		if _, err := app.DB().Update(
			constants.CollectionGuestbook,
			values,
			dbx.HashExp{"id": entry.ID},
		).Execute(); err != nil {
			return fmt.Errorf("promote guestbook entry %q: %w", entry.ID, err)
		}
	}

	return nil
}

func loadProductionGuestbookEntries(sqlitePath string) ([]sourceGuestbookEntry, error) {
	connectionURL := (&url.URL{
		Scheme:   "file",
		Path:     sqlitePath,
		RawQuery: "mode=ro",
	}).String()
	db, err := sql.Open("sqlite", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("open guestbook source SQLite database: %w", err)
	}
	defer closeDatabase(db)

	entries, err := loadGuestbookEntries(db)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (entry sourceGuestbookEntry) isHistoricalArchiveEntry() bool {
	if strings.TrimSpace(entry.Name) == "" {
		return false
	}
	if _, err := time.Parse("2006-01-02 15:04:05.000Z", entry.Created); err != nil {
		return false
	}
	_, err := time.Parse("2006-01-02 15:04:05.000Z", entry.Updated)
	return err == nil
}
