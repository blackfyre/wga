package postcards

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

type verifierFunc func(context.Context, string, string) (bool, error)

func (f verifierFunc) Verify(ctx context.Context, token string, remoteIP string) (bool, error) {
	return f(ctx, token, remoteIP)
}

func TestSavePostcardDoesNotLogSubmittedForm(t *testing.T) {
	app := testutils.NewTestApp(t)
	captured := testutils.CaptureLogs(app)
	form := url.Values{
		"sender_name":          {"sender-name-value"},
		"sender_email":         {"sender@example.test"},
		"recipients[]":         {"recipient@example.test"},
		"message":              {"message-body-value"},
		"image_id":             {"image-id"},
		"g-recaptcha-response": {"captcha-token-value"},
		"name":                 {"honeypot-name-value"},
	}
	request := httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
	}
	logging.SetRequestID(event, "request-123")

	_ = savePostcard(app, event, config.Captcha{}, config.PostcardTokenKeyring{}, nil, newSubmissionLimiter(3, 10*time.Minute), requesttrust.New(requesttrust.SourceDirect))

	testutils.FlushLogs(t, app)
	entry := testutils.LogWithEvent(captured(), "postcard.submission.rejected")
	if entry == nil {
		t.Fatal("expected a postcard rejection log")
	}
	if got := entry.Data["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want %q", got, "request-123")
	}
	if got := entry.Data["outcome"]; got != "honeypot" {
		t.Fatalf("outcome = %v, want %q", got, "honeypot")
	}

	output := fmt.Sprint(testutils.LogData(captured()))
	for _, sensitive := range []string{
		"sender-name-value",
		"sender@example.test",
		"recipient@example.test",
		"message-body-value",
		"captcha-token-value",
		"honeypot-name-value",
	} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}

func TestSavePostcardCaptchaFailuresDoNotPersist(t *testing.T) {
	for _, test := range []struct {
		name     string
		verifier verifierFunc
		wantCode int
	}{
		{
			name: "rejection returns bad request",
			verifier: func(context.Context, string, string) (bool, error) {
				return false, nil
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "provider failure re-renders the composer as unavailable",
			verifier: func(context.Context, string, string) (bool, error) {
				return false, errors.New("provider unavailable")
			},
			wantCode: http.StatusServiceUnavailable,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := testutils.NewTestApp(t)
			artworkID := installComposeArtwork(t, app, "Work")
			collection := core.NewBaseCollection("postcards")
			if err := app.Save(collection); err != nil {
				t.Fatalf("create postcards collection: %v", err)
			}
			form := url.Values{
				"sender_name":          {"sender"},
				"sender_email":         {"sender@example.test"},
				"recipients[]":         {"recipient@example.test"},
				"message":              {"message"},
				"image_id":             {artworkID},
				"g-recaptcha-response": {"captcha-token"},
			}
			recorder := httptest.NewRecorder()
			event := &core.RequestEvent{
				App: app,
				Event: router.Event{
					Request:  httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode())),
					Response: recorder,
				},
			}
			event.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			limiter := newSubmissionLimiter(3, 10*time.Minute)
			_ = savePostcard(app, event, protectedCaptcha(t), config.PostcardTokenKeyring{}, test.verifier, limiter, requesttrust.New(requesttrust.SourceDirect))
			if recorder.Code != test.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantCode)
			}
			if len(limiter.windows) != 0 {
				t.Fatal("CAPTCHA failure consumed the postcard send allowance")
			}
			body := recorder.Body.String()
			if !strings.Contains(body, `id="postcard-error"`) || !strings.Contains(body, `value="sender@example.test"`) {
				t.Fatalf("CAPTCHA failure did not re-render the composer with entered values: %s", body)
			}
			records, err := app.FindRecordsByFilter("postcards", "", "", 0, 0)
			if err != nil {
				t.Fatalf("find postcards: %v", err)
			}
			if len(records) != 0 {
				t.Fatalf("postcards = %d, want 0", len(records))
			}
		})
	}
}

