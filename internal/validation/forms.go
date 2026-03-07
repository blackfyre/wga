package validation

import (
	"strings"

	"github.com/blackfyre/wga/internal/errs"
)

func ValidateHoneypot(name string, email string) error {
	if strings.TrimSpace(name) != "" || strings.TrimSpace(email) != "" {
		return errs.ErrHoneypotTriggered
	}

	return nil
}

func ValidateMessage(message string) error {
	if strings.TrimSpace(message) == "" {
		return errs.ErrMessageRequired
	}

	return nil
}

func ValidateRecaptchaToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return errs.ErrRecaptchaTokenRequired
	}

	return nil
}
