package itineraries

import (
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// sentinelPosition is an out-of-band position used to reorder stops without
// violating the unique (itinerary, position) index during a swap.
const sentinelPosition = -1

// Draft pairs a session-owned itinerary record with its ordered stops.
type Draft struct {
	Record *core.Record
	Stops  []*core.Record
}

// FindDraft returns the single session-owned draft for the owner digest, or
// sql.ErrNoRows when none exists yet.
func FindDraft(app core.App, owner string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		CollectionItineraries,
		"owner = {:owner} && status = {:status}",
		dbx.Params{"owner": owner, "status": string(StatusDraft)},
	)
}

// EnsureDraft returns the session's draft, creating it when absent. The
// partial unique index (owner WHERE status='draft') backstops the check so a
// race cannot create two drafts for one session.
func EnsureDraft(app core.App, owner string) (*core.Record, error) {
	draft, err := FindDraft(app, owner)
	if err == nil {
		return draft, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	collection, err := app.FindCollectionByNameOrId(CollectionItineraries)
	if err != nil {
		return nil, err
	}

	draft = core.NewRecord(collection)
	draft.Set("owner", owner)
	draft.Set("status", string(StatusDraft))
	draft.Set("listed", false)
	if err := app.Save(draft); err != nil {
		// A concurrent request may have won the race; re-resolve the draft.
		if existing, findErr := FindDraft(app, owner); findErr == nil {
			return existing, nil
		}
		return nil, err
	}

	// Reload the freshly persisted draft so its Original() snapshot matches the
	// saved state; otherwise the lifecycle hook would observe the blank
	// NewRecord defaults on the subsequent update.
	return FindDraft(app, owner)
}

// LoadDraft returns the session draft and its stops ordered by position.
func LoadDraft(app core.App, owner string) (*Draft, error) {
	draft, err := FindDraft(app, owner)
	if err != nil {
		return nil, err
	}

	stops, err := LoadStops(app, draft.Id)
	if err != nil {
		return nil, err
	}

	return &Draft{Record: draft, Stops: stops}, nil
}

// LoadStops returns a draft or itinerary's stops in running order.
func LoadStops(app core.App, itineraryID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter(
		CollectionItineraryStops,
		"itinerary = {:itinerary}",
		"+position",
		0,
		0,
		dbx.Params{"itinerary": itineraryID},
	)
}

// AvailableStops returns the ordered stops whose artwork still exists and is
// currently published, preserving running order. Persisted snapshots are left
// untouched; callers that must surface unavailable entries (the builder) use
// LoadStops and mark the stop unavailable themselves.
func AvailableStops(app core.App, itineraryID string) ([]*core.Record, error) {
	stops, err := LoadStops(app, itineraryID)
	if err != nil {
		return nil, err
	}

	available := make([]*core.Record, 0, len(stops))
	for _, stop := range stops {
		if !stopArtworkAvailable(app, stop.GetString("artwork")) {
			continue
		}
		available = append(available, stop)
	}

	return available, nil
}

// stopArtworkAvailable reports whether an artwork exists and is published.
func stopArtworkAvailable(app core.App, artworkID string) bool {
	artwork, err := app.FindRecordById(constants.CollectionArtworks, artworkID)
	if err != nil {
		return false
	}

	return artwork.GetBool("published")
}

// AddStop appends a published artwork to the session draft. The operation is
// idempotent: adding an artwork already in the draft returns the existing stop
// without growing the draft, and the database unique index rejects duplicates.
func AddStop(app core.App, owner string, artworkID string) (*core.Record, error) {
	artwork, err := findPublishedArtwork(app, artworkID)
	if err != nil {
		return nil, err
	}

	var added *core.Record
	err = app.RunInTransaction(func(txApp core.App) error {
		draft, err := EnsureDraft(txApp, owner)
		if err != nil {
			return err
		}
		if draft.GetString("status") != string(StatusDraft) {
			return ErrNotDraft
		}

		existing, err := findStopByArtwork(txApp, draft.Id, artworkID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if existing != nil {
			added = existing
			return nil
		}

		stops, err := LoadStops(txApp, draft.Id)
		if err != nil {
			return err
		}
		if len(stops) >= MaxStops {
			return ErrStopLimit
		}

		collection, err := txApp.FindCollectionByNameOrId(CollectionItineraryStops)
		if err != nil {
			return err
		}
		stop := core.NewRecord(collection)
		stop.Set("itinerary", draft.Id)
		stop.Set("artwork", artworkID)
		stop.Set("title", SanitiseText(artwork.GetString("title")))
		stop.Set("position", len(stops))
		if err := txApp.Save(stop); err != nil {
			// A concurrent insert may have won the artwork or position slot
			// despite the serialised transaction; translate it to a bounded
			// domain result rather than leaking a server failure.
			if uniqueViolation(err) {
				if existing, findErr := findStopByArtwork(txApp, draft.Id, artworkID); findErr == nil && existing != nil {
					added = existing
					return nil
				}
				return ErrStopLimit
			}
			return err
		}

		added = stop
		return touchDraft(txApp, draft)
	})
	if err != nil {
		return nil, err
	}

	return added, nil
}

// RemoveStop deletes a stop from the session draft and renumbers the remaining
// positions so the order stays contiguous.
func RemoveStop(app core.App, owner string, stopID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		draft, err := FindDraft(txApp, owner)
		if err != nil {
			return err
		}

		stop, err := findStop(txApp, draft.Id, stopID)
		if err != nil {
			return err
		}
		if err := txApp.Delete(stop); err != nil {
			return err
		}

		if err := renumberStops(txApp, draft.Id); err != nil {
			return err
		}

		return touchDraft(txApp, draft)
	})
}

