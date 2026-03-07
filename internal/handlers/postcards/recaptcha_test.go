package postcards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host
	req.Host = r.target.Host

	return r.base.RoundTrip(req)
}

func TestVerifyRecaptchaToken(t *testing.T) {
	t.Run("returns true when captcha provider confirms success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST request, got %s", r.Method)
			}

			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("expected form content type, got %s", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success": true}`))
		}))
		defer ts.Close()

		targetURL, err := url.Parse(ts.URL)
		if err != nil {
			t.Fatalf("failed to parse test server url: %v", err)
		}

		client := &http.Client{
			Transport: &rewriteTransport{base: http.DefaultTransport, target: targetURL},
		}

		verified, err := verifyRecaptchaToken(context.Background(), client, "secret", "token", "127.0.0.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !verified {
			t.Fatalf("expected token to be verified")
		}
	})

	t.Run("returns false when captcha provider rejects token", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success": false}`))
		}))
		defer ts.Close()

		targetURL, err := url.Parse(ts.URL)
		if err != nil {
			t.Fatalf("failed to parse test server url: %v", err)
		}

		client := &http.Client{
			Transport: &rewriteTransport{base: http.DefaultTransport, target: targetURL},
		}

		verified, err := verifyRecaptchaToken(context.Background(), client, "secret", "token", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if verified {
			t.Fatalf("expected token to be rejected")
		}
	})
}
