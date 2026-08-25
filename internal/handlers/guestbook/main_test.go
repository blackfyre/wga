package guestbook

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/hooks"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestBuildFiltersEnforcesApprovedNewestFirstArchive(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/guestbook?q=%20%20Jane%20%20&year=all&show=500&sort=oldest", nil)
	got := buildFilters(&core.RequestEvent{Event: router.Event{Request: request}}, "2026")
	want := filters{Query: "Jane", Year: "all", Show: 100}
	if got != want {
		t.Fatalf("filters = %#v, want %#v", got, want)
	}

	filter, params := got.buildFilter()
	if !strings.HasPrefix(filter, "moderation_state = {:approved}") {
		t.Fatalf("filter does not enforce approval: %q", filter)
	}
	if params["approved"] != guestbookStateApproved || params["query"] != "Jane" {
		t.Fatalf("filter params = %#v", params)
	}
}

func TestBuildFiltersRejectsInvalidYearAndBoundsShow(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/guestbook?year=1997&show=2", nil)
	got := buildFilters(&core.RequestEvent{Event: router.Event{Request: request}}, "2026")
	want := filters{Year: "2026", Show: guestbookPageSize}
	if got != want {
		t.Fatalf("filters = %#v, want %#v", got, want)
	}
}

func TestPrepareSubmissionValidationAndHoneypot(t *testing.T) {
	prepared, validationErrors, err := prepareSubmission(submissionInput{
		Name:     "  Jane Doe  ",
		Location: "  Delft  ",
		Message:  "  A thoughtful note.  ",
	})
	if err != nil || validationErrors.any() {
		t.Fatalf("valid submission: errors=%#v err=%v", validationErrors, err)
	}
	if prepared.Name != "Jane Doe" || prepared.Location != "Delft" || prepared.Message != "A thoughtful note." {
		t.Fatalf("prepared submission = %#v", prepared)
	}

	_, validationErrors, err = prepareSubmission(submissionInput{})
	if err != nil || validationErrors.Name == "" || validationErrors.Location == "" || validationErrors.Message == "" {
		t.Fatalf("required validation: errors=%#v err=%v", validationErrors, err)
	}

	_, _, err = prepareSubmission(submissionInput{HoneyPotName: "bot"})
	if err == nil {
		t.Fatal("honeypot submission accepted")
	}
}

func TestSubmissionRateLimiterIsBoundedAndResets(t *testing.T) {
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	limiter := newSubmissionRateLimiter(func() time.Time { return now })
	for i := 0; i < guestbookSubmissionLimit; i++ {
		if !limiter.allow("192.0.2.10") {
			t.Fatalf("submission %d unexpectedly limited", i+1)
		}
	}
	if limiter.allow("192.0.2.10") {
		t.Fatal("submission beyond limit accepted")
	}
	if strings.Contains(strings.Join(mapKeys(limiter.windows), " "), "192.0.2.10") {
		t.Fatal("rate limiter retained raw remote address")
	}

	for i := 0; i < guestbookRateLimitEntries+10; i++ {
		limiter.allow("198.51.100." + string(rune(i)))
	}
	if len(limiter.windows) > guestbookRateLimitEntries {
		t.Fatalf("rate limiter windows = %d", len(limiter.windows))
	}

	now = now.Add(guestbookRateLimitWindow)
	if !limiter.allow("192.0.2.10") {
		t.Fatal("rate limiter did not reset after its window")
	}
}

