package crontab

import (
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func TestSendMail(t *testing.T) {

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: "./wga_data",
	})

	mailClient := app.NewMailClient()

	message := &mailer.Message{
		From: mail.Address{
			Name:    "sender",
			Address: "sender@example.com",
		},
		To: []mail.Address{
			{
				Name:    "recipient",
				Address: "recipient@example.com",
			},
		},
		Subject: "Test Subject",
		HTML:    "<html><body>Test Body</body></html>",
	}

	err := mailClient.Send(message)
	if err != nil {
		if strings.Contains(err.Error(), "failed to locate a sendmail executable path") {
			t.Skip("sendmail is unavailable in this environment")
		}

		t.Errorf("sendMail returned an error: %v", err)
	}

}

func TestLogPostcardDeliveryUsesSafeRunFields(t *testing.T) {
	app := testutils.NewTestApp(t)
	captured := testutils.CaptureLogs(app)

	logPostcardDelivery(
		app,
		"run-123",
		1,
		2,
		"sent",
		nil,
	)
	logPostcardDelivery(
		app,
		"run-123",
		2,
		2,
		"failed",
		errors.New("recipient@example.test token-value message-body-value"),
	)

	testutils.FlushLogs(t, app)
	entries := testutils.LogsWithEvent(captured(), "postcard.delivery.attempt")
	if len(entries) != 2 {
		t.Fatalf("expected 2 postcard delivery logs, got %d", len(entries))
	}
	var failedEntry *core.Log
	for _, entry := range entries {
		if got := entry.Data["run_id"]; got != "run-123" {
			t.Fatalf("run_id = %v, want %q", got, "run-123")
		}
		if entry.Data["outcome"] == "failed" {
			failedEntry = entry
		}
	}
	if failedEntry == nil {
		t.Fatal("expected a failed postcard delivery log")
	}
	entry := failedEntry
	if _, ok := entry.Data["postcard_id"]; ok {
		t.Fatal("delivery log must not contain the postcard pickup identifier")
	}
	if got := fmt.Sprint(entry.Data["attempt"]); got != "1" {
		t.Fatalf("attempt = %s, want 1", got)
	}
	if got := entry.Data["outcome"]; got != "failed" {
		t.Fatalf("outcome = %v, want %q", got, "failed")
	}

	output := fmt.Sprint(testutils.LogData(captured()))
	for _, sensitive := range []string{"recipient@example.test", "token-value", "message-body-value"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}

func TestProcessPostcardCompletesAfterPartialDeliveryFailure(t *testing.T) {
	app := testutils.NewTestApp(t)
	collection := core.NewBaseCollection("postcards")
	collection.Fields.Add(
		&core.TextField{Name: "status"},
		&core.TextField{Name: "recipients"},
		&core.TextField{Name: "sender_name"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("create postcards collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("status", "queued")
	record.Set("recipients", "first@example.test,second@example.test,third@example.test")
	record.Set("sender_name", "sender")
	if err := app.Save(record); err != nil {
		t.Fatalf("create postcard: %v", err)
	}

	mailClient := &scriptedMailer{
		outcomes: []error{nil, errors.New("mail transport failed"), nil},
	}
	if processPostcard(record, app, mailClient, postcardTestConfig(t), "run-123") {
		t.Fatal("expected postcard delivery to fail")
	}
	wantRecipients := []string{"first@example.test", "second@example.test", "third@example.test"}
	if !reflect.DeepEqual(mailClient.recipients, wantRecipients) {
		t.Fatalf("attempted recipients = %v, want %v", mailClient.recipients, wantRecipients)
	}

	stored, err := app.FindRecordById(collection.Id, record.Id)
	if err != nil {
		t.Fatalf("reload postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "sent" {
		t.Fatalf("status = %q, want %q", got, "sent")
	}
	queuedRecords, err := app.FindRecordsByFilter(collection.Id, "status = 'queued'", "", 0, 0)
	if err != nil {
		t.Fatalf("load queued postcards: %v", err)
	}
	if len(queuedRecords) != 0 {
		t.Fatalf("queued postcards = %d, want 0", len(queuedRecords))
	}
}

type scriptedMailer struct {
	outcomes   []error
	recipients []string
}

func (m *scriptedMailer) Send(message *mailer.Message) error {
	m.recipients = append(m.recipients, message.To[0].Address)

	return m.outcomes[len(m.recipients)-1]
}

func postcardTestConfig(t *testing.T) config.Postcards {
	t.Helper()

	values := map[string]string{
		"WGA_ENV":            "test",
		"WGA_PROTOCOL":       "http",
		"WGA_HOSTNAME":       "example.test",
		"WGA_SENDER_NAME":    "WGA",
		"WGA_SENDER_ADDRESS": "sender@example.test",
	}
	runtimeConfig := config.LoadFrom(func(key string) string {
		return values[key]
	})
	server, err := runtimeConfig.Server()
	if err != nil {
		t.Fatalf("load postcard config: %v", err)
	}

	return server.Postcards
}
