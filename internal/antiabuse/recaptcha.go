package antiabuse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const recaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

type VerificationErrorKind string

const (
	VerificationErrorTimeout     VerificationErrorKind = "timeout"
	VerificationErrorUnavailable VerificationErrorKind = "unavailable"
	VerificationErrorContract    VerificationErrorKind = "contract"
)

type VerificationError struct {
	Kind VerificationErrorKind
}

func (e *VerificationError) Error() string {
	return string(e.Kind)
}

type Verifier interface {
	Verify(context.Context, string, string) (bool, error)
}

type recaptchaVerifier struct {
	client *http.Client
	secret string
	url    string
}

type recaptchaResponse struct {
	Success *bool `json:"success"`
}

func NewRecaptchaVerifier(client *http.Client, secret string) Verifier {
	return newRecaptchaVerifier(client, secret, recaptchaVerifyURL)
}

func newRecaptchaVerifier(client *http.Client, secret string, endpoint string) Verifier {
	return &recaptchaVerifier{client: client, secret: secret, url: endpoint}
}

func (v *recaptchaVerifier) Verify(ctx context.Context, token string, remoteIP string) (bool, error) {
	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.url, strings.NewReader(form.Encode()))
	if err != nil {
		return false, &VerificationError{Kind: VerificationErrorContract}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, &VerificationError{Kind: VerificationErrorTimeout}
		}
		return false, &VerificationError{Kind: VerificationErrorUnavailable}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, &VerificationError{Kind: VerificationErrorUnavailable}
	}

	var response recaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, &VerificationError{Kind: VerificationErrorContract}
	}
	if response.Success == nil {
		return false, &VerificationError{Kind: VerificationErrorContract}
	}

	return *response.Success, nil
}