func TestRepositoryModerationSearchYearOrderAndMetadata(t *testing.T) {
	app := newGuestbookTestApp(t)
	saveGuestbookEntry(t, app, guestbookStateApproved, "2025-01-02 00:00:00.000Z", "Jane", "Delft", "Blue chapel", "")
	saveGuestbookEntry(t, app, guestbookStateApproved, "2025-06-02 00:00:00.000Z", "Alex", "London", "Blue altarpiece", "")
	saveGuestbookEntry(t, app, guestbookStateUnreviewed, "2025-07-02 00:00:00.000Z", "Private", "Paris", "Blue private note", "2026-01-01 00:00:00.000Z")
	saveGuestbookEntry(t, app, guestbookStateApproved, "2024-03-02 00:00:00.000Z", "Older", "Rome", "Red fresco", "")

	selected := filters{Query: "Blue", Year: "2025", Show: 10}
	repo := repository{app: app}
	total, scopeTotal, err := repo.publicCounts(selected)
	if err != nil {
		t.Fatalf("count public entries: %v", err)
	}
	if total != 3 || scopeTotal != 2 {
		t.Fatalf("counts = total %d scope %d", total, scopeTotal)
	}
	entries, err := repo.publicEntries(selected, 10)
	if err != nil {
		t.Fatalf("list public entries: %v", err)
	}
	if len(entries) != 2 || entries[0].Name != "Alex" || entries[1].Name != "Jane" {
		t.Fatalf("newest-first entries = %#v", entries)
	}
	if entries[0].Location != "London" || entries[0].Created != "2025-06-02" || entries[0].Email != "" {
		t.Fatalf("public metadata/privacy = %#v", entries[0])
	}

	assertYearOptions(t, app, []string{"2026", "2025", "2024"})
}

func TestApprovedYearsDistinctDescendingExcludesUnapproved(t *testing.T) {
	app := newGuestbookTestApp(t)
	saveGuestbookEntry(t, app, guestbookStateApproved, "2025-01-02 00:00:00.000Z", "Jane", "Delft", "First", "")
	saveGuestbookEntry(t, app, guestbookStateApproved, "2025-06-02 00:00:00.000Z", "Alex", "London", "Same year", "")
	saveGuestbookEntry(t, app, guestbookStateApproved, "2024-03-02 00:00:00.000Z", "Older", "Rome", "Older year", "")
	saveGuestbookEntry(t, app, guestbookStateUnreviewed, "2023-01-01 00:00:00.000Z", "Private", "Paris", "Hidden", "2026-01-01 00:00:00.000Z")
	saveGuestbookEntry(t, app, guestbookStateRejected, "2022-01-01 00:00:00.000Z", "Rejected", "Rome", "Hidden", "")
	saveGuestbookEntry(t, app, guestbookStateApproved, "", "Empty", "Nowhere", "No created", "")

	got, err := (repository{app: app}).approvedYears()
	if err != nil {
		t.Fatalf("approved years: %v", err)
	}
	want := []string{"2025", "2024"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("approved years = %v, want %v", got, want)
	}
}

func TestRepositoryCreatesPrivateUnreviewedEntryWithoutEmail(t *testing.T) {
	app := newGuestbookTestApp(t)
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	input := submissionInput{Name: "Jane", Location: "Delft", Message: "Private note"}
	if err := (repository{app: app}).createUnreviewed(input, now); err != nil {
		t.Fatalf("create unreviewed entry: %v", err)
	}
	records, err := app.FindRecordsByFilter(constants.CollectionGuestbook, "", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("saved records = %d err=%v", len(records), err)
	}
	record := records[0]
	if record.GetString("moderation_state") != guestbookStateUnreviewed {
		t.Fatalf("moderation state = %q", record.GetString("moderation_state"))
	}
	if record.GetString("email") != "" {
		t.Fatalf("email was persisted: %q", record.GetString("email"))
	}
	wantRetention := formatPocketBaseTime(now.Add(guestbookPrivateRetention))
	if record.GetString("retention_until") != wantRetention {
		t.Fatalf("retention_until = %q, want %q", record.GetString("retention_until"), wantRetention)
	}
}

