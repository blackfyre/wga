package migrations

import (
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	guestbookModerationIndex = "pbx_guestbook_public_archive"
	guestbookRetentionIndex  = "pbx_guestbook_private_retention"
)

func init() {
	m.Register(addGuestbookModeration, removeGuestbookModeration)
}

func addGuestbookModeration(app core.App) error {
	guestbook, err := app.FindCollectionByNameOrId("guestbook")
	if err != nil {
		return err
	}

	if guestbook.Fields.GetByName("moderation_state") == nil {
		guestbook.Fields.Add(&core.SelectField{
			Name:      "moderation_state",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"unreviewed", "approved", "rejected"},
		})
	}
	if guestbook.Fields.GetByName("retention_until") == nil {
		guestbook.Fields.Add(&core.DateField{Name: "retention_until"})
	}
	guestbook.Indexes = appendIndex(guestbook.Indexes,
		"CREATE INDEX `"+guestbookModerationIndex+"` ON `Guestbook` (`moderation_state`, `created` DESC, `id`)",
	)
	guestbook.Indexes = appendIndex(guestbook.Indexes,
		"CREATE INDEX `"+guestbookRetentionIndex+"` ON `Guestbook` (`moderation_state`, `retention_until`, `id`)",
	)
	if err := app.Save(guestbook); err != nil {
		return err
	}

	// Backfill existing rows to the private unreviewed outcome with a single
	// set-based update. A per-record app.Save would revalidate every field and
	// abort on legacy optional data (for example empty or otherwise invalid
	// email values) that predates current validation. Only moderation_state and
	// retention_until are changed; every other stored byte is left untouched.
	retentionUntil := time.Now().UTC().Add(90 * 24 * time.Hour).Format("2006-01-02 15:04:05.000Z")
	_, err = app.DB().Update(
		guestbook.Name,
		dbx.Params{
			"moderation_state": "unreviewed",
			"retention_until":  retentionUntil,
		},
		dbx.HashExp{"moderation_state": ""},
	).Execute()
	if err != nil {
		return err
	}

	return nil
}

func removeGuestbookModeration(app core.App) error {
	guestbook, err := app.FindCollectionByNameOrId("guestbook")
	if err != nil {
		return err
	}

	guestbook.Indexes = removeIndex(guestbook.Indexes, guestbookModerationIndex, guestbookRetentionIndex)
	// Moderation and retention data are authoritative and intentionally survive
	// rollback. Operators must disable the public guestbook route until the active
	// implementation enforces moderation_state on every public query.
	return app.Save(guestbook)
}

func appendIndex(indexes []string, index string) []string {
	nameStart := strings.Index(index, "`") + 1
	nameEnd := strings.Index(index[nameStart:], "`") + nameStart
	name := index[nameStart:nameEnd]
	for _, existing := range indexes {
		if strings.Contains(existing, "`"+name+"`") {
			return indexes
		}
	}
	return append(indexes, index)
}

func removeIndex(indexes []string, names ...string) []string {
	kept := indexes[:0]
	for _, index := range indexes {
		remove := false
		for _, name := range names {
			if strings.Contains(index, "`"+name+"`") {
				remove = true
				break
			}
		}
		if !remove {
			kept = append(kept, index)
		}
	}
	return kept
}