// MoveStop reorders a stop one place up or down. dir is "up" or "down".
func MoveStop(app core.App, owner string, stopID string, dir string) error {
	if dir != "up" && dir != "down" {
		return ErrInvalidMove
	}

	return app.RunInTransaction(func(txApp core.App) error {
		draft, err := FindDraft(txApp, owner)
		if err != nil {
			return err
		}

		stops, err := LoadStops(txApp, draft.Id)
		if err != nil {
			return err
		}

		index := stopIndex(stops, stopID)
		if index < 0 {
			return ErrStopNotFound
		}

		swapWith := index - 1
		if dir == "down" {
			swapWith = index + 1
		}
		if swapWith < 0 || swapWith >= len(stops) {
			return nil
		}

		// Move via a temporary out-of-band position so the unique
		// (itinerary, position) index is never violated mid-swap.
		current, other := stops[index], stops[swapWith]
		current.Set("position", sentinelPosition)
		if err := txApp.Save(current); err != nil {
			return err
		}
		other.Set("position", index)
		if err := txApp.Save(other); err != nil {
			return err
		}
		current.Set("position", swapWith)
		if err := txApp.Save(current); err != nil {
			return err
		}

		return touchDraft(txApp, draft)
	})
}

// SetNarration validates and stores a stop's narration.
func SetNarration(app core.App, owner string, stopID string, narration string) error {
	narration = SanitiseText(narration)
	if len([]rune(narration)) > MaxNarrationLength {
		narration = string([]rune(narration)[:MaxNarrationLength])
	}

	return app.RunInTransaction(func(txApp core.App) error {
		draft, err := FindDraft(txApp, owner)
		if err != nil {
			return err
		}

		stop, err := findStop(txApp, draft.Id, stopID)
		if err != nil {
			return err
		}
		stop.Set("narration", narration)
		if err := txApp.Save(stop); err != nil {
			return err
		}

		return touchDraft(txApp, draft)
	})
}

// ClearDraft deletes every stop in the session draft.
func ClearDraft(app core.App, owner string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		draft, err := FindDraft(txApp, owner)
		if err != nil {
			return err
		}

		stops, err := LoadStops(txApp, draft.Id)
		if err != nil {
			return err
		}
		for _, stop := range stops {
			if err := txApp.Delete(stop); err != nil {
				return err
			}
		}

		return touchDraft(txApp, draft)
	})
}

