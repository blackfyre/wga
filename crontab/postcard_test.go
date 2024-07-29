package crontab

import (
	"net/mail"
	"testing"
)
	"net/mail"
	"testing"

	"github.com/pocketbase/pocketbase"
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
		t.Errorf("sendMail returned an error: %v", err)
	}

}
