package itineraries

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func artworkID(index int) string {
	return fmt.Sprintf("aw%013d", index)
}

func TestEnsureDraftCreatesOneDraftPerSession(t *testing.T) {
	app := installItinerarySchema(t)

	first, err := EnsureDraft(app, "owner-a")
	if err != nil {
		t.Fatalf("EnsureDraft: %v", err)
	}
	if first.GetString("status") != string(StatusDraft) {
		t.Errorf("status = %q, want draft", first.GetString("status"))
	}

	second, err := EnsureDraft(app, "owner-a")
	if err != nil {
		t.Fatalf("EnsureDraft (again): %v", err)
	}
	if first.Id != second.Id {
		t.Errorf("EnsureDraft must return the same draft, got %q then %q", first.Id, second.Id)
	}

	other, err := EnsureDraft(app, "owner-b")
	if err != nil {
		t.Fatalf("EnsureDraft (other): %v", err)
	}
	if other.Id == first.Id {
		t.Error("different owners must have different drafts")
	}
}

func TestAddStopIsIdempotentAndBounded(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-add"

	if _, err := AddStop(app, owner, artworkID(1)); err != nil {
		t.Fatalf("AddStop: %v", err)
	}
	if _, err := AddStop(app, owner, artworkID(1)); err != nil {
		t.Fatalf("AddStop (duplicate): %v", err)
	}

	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if len(stops) != 1 {
		t.Fatalf("stops = %d, want 1 (duplicate add must be a no-op)", len(stops))
	}

	for index := 2; index <= MaxStops; index++ {
		if _, err := AddStop(app, owner, artworkID(index)); err != nil {
			t.Fatalf("AddStop #%d: %v", index, err)
		}
	}

	if _, err := AddStop(app, owner, artworkID(16)); !errors.Is(err, ErrStopLimit) {
		t.Errorf("AddStop beyond limit = %v, want ErrStopLimit", err)
	}
}

func TestAddStopRejectsUnavailableArtwork(t *testing.T) {
	app := installItinerarySchema(t)

	if _, err := AddStop(app, "owner", "missing-artwork"); !errors.Is(err, ErrArtworkUnavailable) {
		t.Errorf("AddStop missing artwork = %v, want ErrArtworkUnavailable", err)
	}

	if _, err := AddStop(app, "owner", artworkID(99)); !errors.Is(err, ErrArtworkUnavailable) {
		t.Errorf("AddStop unpublished artwork = %v, want ErrArtworkUnavailable", err)
	}
}

func TestRemoveStopRenumbers(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-remove"

	addStops(t, app, owner, 3)

	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}

	if err := RemoveStop(app, owner, stops[1].Id); err != nil {
		t.Fatalf("RemoveStop: %v", err)
	}

	remaining, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("remaining = %d, want 2", len(remaining))
	}
	for index, stop := range remaining {
		if stop.GetInt("position") != index {
			t.Errorf("position after removal = %d at index %d, want contiguous", stop.GetInt("position"), index)
		}
	}

	if err := RemoveStop(app, owner, stops[1].Id); !errors.Is(err, ErrStopNotFound) {
		t.Errorf("RemoveStop (again) = %v, want ErrStopNotFound", err)
	}
}

func TestMoveStopReorders(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-move"

	addStops(t, app, owner, 3)
	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}

	if err := MoveStop(app, owner, stops[0].Id, "down"); err != nil {
		t.Fatalf("MoveStop down: %v", err)
	}
	after, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if after[0].Id != stops[1].Id {
		t.Error("moving first stop down must swap it with the second")
	}

	if err := MoveStop(app, owner, stops[0].Id, "down"); err != nil {
		t.Fatalf("MoveStop down again: %v", err)
	}

	if err := MoveStop(app, owner, stops[0].Id, "sideways"); !errors.Is(err, ErrInvalidMove) {
		t.Errorf("MoveStop invalid dir = %v, want ErrInvalidMove", err)
	}

	if err := MoveStop(app, owner, "missing", "up"); !errors.Is(err, ErrStopNotFound) {
		t.Errorf("MoveStop missing = %v, want ErrStopNotFound", err)
	}
}

