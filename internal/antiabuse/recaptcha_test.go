package antiabuse

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecaptchaVerifier(t *testing.T) {
	t.Run("accepts and rejects provider decisions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("secret") != "secret" || r.Form.Get("response") != "token" {
				t.Fatalf("form = %v", r.Form)
			}
			_, _ = w.Write([]byte(`{"success": true}`))
		}))
		defer server.Close()

		verified, err := newRecaptchaVerifier(server.Client(), "secret", server.URL).Verify(context.Background(), "token", "")
		if err != nil || !verified {
			t.Fatalf("verify = %t, %v", verified, err)
		}
	})

	t.Run("returns false for a rejected token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"success": false}`))
		}))
		defer server.Close()

		verified, err := newRecaptchaVerifier(server.Client(), "secret", server.URL).Verify(context.Background(), "token", "")
		if err != nil || verified {
			t.Fatalf("verify = %t, %v", verified, err)
		}
	})

	for _, test := range []struct {
		name    string
		handler http.Handler
		timeout time.Duration
		want    VerificationErrorKind
	}{
		{
			name: "maps malformed responses",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`not json`))
			}),
			want: VerificationErrorContract,
		},
		{
			name: "maps incomplete responses",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			}),
			want: VerificationErrorContract,
		},
		{
			name: "maps provider status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}),
			want: VerificationErrorUnavailable,
		},
		{
			name: "maps client timeout",
			handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				time.Sleep(100 * time.Millisecond)
			}),
			timeout: 10 * time.Millisecond,
			want:    VerificationErrorTimeout,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.handler)
			defer server.Close()
			client := server.Client()
			client.Timeout = test.timeout

			_, err := newRecaptchaVerifier(client, "secret", server.URL).Verify(context.Background(), "token", "")
			var verificationErr *VerificationError
			if !errors.As(err, &verificationErr) {
				t.Fatalf("error = %v, want verification error", err)
			}
			if verificationErr.Kind != test.want {
				t.Fatalf("error kind = %q, want %q", verificationErr.Kind, test.want)
			}
		})
	}
}