func protectedCaptcha(t *testing.T) config.Captcha {
	t.Helper()
	values := map[string]string{
		"WGA_ENV":                    "production",
		"WGA_PROTOCOL":               "https",
		"WGA_HOSTNAME":               "gallery.example",
		"WGA_CLIENT_IP_SOURCE":       "cloudflare-railway",
		"WGA_CLOUDFLARE_EDGE_SECRET": base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x61}, 32)),
		"WGA_SENDER_NAME":            "WGA",
		"WGA_SENDER_ADDRESS":         "sender@example.test",
		"WGA_RECAPTCHA_SECRET":       "secret",
		"WGA_RECAPTCHA_SITE_KEY":     "site-key",
	}
	server, err := config.LoadFrom(func(key string) string { return values[key] }).Server()
	if err != nil {
		t.Fatalf("load protected config: %v", err)
	}

	return server.Captcha
}

func TestSavePostcardFailsClosedWithoutTrustedIdentity(t *testing.T) {
	app := testutils.NewTestApp(t)
	collection := core.NewBaseCollection("postcards")
	if err := app.Save(collection); err != nil {
		t.Fatalf("create postcards collection: %v", err)
	}

	verifierCalled := false
	verifier := verifierFunc(func(context.Context, string, string) (bool, error) {
		verifierCalled = true
		return true, nil
	})
	resolver := requesttrust.Resolver(func(*http.Request) (string, bool) {
		return "", false
	})
	form := url.Values{
		"sender_name":          {"sender"},
		"sender_email":         {"sender@example.test"},
		"recipients[]":         {"recipient@example.test"},
		"message":              {"message"},
		"image_id":             {"image-id"},
		"g-recaptcha-response": {"captcha-token"},
	}
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode())),
			Response: recorder,
		},
	}
	event.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_ = savePostcard(app, event, protectedCaptcha(t), config.PostcardTokenKeyring{}, verifier, newSubmissionLimiter(3, 10*time.Minute), resolver)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (fail closed)", recorder.Code)
	}
	if verifierCalled {
		t.Fatal("CAPTCHA must not be reached without a trusted identity")
	}
	records, err := app.FindRecordsByFilter("postcards", "", "", 0, 0)
	if err != nil || len(records) != 0 {
		t.Fatalf("postcards = %d err=%v, want 0", len(records), err)
	}
}

func TestSavePostcardUsesResolvedIdentityForCaptcha(t *testing.T) {
	app := testutils.NewTestApp(t)
	collection := core.NewBaseCollection("postcards")
	if err := app.Save(collection); err != nil {
		t.Fatalf("create postcards collection: %v", err)
	}

	var capturedIP string
	verifier := verifierFunc(func(_ context.Context, _ string, remoteIP string) (bool, error) {
		capturedIP = remoteIP
		return false, nil
	})
	resolver := requesttrust.Resolver(func(*http.Request) (string, bool) {
		return "203.0.113.99", true
	})
	form := url.Values{
		"sender_name":          {"sender"},
		"sender_email":         {"sender@example.test"},
		"recipients[]":         {"recipient@example.test"},
		"message":              {"message"},
		"image_id":             {"image-id"},
		"g-recaptcha-response": {"captcha-token"},
	}
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode())),
			Response: recorder,
		},
	}
	event.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	event.Request.RemoteAddr = "198.51.100.7:54321"

	_ = savePostcard(app, event, protectedCaptcha(t), config.PostcardTokenKeyring{}, verifier, newSubmissionLimiter(3, 10*time.Minute), resolver)
	if capturedIP != "203.0.113.99" {
		t.Errorf("captcha received %q, want resolved identity 203.0.113.99", capturedIP)
	}
}