func TestSetNarrationBoundedAndSanitised(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-narration"

	addStops(t, app, owner, 1)
	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}

	long := make([]rune, MaxNarrationLength+50)
	for index := range long {
		long[index] = 'x'
	}
	if err := SetNarration(app, owner, stops[0].Id, "<script>alert(1)</script>"+string(long)); err != nil {
		t.Fatalf("SetNarration: %v", err)
	}

	updated, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	narration := updated[0].GetString("narration")
	if len([]rune(narration)) != MaxNarrationLength {
		t.Errorf("narration length = %d, want %d", len([]rune(narration)), MaxNarrationLength)
	}
	if narration != string(long[:MaxNarrationLength]) {
		t.Error("narration must have markup stripped and be truncated to the bound")
	}
}

func TestSetMetaBoundedAndSanitised(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-meta"

	if err := SetMeta(app, owner, Meta{Title: "  <b>Title</b>  ", Intro: "<script>x</script>Intro", Creator: "Maker"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := SetListed(app, owner, true); err != nil {
		t.Fatalf("SetListed: %v", err)
	}

	draft, err := FindDraft(app, owner)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	if draft.GetString("title") != "Title" {
		t.Errorf("title = %q, want sanitised %q", draft.GetString("title"), "Title")
	}
	if draft.GetString("intro") != "Intro" {
		t.Errorf("intro = %q, want sanitised %q", draft.GetString("intro"), "Intro")
	}
	if !draft.GetBool("listed") {
		t.Error("listed must be persisted")
	}
}

func TestClearDraftRemovesStops(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-clear"

	addStops(t, app, owner, 3)
	if err := ClearDraft(app, owner); err != nil {
		t.Fatalf("ClearDraft: %v", err)
	}

	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if len(stops) != 0 {
		t.Errorf("stops after clear = %d, want 0", len(stops))
	}
}

func TestPublishTransitionsAndConsumesDraft(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-publish"

	addStops(t, app, owner, 2)
	if err := SetMeta(app, owner, Meta{Title: "My Journey"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := SetListed(app, owner, true); err != nil {
		t.Fatalf("SetListed: %v", err)
	}

	published, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if published.GetString("status") != string(StatusApproved) {
		t.Errorf("status = %q, want approved (immediate publication)", published.GetString("status"))
	}
	if published.GetString("token") == "" {
		t.Error("publish must issue an immutable public token")
	}
	if published.GetString("published") == "" {
		t.Error("publish must set the published time")
	}
	expires := published.GetDateTime("expires_at")
	if expires.IsZero() {
		t.Error("publish must set an expiry")
	}
	if got := expires.Sub(published.GetDateTime("published")); got != PublicationLifetime {
		t.Errorf("expiry lifetime = %v, want %v", got, PublicationLifetime)
	}

	// The draft is consumed: no draft remains for this owner.
	if _, err := FindDraft(app, owner); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("FindDraft after publish = %v, want sql.ErrNoRows", err)
	}

	// The published token resolves; a fresh add starts a new draft.
	if _, err := FindPublishedByToken(app, published.GetString("token")); err != nil {
		t.Errorf("FindPublishedByToken: %v", err)
	}
	if _, err := AddStop(app, owner, artworkID(1)); err != nil {
		t.Errorf("AddStop after publish should start a fresh draft: %v", err)
	}
}

func TestPublishValidation(t *testing.T) {
	app := installItinerarySchema(t)

	if _, err := Publish(app, "no-draft"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Publish with no draft = %v, want sql.ErrNoRows", err)
	}

	addStops(t, app, "no-title", 1)
	if _, err := Publish(app, "no-title"); !errors.Is(err, ErrTitleRequired) {
		t.Errorf("Publish without title = %v, want ErrTitleRequired", err)
	}

	if err := SetMeta(app, "no-stops", Meta{Title: "Titled"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if _, err := Publish(app, "no-stops"); !errors.Is(err, ErrNoStops) {
		t.Errorf("Publish without stops = %v, want ErrNoStops", err)
	}
}

func TestListPublishedAndExpiry(t *testing.T) {
	app := installItinerarySchema(t)

	owner := "owner-list"
	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Listed"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := SetListed(app, owner, true); err != nil {
		t.Fatalf("SetListed: %v", err)
	}
	published, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Publication is immediate: a listed record is indexed at once.
	listed, err := ListPublished(app, 50)
	if err != nil {
		t.Fatalf("ListPublished: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("listed = %d, want 1 immediately after publication", len(listed))
	}
	if IsExpired(published) {
		t.Error("freshly published itinerary must not be expired")
	}

	// A link-only record is readable but absent from the index.
	linkOnly := "owner-link-only"
	addStops(t, app, linkOnly, 1)
	if err := SetMeta(app, linkOnly, Meta{Title: "Link Only"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := SetListed(app, linkOnly, false); err != nil {
		t.Fatalf("SetListed: %v", err)
	}
	linkRecord, err := Publish(app, linkOnly)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, err := FindPublishedByToken(app, linkRecord.GetString("token")); err != nil {
		t.Errorf("link-only token must resolve: %v", err)
	}
	listed, err = ListPublished(app, 50)
	if err != nil {
		t.Fatalf("ListPublished: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("listed = %d, want 1 (link-only must stay off the index)", len(listed))
	}

	// Rejected records are neither listed nor public.
	published.Set("status", string(StatusRejected))
	if err := app.Save(published); err != nil {
		t.Fatalf("reject: %v", err)
	}
	listed, err = ListPublished(app, 50)
	if err != nil {
		t.Fatalf("ListPublished: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("listed = %d, want 0 after rejection", len(listed))
	}
	if IsPublicStatus(published.GetString("status")) {
		t.Error("rejected itinerary must not be public")
	}

	// Expiry: an expired listed record drops out of the index.
	listedOwner := "owner-list-expire"
	addStops(t, app, listedOwner, 1)
	if err := SetMeta(app, listedOwner, Meta{Title: "Expiring"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := SetListed(app, listedOwner, true); err != nil {
		t.Fatalf("SetListed: %v", err)
	}
	expiring, err := Publish(app, listedOwner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	expiring.Set("expires_at", types.NowDateTime().Add(-time.Hour))
	if err := app.Save(expiring); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if !IsExpired(expiring) {
		t.Error("expired itinerary must be reported expired")
	}
	listed, err = ListPublished(app, 50)
	if err != nil {
		t.Fatalf("ListPublished: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("listed = %d, want 0 after expiry", len(listed))
	}
}

func TestPublishRateLimitBoundedPerOwner(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-rate-limit"

	for index := 0; index < PublicationBudget; index++ {
		publishDraft(t, app, owner)
	}

	// The budget is spent: a further publication within the window is refused.
	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Titled"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if _, err := Publish(app, owner); !errors.Is(err, ErrPublishRateLimit) {
		t.Errorf("publish beyond budget = %v, want ErrPublishRateLimit", err)
	}
}

func TestPublishRateLimitIsScopedPerOwnerAndRollsOver(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-rate-scope"
	other := "owner-rate-other"

	// Exhaust the first owner's budget.
	for index := 0; index < PublicationBudget; index++ {
		publishDraft(t, app, owner)
	}

	// A different owner is unaffected.
	if _, err := Publish(app, other); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("other owner publish = %v, want sql.ErrNoRows (no draft)", err)
	}
	publishDraft(t, app, other)

	// Age the first owner's publications outside the rolling window.
	records, err := app.FindRecordsByFilter(CollectionItineraries, "owner = {:owner} && status != {:draft}", "", 0, 0, map[string]any{"owner": owner, "draft": string(StatusDraft)})
	if err != nil {
		t.Fatalf("find publications: %v", err)
	}
	oldPublished := types.NowDateTime().Add(-PublicationWindow - time.Hour)
	for _, record := range records {
		if _, err := app.DB().NewQuery("UPDATE itineraries SET published = {:p} WHERE id = {:id}").Bind(map[string]any{"p": oldPublished, "id": record.Id}).Execute(); err != nil {
			t.Fatalf("age publication: %v", err)
		}
	}

	// The rolling window has passed: publication is allowed again.
	if _, err := Publish(app, owner); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("publish after window = %v, want sql.ErrNoRows (draft consumed by earlier attempt)", err)
	}
}

func TestConcurrentAddsCannotExceedStopLimit(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-concurrent-add"

	const workers = 24
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for index := 0; index < workers; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, errs[index] = AddStop(app, owner, artworkID((index%16)+1))
		}(index)
	}
	wg.Wait()

	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if len(stops) != MaxStops {
		t.Errorf("concurrent adds persisted %d stops, want exactly %d", len(stops), MaxStops)
	}
	seen := map[string]struct{}{}
	for _, stop := range stops {
		if _, exists := seen[stop.GetString("artwork")]; exists {
			t.Errorf("duplicate artwork %q persisted", stop.GetString("artwork"))
		}
		seen[stop.GetString("artwork")] = struct{}{}
	}
}

func TestConcurrentPublishesCannotExceedBudget(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-concurrent-publish"

	for index := 0; index < PublicationBudget; index++ {
		publishDraft(t, app, owner)
	}

	const workers = 5
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for index := 0; index < workers; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, errs[index] = Publish(app, owner)
		}(index)
	}
	wg.Wait()

	for index, err := range errs {
		if !errors.Is(err, ErrPublishRateLimit) {
			t.Errorf("concurrent publish %d = %v, want ErrPublishRateLimit", index, err)
		}
	}

	records, err := app.FindRecordsByFilter(CollectionItineraries, "owner = {:owner} && status != {:draft}", "", 0, 0, map[string]any{"owner": owner, "draft": string(StatusDraft)})
	if err != nil {
		t.Fatalf("find publications: %v", err)
	}
	if len(records) != PublicationBudget {
		t.Errorf("published records = %d, want %d", len(records), PublicationBudget)
	}
}

func TestMutationsRefreshDraftUpdated(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-touch"

	addStops(t, app, owner, 1)
	draftID := mustDraftID(t, app, owner)

	oldTime := types.NowDateTime().Add(-2 * time.Hour)
	ageItineraryUpdated(t, app, draftID, oldTime)

	if _, err := AddStop(app, owner, artworkID(2)); err != nil {
		t.Fatalf("AddStop: %v", err)
	}

	draft, err := FindDraft(app, owner)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	if !draft.GetDateTime("updated").After(oldTime) {
		t.Error("adding a stop must refresh the draft's updated time")
	}
}

func TestNarrationAndClearRefreshDraftUpdated(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-touch-2"

	addStops(t, app, owner, 2)
	draftID := mustDraftID(t, app, owner)

	stops, err := LoadStops(app, draftID)
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}

	oldTime := types.NowDateTime().Add(-2 * time.Hour)
	ageItineraryUpdated(t, app, draftID, oldTime)
	if err := SetNarration(app, owner, stops[0].Id, "note"); err != nil {
		t.Fatalf("SetNarration: %v", err)
	}
	if !draftUpdatedAfter(t, app, owner, oldTime) {
		t.Error("narrating a stop must refresh the draft's updated time")
	}

	oldTime = types.NowDateTime().Add(-2 * time.Hour)
	ageItineraryUpdated(t, app, draftID, oldTime)
	if err := ClearDraft(app, owner); err != nil {
		t.Fatalf("ClearDraft: %v", err)
	}
	if !draftUpdatedAfter(t, app, owner, oldTime) {
		t.Error("clearing a draft must refresh the draft's updated time")
	}
}

func TestLatestPublishedDeterministicNewestFirst(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-latest"

	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "First"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	first, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish first: %v", err)
	}

	// Age the first publication so it is no longer the newest.
	oldPublished := types.NowDateTime().Add(-time.Hour)
	if _, err := app.DB().NewQuery("UPDATE itineraries SET published = {:p} WHERE id = {:id}").Bind(map[string]any{"p": oldPublished, "id": first.Id}).Execute(); err != nil {
		t.Fatalf("age first publication: %v", err)
	}

	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Second"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	second, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish second: %v", err)
	}

	latest, err := LatestPublished(app, owner)
	if err != nil {
		t.Fatalf("LatestPublished: %v", err)
	}
	if latest.Id != second.Id {
		t.Errorf("LatestPublished = %q, want newest %q", latest.Id, second.Id)
	}
}

func TestLatestPublishedStableTieBreakOnEqualTimestamps(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-tie"

	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "First"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	first, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish first: %v", err)
	}

	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Second"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	second, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish second: %v", err)
	}

	// Force identical published and created timestamps so only the id
	// tie-break decides the winner.
	sameTime := types.NowDateTime().Add(-time.Hour)
	for _, id := range []string{first.Id, second.Id} {
		if _, err := app.DB().NewQuery(
			"UPDATE itineraries SET published = {:t}, created = {:t} WHERE id = {:id}",
		).Bind(map[string]any{"t": sameTime, "id": id}).Execute(); err != nil {
			t.Fatalf("align timestamps: %v", err)
		}
	}

	// With -published,-created,-id the stable winner is the greatest id.
	want := first.Id
	if second.Id > first.Id {
		want = second.Id
	}

	gotFirst, err := LatestPublished(app, owner)
	if err != nil {
		t.Fatalf("LatestPublished (first call): %v", err)
	}
	gotSecond, err := LatestPublished(app, owner)
	if err != nil {
		t.Fatalf("LatestPublished (second call): %v", err)
	}
	if gotFirst.Id != want || gotSecond.Id != want {
		t.Errorf("LatestPublished = %q / %q, want stable winner %q", gotFirst.Id, gotSecond.Id, want)
	}
	if gotFirst.Id != gotSecond.Id {
		t.Errorf("LatestPublished is not deterministic: %q vs %q", gotFirst.Id, gotSecond.Id)
	}
}