func TestPurgeExpiredPrivateEntriesPreservesApprovedAndFutureRecords(t *testing.T) {
	app := newGuestbookTestApp(t)
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	expired := saveGuestbookEntry(t, app, guestbookStateUnreviewed, "2026-01-01 00:00:00.000Z", "Expired", "", "private", formatPocketBaseTime(now.Add(-time.Minute)))
	approved := saveGuestbookEntry(t, app, guestbookStateApproved, "2026-01-02 00:00:00.000Z", "Approved", "", "public", formatPocketBaseTime(now.Add(-time.Minute)))
	future := saveGuestbookEntry(t, app, guestbookStateRejected, "2026-01-03 00:00:00.000Z", "Future", "", "private", formatPocketBaseTime(now.Add(time.Minute)))

	deleted, err := PurgeExpiredPrivateEntries(app, now)
	if err != nil {
		t.Fatalf("purge expired private entries: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := app.FindRecordById(constants.CollectionGuestbook, expired.Id); err == nil {
		t.Fatal("expired private entry remains")
	}
	for _, record := range []*core.Record{approved, future} {
		if _, err := app.FindRecordById(constants.CollectionGuestbook, record.Id); err != nil {
			t.Fatalf("preserved entry %q missing: %v", record.Id, err)
		}
	}
}

func TestWithdrawalRedactsPersonalFieldsAtPersistenceBoundary(t *testing.T) {
	app := newGuestbookTestApp(t)
	record := saveGuestbookEntry(t, app, guestbookStateApproved, "2026-01-01 00:00:00.000Z", "Sensitive Name", "Sensitive Location", "Sensitive Message", "")
	record.Set("email", "sensitive@example.test")
	if err := app.Save(record); err != nil {
		t.Fatalf("save legacy email: %v", err)
	}

	// Simulate the superuser withdrawal: reload the persisted state, move it to
	// the rejected outcome, and save. The redaction hook must clear the visitor
	// fields as part of that same update.
	loaded, err := app.FindRecordById(constants.CollectionGuestbook, record.Id)
	if err != nil {
		t.Fatalf("reload approved entry: %v", err)
	}
	loaded.Set("moderation_state", guestbookStateRejected)
	if err := app.Save(loaded); err != nil {
		t.Fatalf("withdraw approved entry: %v", err)
	}

	redacted, err := app.FindRecordById(constants.CollectionGuestbook, record.Id)
	if err != nil {
		t.Fatalf("find withdrawn entry: %v", err)
	}
	for _, field := range []string{"name", "email", "location", "message"} {
		if value := redacted.GetString(field); value != "" {
			t.Fatalf("withdrawn %s = %q, want empty", field, value)
		}
	}
	if redacted.GetString("moderation_state") != guestbookStateRejected {
		t.Fatalf("withdrawal outcome = %q, want %q", redacted.GetString("moderation_state"), guestbookStateRejected)
	}
	entries, err := (repository{app: app}).publicEntries(filters{Year: "all", Show: 10}, 10)
	if err != nil {
		t.Fatalf("list entries after withdrawal: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("withdrawn entry remains public: %#v", entries)
	}
}

func TestWithdrawalRedactionIsIdempotentAndSparesStillApprovedEdits(t *testing.T) {
	app := newGuestbookTestApp(t)

	// Withdraw an approved entry, then re-save the withdrawn husk. The second
	// save must not error and must not restore the cleared visitor fields.
	approved := saveGuestbookEntry(t, app, guestbookStateApproved, "2026-01-01 00:00:00.000Z", "Sensitive Name", "Delft", "Sensitive Message", "")
	withdraw, err := app.FindRecordById(constants.CollectionGuestbook, approved.Id)
	if err != nil {
		t.Fatalf("reload approved entry: %v", err)
	}
	withdraw.Set("moderation_state", guestbookStateRejected)
	if err := app.Save(withdraw); err != nil {
		t.Fatalf("withdraw approved entry: %v", err)
	}

	resave, err := app.FindRecordById(constants.CollectionGuestbook, approved.Id)
	if err != nil {
		t.Fatalf("reload withdrawn entry: %v", err)
	}
	if err := app.Save(resave); err != nil {
		t.Fatalf("resave withdrawn entry: %v", err)
	}
	after, err := app.FindRecordById(constants.CollectionGuestbook, approved.Id)
	if err != nil {
		t.Fatalf("find resaved entry: %v", err)
	}
	for _, field := range []string{"name", "email", "location", "message"} {
		if value := after.GetString(field); value != "" {
			t.Fatalf("resaved %s = %q, want empty (idempotent redaction)", field, value)
		}
	}

	// Editing an entry that stays approved must preserve the corrected fields.
	approvedEdit := saveGuestbookEntry(t, app, guestbookStateApproved, "2026-01-02 00:00:00.000Z", "Jnae", "Delft", "Typo note", "")
	stillApproved, err := app.FindRecordById(constants.CollectionGuestbook, approvedEdit.Id)
	if err != nil {
		t.Fatalf("reload approved entry: %v", err)
	}
	stillApproved.Set("name", "Jane")
	if err := app.Save(stillApproved); err != nil {
		t.Fatalf("edit approved entry: %v", err)
	}
	edited, err := app.FindRecordById(constants.CollectionGuestbook, approvedEdit.Id)
	if err != nil {
		t.Fatalf("find edited approved entry: %v", err)
	}
	if edited.GetString("name") != "Jane" {
		t.Fatalf("approved name = %q, want corrected Jane", edited.GetString("name"))
	}
	if edited.GetString("message") != "Typo note" {
		t.Fatalf("approved message = %q, want preserved", edited.GetString("message"))
	}
}

func TestStoreEntryLogsNeverContainNameLocationOrMessage(t *testing.T) {
	app := newGuestbookTestApp(t)
	batch, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("logger handler = %T", app.Logger().Handler())
	}
	batch.SetLevel(-4)
	captured := testutils.CaptureLogs(app)
	form := url.Values{
		"sender_name": {"Sensitive Name"},
		"location":    {"Sensitive Location"},
		"message":     {"Sensitive Message"},
	}
	request := httptest.NewRequest(http.MethodPost, "/guestbook/add", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: httptest.NewRecorder()}}
	logging.SetRequestID(event, "request-guestbook")
	if err := storeEntryHandler(app, event, newSubmissionRateLimiter(time.Now), requesttrust.New(requesttrust.SourceDirect)); err != nil {
		t.Fatalf("store entry: %v", err)
	}
	testutils.FlushLogs(t, app)
	logged := strings.Join(anyStrings(testutils.LogData(captured())), " ")
	for _, sensitive := range []string{"Sensitive Name", "Sensitive Location", "Sensitive Message"} {
		if strings.Contains(logged, sensitive) {
			t.Fatalf("logs contain %q: %s", sensitive, logged)
		}
	}
}

