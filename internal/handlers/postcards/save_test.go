package postcards

import (
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

	_ = savePostcard(app, event, bluemonday.NewPolicy(), config.Captcha{})

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
