package validation

import "testing"

func TestValidateHoneypot(t *testing.T) {
	t.Run("passes when honeypot fields are empty", func(t *testing.T) {
		if err := ValidateHoneypot("", ""); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("fails when honeypot fields are populated", func(t *testing.T) {
		if err := ValidateHoneypot("bot", ""); err == nil {
			t.Fatalf("expected honeypot validation error")
		}
	})
}

func TestValidateMessage(t *testing.T) {
	t.Run("passes for non-empty message", func(t *testing.T) {
		if err := ValidateMessage("hello"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("fails for blank message", func(t *testing.T) {
		if err := ValidateMessage("   "); err == nil {
			t.Fatalf("expected message-required validation error")
		}
	})
}

func TestValidateRecaptchaToken(t *testing.T) {
	t.Run("passes for non-empty token", func(t *testing.T) {
		if err := ValidateRecaptchaToken("token"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("fails for blank token", func(t *testing.T) {
		if err := ValidateRecaptchaToken(" "); err == nil {
			t.Fatalf("expected recaptcha-required validation error")
		}
	})
}