func TestSameOriginRequest(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		origin  string
		referer string
		want    bool
	}{
		{name: "same-origin https origin", host: "example.com", origin: "https://example.com/guestbook", want: true},
		{name: "same-origin http origin", host: "example.com", origin: "http://example.com/guestbook", want: true},
		{name: "origin default port matches bare host", host: "example.com", origin: "https://example.com:443/guestbook", want: true},
		{name: "bare host matches origin default port", host: "example.com:443", origin: "https://example.com/guestbook", want: true},
		{name: "non-default port matches host", host: "example.com:8443", origin: "https://example.com:8443/guestbook", want: true},
		{name: "cross-origin host", host: "example.com", origin: "https://evil.example.net/guestbook", want: false},
		{name: "cross-origin port", host: "example.com:443", origin: "https://example.com:8443/guestbook", want: false},
		{name: "cross-origin default port mismatch", host: "example.com", origin: "http://example.com:443/guestbook", want: false},
		{name: "non-http origin scheme", host: "example.com", origin: "ftp://example.com/guestbook", want: false},
		{name: "null origin", host: "example.com", origin: "null", want: false},
		{name: "malformed origin", host: "example.com", origin: "https://[::1", want: false},
		{name: "same-origin referer", host: "example.com", referer: "https://example.com/guestbook", want: true},
		{name: "cross-origin referer", host: "example.com", referer: "https://evil.example.net/guestbook", want: false},
		{name: "malformed referer", host: "example.com", referer: "https://[::1", want: false},
		{name: "no headers allowed", host: "example.com", want: true},
		{name: "origin takes precedence over referer", host: "example.com", origin: "https://example.com/guestbook", referer: "https://evil.example.net/", want: true},
		{name: "empty host with origin rejects", host: "", origin: "https://example.com/guestbook", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/guestbook/add", nil)
			request.Host = tt.host
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				request.Header.Set("Referer", tt.referer)
			}
			if got := sameOriginRequest(request); got != tt.want {
				t.Fatalf("sameOriginRequest(host=%q origin=%q referer=%q) = %v, want %v", tt.host, tt.origin, tt.referer, got, tt.want)
			}
		})
	}
}

