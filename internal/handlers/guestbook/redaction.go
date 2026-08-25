package guestbook

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRedactionHook enforces the guestbook withdrawal-redaction contract at
// the persistence boundary. A PocketBase superuser owns moderation and may move
// an approved entry to any non-approved outcome; that withdrawal must
// irreversibly remove the visitor-supplied personal fields so the original
// values cannot be recovered from the application record afterwards.
//
// The hook is bound to the model-level update event so it also applies to
// superuser edits made through the PocketBase admin UI, and it runs inside the
// record save so the clearing is atomic with the withdrawal. It only clears
// when an approved entry leaves the approved state, which makes it idempotent:
// re-saving an already withdrawn entry, or editing an entry that stays
// approved, leaves the fields untouched.
func RegisterRedactionHook(app core.App) {
	app.OnRecordUpdate(constants.CollectionGuestbook).BindFunc(func(e *core.RecordEvent) error {
		original := e.Record.Original()
		if original.GetString("moderation_state") != guestbookStateApproved {
			return e.Next()
		}
		if e.Record.GetString("moderation_state") == guestbookStateApproved {
			return e.Next()
		}

		e.Record.Set("name", "")
		e.Record.Set("email", "")
		e.Record.Set("location", "")
		e.Record.Set("message", "")

		return e.Next()
	})
}
