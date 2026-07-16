package postcards

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const recaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

type recaptchaVerifyResponse struct {
	Success bool `json:"success"`
}

func verifyRecaptchaToken(ctx context.Context, client *http.Client, secret string, token string, remoteIP string) (bool, error) {
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, recaptchaVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("recaptcha verify endpoint returned status %d", resp.StatusCode)
	}

	var payload recaptchaVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}

	return payload.Success, nil
}