// Meta holds the bounded, sanitised draft metadata. Listed is deliberately not
// part of Meta: visibility is a publish-time choice set through SetListed, so
// ordinary metadata edits never reset the maker's listing decision.
type Meta struct {
	Title   string
	Intro   string
	Creator string
}

// SetMeta validates and stores the draft metadata.
func SetMeta(app core.App, owner string, meta Meta) error {
	title := truncateRunes(SanitiseText(meta.Title), MaxTitleLength)
	intro := truncateRunes(SanitiseText(meta.Intro), MaxIntroLength)
	creator := truncateRunes(SanitiseText(meta.Creator), MaxCreatorLength)

	return app.RunInTransaction(func(txApp core.App) error {
		draft, err := EnsureDraft(txApp, owner)
		if err != nil {
			return err
		}
		draft.Set("title", title)
		draft.Set("intro", intro)
		draft.Set("creator", creator)
		return txApp.Save(draft)
	})
}

// SetListed persists the publish-time listing choice on the session draft. It
// is a targeted update: title, intro, and creator are left untouched, so the
// publish bar can record visibility without disturbing saved metadata.
func SetListed(app core.App, owner string, listed bool) error {
	return app.RunInTransaction(func(txApp core.App) error {
		draft, err := FindDraft(txApp, owner)
		if err != nil {
			return err
		}
		draft.Set("listed", listed)
		return txApp.Save(draft)
	})
}

// Publish validates the draft and transitions it straight to approved with an
// immutable public token and a fixed one-year expiry. Publication is immediate:
// the token is readable as soon as this returns, and the listed flag (already
// persisted on the draft) governs index discovery. The draft is consumed: its
// stops remain attached to the now-published record and it is no longer the
// session draft, so the next add starts a fresh draft.
func Publish(app core.App, owner string) (*core.Record, error) {
	var published *core.Record
	err := app.RunInTransaction(func(txApp core.App) error {
		if err := enforcePublicationPolicy(txApp, owner); err != nil {
			return err
		}

		draft, err := FindDraft(txApp, owner)
		if err != nil {
			return err
		}
		if draft.GetString("status") != string(StatusDraft) {
			return ErrNotDraft
		}

		title := truncateRunes(SanitiseText(draft.GetString("title")), MaxTitleLength)
		if title == "" {
			return ErrTitleRequired
		}

		stops, err := LoadStops(txApp, draft.Id)
		if err != nil {
			return err
		}
		if len(stops) == 0 {
			return ErrNoStops
		}

		token, err := NewToken()
		if err != nil {
			return err
		}

		now := types.NowDateTime()
		draft.Set("title", title)
		draft.Set("intro", truncateRunes(SanitiseText(draft.GetString("intro")), MaxIntroLength))
		draft.Set("creator", truncateRunes(SanitiseText(draft.GetString("creator")), MaxCreatorLength))
		draft.Set("status", string(StatusApproved))
		draft.Set("token", token)
		draft.Set("published", now)
		draft.Set("expires_at", now.Add(PublicationLifetime))
		if err := txApp.Save(draft); err != nil {
			return err
		}

		published = draft
		return nil
	})
	if err != nil {
		return nil, err
	}

	return published, nil
}

// FindPublishedByToken returns a published itinerary by its immutable token.
// Drafts never carry a token, so a draft cannot be addressed publicly.
func FindPublishedByToken(app core.App, token string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		CollectionItineraries,
		"token = {:token} && status != {:draft}",
		dbx.Params{"token": token, "draft": string(StatusDraft)},
	)
}

// LatestPublished returns the owner's most recently published itinerary,
// deterministically newest-first with a stable id tie-break.
func LatestPublished(app core.App, owner string) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(
		CollectionItineraries,
		"owner = {:owner} && status != {:draft}",
		"-published,-created,-id",
		1,
		0,
		dbx.Params{"owner": owner, "draft": string(StatusDraft)},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}

	return records[0], nil
}

// ListPublished returns approved, unexpired, listed itineraries newest first.
func ListPublished(app core.App, limit int) ([]*core.Record, error) {
	now := types.NowDateTime()
	return app.FindRecordsByFilter(
		CollectionItineraries,
		"status = {:status} && listed = true && expires_at > {:now}",
		"-published",
		limit,
		0,
		dbx.Params{"status": string(StatusApproved), "now": now},
	)
}