func TestSavePostcardRecipientRowActionsUseExistingEndpoint(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installComposeArtwork(t, app, "Work")
	resolver := requesttrust.New(requesttrust.SourceDirect)
	for _, test := range []struct {
		name       string
		action     url.Values
		htmx       bool
		wantRows   int
		wantValues []string
	}{
		{name: "ordinary add", action: url.Values{"add_recipient": {"1"}, "recipients[]": {"first@example.test"}}, wantRows: 2, wantValues: []string{"first@example.test"}},
		{name: "htmx remove", action: url.Values{"remove_recipient": {"0"}, "recipients[]": {"first@example.test", "second@example.test"}}, htmx: true, wantRows: 1, wantValues: []string{"second@example.test"}},
		{name: "retain final row", action: url.Values{"remove_recipient": {"0"}, "recipients[]": {"first@example.test"}}, wantRows: 1, wantValues: []string{"first@example.test"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.action.Set("image_id", artworkID)
			request := httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(test.action.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if test.htmx {
				request.Header.Set("HX-Request", "true")
			}
			recorder := httptest.NewRecorder()
			event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: recorder}}
			if err := savePostcard(app, event, config.Captcha{}, config.PostcardTokenKeyring{}, nil, newSubmissionLimiter(3, 10*time.Minute), resolver); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || strings.Count(recorder.Body.String(), `name="recipients[]"`) != test.wantRows {
				t.Fatalf("status=%d rows=%d, want 200/%d", recorder.Code, strings.Count(recorder.Body.String(), `name="recipients[]"`), test.wantRows)
			}
			for _, value := range test.wantValues {
				if !strings.Contains(recorder.Body.String(), `value="`+value+`"`) {
					t.Fatalf("response did not retain %q", value)
				}
			}
			if !test.htmx && !strings.Contains(recorder.Body.String(), "<!doctype html>") {
				t.Fatal("ordinary row action did not render a complete page")
			}
		})
	}
}

func TestSavePostcardSanitisesMessageBeforeValidationRedisplay(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installComposeArtwork(t, app, "Work")
	form := url.Values{
		"sender_name":          {""},
		"sender_email":         {"sender@example.test"},
		"recipients[]":         {"recipient@example.test"},
		"message":              {`<p onclick="alert(1)">Hello <b style="color:red">there</b>.</p><script>alert(2)</script>`},
		"image_id":             {artworkID},
		"g-recaptcha-response": {"dev"},
	}
	request := httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: recorder}}

	if err := savePostcard(app, event, config.Captcha{}, config.PostcardTokenKeyring{}, nil, newSubmissionLimiter(3, 10*time.Minute), requesttrust.New(requesttrust.SourceDirect)); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	body := recorder.Body.String()
	for _, forbidden := range []string{"onclick=\"alert", "style=\"color:red\"", "alert(2)"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("redisplayed form contains %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `&lt;p&gt;Hello &lt;b&gt;there&lt;/b&gt;.&lt;/p&gt;`) {
		t.Fatalf("redisplayed form did not retain safe formatting: %s", body)
	}
}

func TestSavePostcardKeysRateLimitByResolvedIdentity(t *testing.T) {
	app := testutils.NewTestApp(t)
	resolver := requesttrust.Resolver(func(*http.Request) (string, bool) {
		return "203.0.113.99", true
	})
	limiter := newSubmissionLimiter(3, 10*time.Minute)
	form := url.Values{
		"sender_name":          {"sender"},
		"sender_email":         {"sender@example.test"},
		"recipients[]":         {"recipient@example.test"},
		"message":              {"message"},
		"image_id":             {"image-id"},
		"g-recaptcha-response": {"captcha-token"},
	}
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode())),
			Response: httptest.NewRecorder(),
		},
	}
	event.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	event.Request.RemoteAddr = "198.51.100.7:54321"

	_ = savePostcard(app, event, config.Captcha{}, config.PostcardTokenKeyring{}, nil, limiter, resolver)

	if _, ok := limiter.windows["203.0.113.99"]; !ok {
		t.Error("rate limiter was not keyed by the resolved identity")
	}
	if _, ok := limiter.windows["198.51.100.7"]; ok {
		t.Error("rate limiter was keyed by the raw remote address")
	}
}