func TestStoreEntryRejectsCrossOriginBeforePersistence(t *testing.T) {
	app := newGuestbookTestApp(t)
	limiter := newSubmissionRateLimiter(time.Now)
	form := url.Values{
		"sender_name": {"Jane"},
		"location":    {"Delft"},
		"message":     {"Hello"},
	}
	request := httptest.NewRequest(http.MethodPost, "/guestbook/add", strings.NewReader(form.Encode()))
	request.Host = "example.com"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://evil.example.net")
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: recorder}}
	logging.SetRequestID(event, "request-cross-origin")

	if err := storeEntryHandler(app, event, limiter, requesttrust.New(requesttrust.SourceDirect)); err != nil {
		t.Fatalf("store entry: %v", err)
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}

	records, err := app.FindRecordsByFilter(constants.CollectionGuestbook, "", "", 0, 0)
	if err != nil || len(records) != 0 {
		t.Fatalf("records persisted = %d err=%v, want 0", len(records), err)
	}
	if len(limiter.windows) != 0 {
		t.Fatalf("rate limiter consumed %d windows, want 0", len(limiter.windows))
	}
}

func TestStoreEntryAcceptsSameOriginOrigin(t *testing.T) {
	app := newGuestbookTestApp(t)
	limiter := newSubmissionRateLimiter(time.Now)
	form := url.Values{
		"sender_name": {"Jane"},
		"location":    {"Delft"},
		"message":     {"Hello"},
	}
	request := httptest.NewRequest(http.MethodPost, "/guestbook/add", strings.NewReader(form.Encode()))
	request.Host = "example.com"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://example.com/guestbook")
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: recorder}}
	logging.SetRequestID(event, "request-same-origin")

	if err := storeEntryHandler(app, event, limiter, requesttrust.New(requesttrust.SourceDirect)); err != nil {
		t.Fatalf("store entry: %v", err)
	}
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	records, err := app.FindRecordsByFilter(constants.CollectionGuestbook, "", "", 0, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("records persisted = %d err=%v, want 1", len(records), err)
	}
}

func TestStoreEntryFailsClosedWithoutTrustedIdentity(t *testing.T) {
	app := newGuestbookTestApp(t)
	limiter := newSubmissionRateLimiter(time.Now)
	resolver := requesttrust.Resolver(func(*http.Request) (string, bool) {
		return "", false
	})
	form := url.Values{
		"sender_name": {"Jane"},
		"location":    {"Delft"},
		"message":     {"Hello"},
	}
	request := httptest.NewRequest(http.MethodPost, "/guestbook/add", strings.NewReader(form.Encode()))
	request.Host = "example.com"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://example.com/guestbook")
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: recorder}}
	logging.SetRequestID(event, "request-missing-identity")

	if err := storeEntryHandler(app, event, limiter, resolver); err != nil {
		t.Fatalf("store entry: %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (fail closed)", recorder.Code)
	}
	records, err := app.FindRecordsByFilter(constants.CollectionGuestbook, "", "", 0, 0)
	if err != nil || len(records) != 0 {
		t.Fatalf("records persisted = %d err=%v, want 0", len(records), err)
	}
	if len(limiter.windows) != 0 {
		t.Fatalf("rate limiter consumed %d windows, want 0", len(limiter.windows))
	}
}

