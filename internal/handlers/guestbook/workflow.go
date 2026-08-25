package guestbook

import (
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/validation"
)

const (
	guestbookNameMaxLength       = 100
	guestbookLocationMaxLength   = 120
	guestbookMessageMaxLength    = 1000
	guestbookPrivateRetention    = 90 * 24 * time.Hour
	guestbookStateUnreviewed     = "unreviewed"
	guestbookStateApproved       = "approved"
	guestbookStateRejected       = "rejected"
)

type submissionInput struct {
	Name          string `form:"sender_name"`
	Location      string `form:"location"`
	Message       string `form:"message"`
	HoneyPotName  string `form:"name"`
	HoneyPotEmail string `form:"email"`
}

type submissionErrors struct {
	Name     string
	Location string
	Message  string
	Form     string
}

func (e submissionErrors) any() bool {
	return e.Name != "" || e.Location != "" || e.Message != "" || e.Form != ""
}

func prepareSubmission(input submissionInput) (submissionInput, submissionErrors, error) {
	if err := validation.ValidateHoneypot(input.HoneyPotName, input.HoneyPotEmail); err != nil {
		return submissionInput{}, submissionErrors{}, err
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Location = strings.TrimSpace(input.Location)
	input.Message = strings.TrimSpace(input.Message)
	input.HoneyPotName = ""
	input.HoneyPotEmail = ""

	validationErrors := submissionErrors{}
	if input.Name == "" {
		validationErrors.Name = "Enter your name."
	} else if len([]rune(input.Name)) > guestbookNameMaxLength {
		validationErrors.Name = "Name must be 100 characters or fewer."
	}
	if input.Location == "" {
		validationErrors.Location = "Enter your location."
	} else if len([]rune(input.Location)) > guestbookLocationMaxLength {
		validationErrors.Location = "Location must be 120 characters or fewer."
	}
	if input.Message == "" {
		validationErrors.Message = "Enter a note."
	} else if len([]rune(input.Message)) > guestbookMessageMaxLength {
		validationErrors.Message = "Note must be 1,000 characters or fewer."
	}

	return input, validationErrors, nil
}