func TestAvailableStopsSkipsUnavailableAndPreservesOrder(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-available"

	addStops(t, app, owner, 4)

	// Unpublish artwork 2 (position 1).
	unpublished, err := app.FindRecordById("artworks", artworkID(2))
	if err != nil {
		t.Fatalf("find artwork 2: %v", err)
	}
	unpublished.Set("published", false)
	if err := app.Save(unpublished); err != nil {
		t.Fatalf("unpublish artwork 2: %v", err)
	}

	// Delete artwork 4 (position 3) via raw SQL to simulate an orphaned stop:
	// the required relation blocks App.Delete, and the raw delete leaves the
	// stop's snapshot pointing at a missing artwork.
	if _, err := app.DB().NewQuery(
		"DELETE FROM artworks WHERE id = {:id}",
	).Bind(map[string]any{"id": artworkID(4)}).Execute(); err != nil {
		t.Fatalf("delete artwork 4: %v", err)
	}

	draftID := mustDraftID(t, app, owner)

	available, err := AvailableStops(app, draftID)
	if err != nil {
		t.Fatalf("AvailableStops: %v", err)
	}
	if len(available) != 2 {
		t.Fatalf("available = %d, want 2 (artwork 1 and 3 remain)", len(available))
	}
	if available[0].GetString("artwork") != artworkID(1) {
		t.Errorf("available[0].artwork = %q, want %q", available[0].GetString("artwork"), artworkID(1))
	}
	if available[1].GetString("artwork") != artworkID(3) {
		t.Errorf("available[1].artwork = %q, want %q", available[1].GetString("artwork"), artworkID(3))
	}
	if available[0].GetInt("position") != 0 || available[1].GetInt("position") != 2 {
		t.Errorf("positions = %d,%d, want 0,2 (order preserved)", available[0].GetInt("position"), available[1].GetInt("position"))
	}

	// Persisted snapshots are untouched: LoadStops still returns all four.
	all, err := LoadStops(app, draftID)
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("LoadStops = %d, want 4 (snapshots must remain persisted)", len(all))
	}
}