func TestSavePostcardUnavailableArtworkRetainsValuesAndAllowance(t *testing.T) {
	for _, htmx := range []bool{false, true} {
		t.Run(fmt.Sprintf("htmx=%t", htmx), func(t *testing.T) {
			app := testutils.NewTestApp(t)
			form := url.Values{
				"sender_name":          {"Keyboard Sender"},
				"sender_email":         {"sender@example.test"},
				"recipients[]":         {"recipient@example.test"},
				"message":              {"message"},
				"image_id":             {"missing-artwork"},
				"g-recaptcha-response": {"captcha-token"},
			}
			recorder := httptest.NewRecorder()
			event := &core.RequestEvent{
				App: app,
				Event: router.Event{
					Request:  httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode())),
					Response: recorder,
				},
			}
			event.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if htmx {
				event.Request.Header.Set("HX-Request", "true")
			}
			limiter := newSubmissionLimiter(1, 10*time.Minute)

			if err := savePostcard(app, event, config.Captcha{}, config.PostcardTokenKeyring{}, nil, limiter, requesttrust.New(requesttrust.SourceDirect)); err != nil {
				t.Fatalf("save postcard: %v", err)
			}

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
			}
			body := recorder.Body.String()
			for _, want := range []string{
				`id="postcard-compose"`,
				"no longer available",
				"CHOOSE ANOTHER ARTWORK",
				`value="Keyboard Sender"`,
				`value="recipient@example.test"`,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("response lacks %q: %s", want, body)
				}
			}
			if htmx == strings.Contains(body, "<html") {
				t.Fatalf("htmx=%t rendered the wrong response shape", htmx)
			}
			if !limiter.allow("192.0.2.1", time.Now()) {
				t.Fatal("a submission that queued nothing consumed the send allowance")
			}
		})
	}
}

func TestSubmissionLimiterReleaseRestoresOneAdmission(t *testing.T) {
	limiter := newSubmissionLimiter(1, 10*time.Minute)
	now := time.Now()
	admission, ok := limiter.admit("visitor", now)
	if !ok {
		t.Fatal("first admission rejected")
	}
	if limiter.allow("visitor", now) {
		t.Fatal("second admission accepted beyond the limit")
	}
	limiter.release(admission)
	limiter.release(admission)
	if !limiter.allow("visitor", now) {
		t.Fatal("released admission was not restored")
	}
	if limiter.allow("visitor", now) {
		t.Fatal("repeated release restored more than the consumed admissions")
	}
	limiter.release(submissionAdmission{visitorKey: "unknown", started: now})
	limiter.release(submissionAdmission{})
	if _, ok := limiter.windows["unknown"]; ok {
		t.Fatal("release created a window for an unknown visitor")
	}
}

// TestSubmissionLimiterReleaseIgnoresRolledOverWindow verifies that a
// submission admitted before a window rollover, and failing after it, cannot
// refund an admission in the new window.
func TestSubmissionLimiterReleaseIgnoresRolledOverWindow(t *testing.T) {
	limiter := newSubmissionLimiter(3, 10*time.Minute)
	start := time.Now()
	// Admitted in the old window; its database result arrives after rollover.
	inFlight, ok := limiter.admit("visitor", start)
	if !ok {
		t.Fatal("in-flight submission rejected")
	}
	rolledOver := start.Add(10 * time.Minute)
	for index := range 3 {
		if !limiter.allow("visitor", rolledOver.Add(time.Duration(index)*time.Second)) {
			t.Fatalf("new-window submission %d rejected", index+1)
		}
	}
	limiter.release(inFlight)
	if limiter.allow("visitor", rolledOver.Add(5*time.Second)) {
		t.Fatal("a stale refund allowed a fourth submission in the new window")
	}
}

func TestRenderQueuedPostcardNamesWorkAndKeepsTheLinkPrivate(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installComposeArtwork(t, app, "Queued Work")
	postcard := core.NewRecord(core.NewBaseCollection("postcards"))
	postcard.Set("image_id", artworkID)
	result := &postcardworkflow.QueueResult{
		Postcard:         postcard,
		Access:           []postcardworkflow.RecipientAccess{{Token: "opaque-token", ExpiresAt: types.NowDateTime().Add(time.Hour)}},
		MaskedRecipients: []string{"r••••@example.test"},
	}
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: httptest.NewRequest(http.MethodPost, "/postcard", nil), Response: recorder}}

	if err := renderQueuedPostcard(result, app, event); err != nil {
		t.Fatalf("render queued postcard: %v", err)
	}

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{"Queued Work", "accepted and queued for delivery", `href="/postcard/send?awid=` + artworkID + `"`, "token=opaque-token"} {
		if !strings.Contains(body, want) {
			t.Fatalf("confirmation lacks %q: %s", want, body)
		}
	}
}