// EstimateDuration returns a pre-formatted, conservative reading-time estimate
// derived from the reference formula: one minute per 150 narration words plus
// 0.6 minutes per stop, floored at one minute. It reads only persisted
// narration, so it never fabricates figures from absent commentary.
func EstimateDuration(stops []*core.Record) string {
	words := 0
	for _, stop := range stops {
		words += len(strings.Fields(stop.GetString("narration")))
	}

	minutes := int(math.Round(float64(words)/150 + float64(len(stops))*0.6))
	if minutes < 1 {
		minutes = 1
	}

	return strconv.Itoa(minutes) + " MIN"
}

// IsExpired reports whether a published itinerary has passed its expiry time.
func IsExpired(record *core.Record) bool {
	expires := record.GetDateTime("expires_at")
	if expires.IsZero() {
		return false
	}

	return types.NowDateTime().After(expires)
}

// findPublishedArtwork returns a published artwork or ErrArtworkUnavailable.
func findPublishedArtwork(app core.App, artworkID string) (*core.Record, error) {
	artwork, err := app.FindRecordById(constants.CollectionArtworks, artworkID)
	if err != nil {
		return nil, ErrArtworkUnavailable
	}
	if !artwork.GetBool("published") {
		return nil, ErrArtworkUnavailable
	}

	return artwork, nil
}

func findStop(app core.App, itineraryID string, stopID string) (*core.Record, error) {
	stop, err := app.FindFirstRecordByFilter(
		CollectionItineraryStops,
		"itinerary = {:itinerary} && id = {:id}",
		dbx.Params{"itinerary": itineraryID, "id": stopID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStopNotFound
		}
		return nil, err
	}

	return stop, nil
}

func findStopByArtwork(app core.App, itineraryID string, artworkID string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		CollectionItineraryStops,
		"itinerary = {:itinerary} && artwork = {:artwork}",
		dbx.Params{"itinerary": itineraryID, "artwork": artworkID},
	)
}

func stopIndex(stops []*core.Record, stopID string) int {
	for index, stop := range stops {
		if stop.Id == stopID {
			return index
		}
	}

	return -1
}

func renumberStops(app core.App, itineraryID string) error {
	stops, err := LoadStops(app, itineraryID)
	if err != nil {
		return err
	}
	for index, stop := range stops {
		if stop.GetInt("position") == index {
			continue
		}
		stop.Set("position", index)
		if err := app.Save(stop); err != nil {
			return err
		}
	}

	return nil
}

func truncateRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}

// touchDraft re-saves the draft so its autodate `updated` field advances. It is
// called after every successful stop mutation so active editing is observable
// to the abandoned-draft purge rather than only at first creation.
func touchDraft(app core.App, draft *core.Record) error {
	return app.Save(draft)
}

// enforcePublicationPolicy counts the owner's non-draft records published
// within the rolling window and rejects publication when the budget is spent.
// It runs inside Publish's transaction, so concurrent publications for the
// same owner serialise and cannot jointly exceed the budget.
func enforcePublicationPolicy(app core.App, owner string) error {
	cutoff := types.NowDateTime().Add(-PublicationWindow)
	count, err := countRecordsByFilter(
		app,
		CollectionItineraries,
		"owner = {:owner} && status != {:draft} && published > {:cutoff}",
		dbx.Params{"owner": owner, "draft": string(StatusDraft), "cutoff": cutoff},
	)
	if err != nil {
		return err
	}
	if count >= PublicationBudget {
		return ErrPublishRateLimit
	}

	return nil
}

func countRecordsByFilter(app core.App, collection string, filter string, params dbx.Params) (int, error) {
	records, err := app.FindRecordsByFilter(collection, filter, "", 0, 0, params)
	if err != nil {
		return 0, err
	}

	return len(records), nil
}

// uniqueViolation reports whether err is a SQLite unique-constraint failure,
// used to translate a lost concurrent insert into a bounded domain error.
func uniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