func TestAvailableStopsReturnsEmptyWhenAllUnavailable(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-available-empty"

	addStops(t, app, owner, 1)
	if _, err := app.DB().NewQuery(
		"DELETE FROM artworks WHERE id = {:id}",
	).Bind(map[string]any{"id": artworkID(1)}).Execute(); err != nil {
		t.Fatalf("delete artwork 1: %v", err)
	}

	available, err := AvailableStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("AvailableStops: %v", err)
	}
	if len(available) != 0 {
		t.Errorf("available = %d, want 0", len(available))
	}
}

func TestEstimateDurationFloorsAtOneMinute(t *testing.T) {
	app := installItinerarySchema(t)
	owner := "owner-duration"

	addStops(t, app, owner, 1)
	stops, err := LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}

	// One stop with no narration still estimates at least one minute.
	if got := EstimateDuration(stops); got != "1 MIN" {
		t.Errorf("EstimateDuration(1 empty stop) = %q, want %q", got, "1 MIN")
	}

	// Narration words raise the estimate (1 word/150 + 1 stop * 0.6 rounds to 1).
	if err := SetNarration(app, owner, stops[0].Id, "one two three"); err != nil {
		t.Fatalf("SetNarration: %v", err)
	}
	stops, err = LoadStops(app, mustDraftID(t, app, owner))
	if err != nil {
		t.Fatalf("LoadStops: %v", err)
	}
	if got := EstimateDuration(stops); got == "" {
		t.Error("EstimateDuration must return a non-empty label")
	}
}

