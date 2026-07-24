package crontab

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/mail"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
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
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Logs.MaxDays = 1
	captured := captureDeliveryLogs(app)

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

	flushDeliveryLogs(t, app)
	entries := deliveryLogsWithEvent(captured(), "postcard.delivery.attempt")
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

	output := fmt.Sprint(deliveryLogData(captured()))
	for _, sensitive := range []string{"recipient@example.test", "token-value", "message-body-value"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}

func TestProcessPostcardLeavesFailedDeliveryQueued(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
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
	record.Set("recipients", "recipient@example.test")
	record.Set("sender_name", "sender")
	if err := app.Save(record); err != nil {
		t.Fatalf("create postcard: %v", err)
	}

	if processPostcard(record, app, failingMailer{}, postcardTestConfig(t), "run-123") {
		t.Fatal("expected postcard delivery to fail")
	}

	stored, err := app.FindRecordById(collection.Id, record.Id)
	if err != nil {
		t.Fatalf("reload postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "queued" {
		t.Fatalf("status = %q, want %q", got, "queued")
	}
}

type failingMailer struct{}

func (failingMailer) Send(*mailer.Message) error {
	return errors.New("mail transport failed")
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

func captureDeliveryLogs(app *tests.TestApp) func() []*core.Log {
	var captured []*core.Log
	app.OnModelCreate(core.LogsTableName).BindFunc(func(e *core.ModelEvent) error {
		log, ok := e.Model.(*core.Log)
		if ok {
			entry := *log
			entry.Data = maps.Clone(log.Data)
			captured = append(captured, &entry)
		}

		return e.Next()
	})

	return func() []*core.Log {
		return captured
	}
}

func flushDeliveryLogs(t *testing.T, app *tests.TestApp) {
	t.Helper()

	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	if err := handler.WriteAll(context.Background()); err != nil {
		t.Fatalf("write logs: %v", err)
	}
}

func deliveryLogsWithEvent(logs []*core.Log, event string) []*core.Log {
	entries := []*core.Log{}
	for _, entry := range logs {
		if entry.Data["event"] == event {
			entries = append(entries, entry)
		}
	}

	return entries
}

func deliveryLogData(logs []*core.Log) []any {
	data := make([]any, len(logs))
	for index, entry := range logs {
		data[index] = entry.Data
	}

	return data
}
