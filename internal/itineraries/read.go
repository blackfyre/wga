package itineraries

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
)

// IsAdded reports whether the session draft already contains the artwork. It
// fails closed (false) on any error.
func IsAdded(app core.App, owner string, artworkID string) bool {
	draft, err := FindDraft(app, owner)
	if err != nil {
		return false
	}

	_, err = findStopByArtwork(app, draft.Id, artworkID)
	return err == nil
}

// TrayView is the workflow-level tray projection. ArtworkIDs holds up to three
// artwork ids for tray thumbnails; the handler resolves delivery URLs.
type TrayView struct {
	Count      int
	ArtworkIDs []string
}

// LoadTrayView returns the session draft's persistent-tray projection. An
// absent draft yields an empty view rather than an error.
func LoadTrayView(app core.App, owner string) (TrayView, error) {
	draft, err := FindDraft(app, owner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TrayView{}, nil
		}
		return TrayView{}, err
	}

	stops, err := LoadStops(app, draft.Id)
	if err != nil {
		return TrayView{}, err
	}

	view := TrayView{Count: len(stops), ArtworkIDs: make([]string, 0, 3)}
	for _, stop := range stops {
		if len(view.ArtworkIDs) == 3 {
			break
		}
		view.ArtworkIDs = append(view.ArtworkIDs, stop.GetString("artwork"))
	}

	return view, nil
}