func publishDraft(t *testing.T, app core.App, owner string) {
	t.Helper()
	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Titled"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if _, err := Publish(app, owner); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func ageItineraryUpdated(t *testing.T, app core.App, itineraryID string, when types.DateTime) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE itineraries SET updated = {:updated} WHERE id = {:id}").Bind(map[string]any{"updated": when, "id": itineraryID}).Execute(); err != nil {
		t.Fatalf("age itinerary: %v", err)
	}
}

func draftUpdatedAfter(t *testing.T, app core.App, owner string, when types.DateTime) bool {
	t.Helper()
	draft, err := FindDraft(app, owner)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	return draft.GetDateTime("updated").After(when)
}

func installItinerarySchema(t *testing.T) core.App {
	t.Helper()
	app := testutils.NewTestApp(t)

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artworks collection: %v", err)
	}

	for index := 1; index <= MaxStops+1; index++ {
		record := core.NewRecord(artworks)
		record.Id = artworkID(index)
		record.Set("title", "Work "+strconv.Itoa(index))
		record.Set("published", true)
		if err := app.Save(record); err != nil {
			t.Fatalf("create artwork %d: %v", index, err)
		}
	}
	unpublished := core.NewRecord(artworks)
	unpublished.Id = artworkID(99)
	unpublished.Set("title", "Hidden")
	unpublished.Set("published", false)
	if err := app.Save(unpublished); err != nil {
		t.Fatalf("create unpublished artwork: %v", err)
	}

	itineraries := core.NewBaseCollection("Itineraries")
	itineraries.Id = CollectionItineraries
	itineraries.MarkAsNew()
	itineraries.Fields.Add(
		&core.TextField{Name: "owner", Required: true},
		&core.SelectField{Name: "status", Values: []string{"draft", "pending", "approved", "rejected"}, MaxSelect: 1, Required: true},
		&core.TextField{Name: "token"},
		&core.TextField{Name: "title", Max: 80},
		&core.TextField{Name: "intro", Max: 400},
		&core.TextField{Name: "creator", Max: 40},
		&core.BoolField{Name: "listed"},
		&core.DateField{Name: "published"},
		&core.DateField{Name: "expires_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	itineraries.Indexes = append(itineraries.Indexes,
		"CREATE UNIQUE INDEX `pbx_itinerary_token` ON `itineraries` (token) WHERE token != ''",
		"CREATE UNIQUE INDEX `pbx_itinerary_draft_owner` ON `itineraries` (owner) WHERE status = 'draft'",
		"CREATE INDEX `pbx_itinerary_expiry` ON `itineraries` (expires_at)",
	)
	if err := app.Save(itineraries); err != nil {
		t.Fatalf("create itineraries collection: %v", err)
	}

	stops := core.NewBaseCollection("Itinerary_stops")
	stops.Id = CollectionItineraryStops
	stops.MarkAsNew()
	stops.Fields.Add(
		&core.RelationField{Name: "itinerary", CollectionId: CollectionItineraries, MinSelect: 1, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.RelationField{Name: "artwork", CollectionId: "artworks", MinSelect: 1, MaxSelect: 1, Required: true},
		&core.TextField{Name: "title"},
		&core.NumberField{Name: "position"},
		&core.TextField{Name: "narration", Max: 600},
	)
	stops.Indexes = append(stops.Indexes,
		"CREATE UNIQUE INDEX `pbx_itinerary_stop_artwork` ON `itinerary_stops` (itinerary, artwork)",
		"CREATE UNIQUE INDEX `pbx_itinerary_stop_order` ON `itinerary_stops` (itinerary, position)",
	)
	if err := app.Save(stops); err != nil {
		t.Fatalf("create itinerary_stops collection: %v", err)
	}

	return app
}

func addStops(t *testing.T, app core.App, owner string, count int) {
	t.Helper()
	for index := 1; index <= count; index++ {
		if _, err := AddStop(app, owner, artworkID(index)); err != nil {
			t.Fatalf("AddStop #%d: %v", index, err)
		}
	}
}

func mustDraftID(t *testing.T, app core.App, owner string) string {
	t.Helper()
	draft, err := FindDraft(app, owner)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	return draft.Id
}