func postReplacementForm(t *testing.T, app core.App, form url.Values, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Event: router.Event{Request: httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode())), Response: recorder}}
	event.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if htmx {
		event.Request.Header.Set("HX-Request", "true")
	}
	if err := savePostcard(app, event, config.Captcha{}, config.PostcardTokenKeyring{}, nil, newSubmissionLimiter(3, 10*time.Minute), requesttrust.New(requesttrust.SourceDirect)); err != nil {
		t.Fatalf("save postcard: %v", err)
	}
	return recorder
}

func TestSavePostcardReplacesUnavailableArtworkWithoutLosingTheDraft(t *testing.T) {
	for _, htmx := range []bool{false, true} {
		t.Run(fmt.Sprintf("htmx=%t", htmx), func(t *testing.T) {
			app := testutils.NewTestApp(t)
			replacementID := installComposeArtwork(t, app, "Madonna of the Meadow")
			installComposeArtwork(t, app, "Still Life")
			hiddenID := installComposeArtwork(t, app, "Madonna Withdrawn")
			hidden, err := app.FindRecordById("artworks", hiddenID)
			if err != nil {
				t.Fatal(err)
			}
			hidden.Set("published", false)
			if err := app.Save(hidden); err != nil {
				t.Fatal(err)
			}
			draft := url.Values{
				"sender_name":    {"Keyboard Sender"},
				"sender_email":   {"sender@example.test"},
				"recipients[]":   {"first@example.test", "second@example.test"},
				"message":        {"<p>Kept message</p>"},
				"image_id":       {hiddenID},
				"submission_key": {"draft-key"},
			}
			withAction := func(extra url.Values) url.Values {
				form := url.Values{}
				for key, value := range draft {
					form[key] = value
				}
				for key, value := range extra {
					form[key] = value
				}
				return form
			}
			assertDraft := func(body string) {
				t.Helper()
				for _, want := range []string{`value="Keyboard Sender"`, `value="sender@example.test"`, `value="first@example.test"`, `value="second@example.test"`, "Kept message", `value="draft-key"`} {
					if !strings.Contains(body, want) {
						t.Fatalf("draft lost %q: %s", want, body)
					}
				}
				if htmx == strings.Contains(body, "<html") {
					t.Fatalf("htmx=%t rendered the wrong response shape", htmx)
				}
			}

			short := postReplacementForm(t, app, withAction(url.Values{"find_artwork": {"1"}, "artwork_query": {"M"}}), htmx)
			if !strings.Contains(short.Body.String(), "Enter between 2 and 100 characters") {
				t.Fatalf("short query was not explained: %s", short.Body.String())
			}

			found := postReplacementForm(t, app, withAction(url.Values{"find_artwork": {"1"}, "artwork_query": {"Madonna"}}), htmx)
			body := found.Body.String()
			assertDraft(body)
			if !strings.Contains(body, `name="replace_artwork" value="`+replacementID+`"`) {
				t.Fatalf("matching published artwork not offered: %s", body)
			}
			if strings.Contains(body, `value="`+hiddenID+`" formnovalidate`) || strings.Contains(body, "Still Life") {
				t.Fatalf("unpublished or non-matching artwork offered: %s", body)
			}

			replaced := postReplacementForm(t, app, withAction(url.Values{"replace_artwork": {replacementID}, "artwork_query": {"Madonna"}}), htmx)
			body = replaced.Body.String()
			if replaced.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", replaced.Code, http.StatusOK)
			}
			assertDraft(body)
			if !strings.Contains(body, `name="image_id" value="`+replacementID+`"`) || !strings.Contains(body, "Madonna of the Meadow") {
				t.Fatalf("composer did not switch to the replacement: %s", body)
			}
			if strings.Contains(body, "data-postcard-replacement") || strings.Contains(body, `id="postcard-error"`) {
				t.Fatalf("replacement composer still shows the unavailable state: %s", body)
			}
		})
	}
}