func TestGuestbookRouteSelectsTargetAwareResponse(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})
	createGuestbookCollection(t, app)
	saveGuestbookEntry(t, app, guestbookStateApproved, "2026-06-02 00:00:00.000Z", "Jane", "Delft", "A public note.", "")
	RegisterHandlers(app, requesttrust.New(requesttrust.SourceDirect))

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		full := httptest.NewRecorder()
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/guestbook", nil))
		if full.Code != http.StatusOK {
			t.Errorf("full status = %d, want %d", full.Code, http.StatusOK)
		}
		if !strings.Contains(full.Body.String(), "<html") || !strings.Contains(full.Body.String(), "Jane") {
			t.Error("full response should render the full document with approved entries")
		}
		if got := strings.Count(full.Body.String(), `id="mc-area"`); got != 1 {
			t.Errorf("full response rendered %d #mc-area elements, want exactly 1", got)
		}

		for _, target := range []string{"mc-area", "#mc-area"} {
			shell := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/guestbook", nil)
			request.Header.Set("HX-Request", "true")
			request.Header.Set("HX-Target", target)
			mux.ServeHTTP(shell, request)
			if shell.Code != http.StatusOK {
				t.Errorf("shell(%s) status = %d, want %d", target, shell.Code, http.StatusOK)
			}
			if !strings.Contains(shell.Body.String(), "<html") {
				t.Errorf("shell(%s) must render the full document", target)
			}
			if got := strings.Count(shell.Body.String(), `id="mc-area"`); got != 1 {
				t.Errorf("shell(%s) rendered %d #mc-area elements, want exactly 1", target, got)
			}
		}

		local := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/guestbook", nil)
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", "guestbook")
		mux.ServeHTTP(local, request)
		if local.Code != http.StatusOK {
			t.Errorf("local status = %d, want %d", local.Code, http.StatusOK)
		}
		if strings.Contains(local.Body.String(), "<html") {
			t.Error("feature-local response should not render the full document")
		}
		if got := strings.Count(local.Body.String(), `id="guestbook"`); got != 1 {
			t.Errorf("feature-local response rendered %d #guestbook elements, want exactly 1", got)
		}
		if strings.Contains(local.Body.String(), `id="mc-area"`) {
			t.Error("feature-local response must not carry #mc-area")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func newGuestbookTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app := testutils.NewTestApp(t)
	createGuestbookCollection(t, app)
	hooks.RegisterHooks(app)
	RegisterRedactionHook(app)
	return app
}

func createGuestbookCollection(t *testing.T, app core.App) {
	t.Helper()
	collection := core.NewBaseCollection("Guestbook")
	collection.Id = constants.CollectionGuestbook
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.EmailField{Name: "email"},
		&core.TextField{Name: "location"},
		&core.TextField{Name: "message"},
		&core.SelectField{Name: "moderation_state", Required: true, MaxSelect: 1, Values: []string{guestbookStateUnreviewed, guestbookStateApproved, guestbookStateRejected}},
		&core.DateField{Name: "retention_until"},
		&core.DateField{Name: "created"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("create guestbook collection: %v", err)
	}
}

func saveGuestbookEntry(t *testing.T, app core.App, state, created, name, location, message, retention string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(constants.CollectionGuestbook)
	if err != nil {
		t.Fatalf("find guestbook collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("moderation_state", state)
	record.Set("created", created)
	record.Set("name", name)
	record.Set("location", location)
	record.Set("message", message)
	if retention != "" {
		record.Set("retention_until", retention)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("create guestbook entry: %v", err)
	}
	return record
}

func assertYearOptions(t *testing.T, app core.App, want []string) {
	t.Helper()
	got, err := yearOptions(app, "2026")
	if err != nil {
		t.Fatalf("get year options: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("year options = %v, want %v", got, want)
	}
}

func mapKeys(values map[string]rateLimitWindow) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func anyStrings(values []any) []string {
	stringsOut := make([]string, len(values))
	for index, value := range values {
		stringsOut[index] = fmt.Sprint(value)
	}
	return stringsOut
}
