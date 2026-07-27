package postcards

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
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

	_ = savePostcard(app, event, bluemonday.NewPolicy(), config.Captcha{}, nil)

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
			wantCode: http.StatusBadRequest,
		},
		{
			name: "provider failure returns server fault",
			verifier: func(context.Context, string, string) (bool, error) {
				return false, errors.New("provider unavailable")
			},
			wantCode: http.StatusInternalServerError,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := testutils.NewTestApp(t)
			collection := core.NewBaseCollection("postcards")
			if err := app.Save(collection); err != nil {
				t.Fatalf("create postcards collection: %v", err)
			}
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

			_ = savePostcard(app, event, bluemonday.NewPolicy(), protectedCaptcha(t), test.verifier)
			if recorder.Code != test.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantCode)
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
		"WGA_ENV":                "production",
		"WGA_PROTOCOL":           "https",
		"WGA_HOSTNAME":           "gallery.example",
		"WGA_SENDER_NAME":        "WGA",
		"WGA_SENDER_ADDRESS":     "sender@example.test",
		"WGA_RECAPTCHA_SECRET":   "secret",
		"WGA_RECAPTCHA_SITE_KEY": "site-key",
	}
	server, err := config.LoadFrom(func(key string) string { return values[key] }).Server()
	if err != nil {
		t.Fatalf("load protected config: %v", err)
	}

	return server.Captcha
}
