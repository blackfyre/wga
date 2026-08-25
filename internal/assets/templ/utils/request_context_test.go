package utils_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	templutils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func TestContextFromRequestBionicReading(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
		want   bool
	}{
		{name: "absent cookie"},
		{name: "on", cookie: "on", want: true},
		{name: "off", cookie: "off"},
		{name: "unrecognised value", cookie: "enabled"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "wga_bionic", Value: test.cookie})
			}

			if got := templutils.GetBionicReading(templutils.ContextFromRequest(request)); got != test.want {
				t.Fatalf("expected bionic reading %t, got %t", test.want, got)
			}
		})
	}
}

func TestContextFromRequestNilRequest(t *testing.T) {
	if templutils.GetBionicReading(templutils.ContextFromRequest(nil)) {
		t.Fatal("expected bionic reading to be disabled")
	}
}

func TestContextFromRequestPreservesRequestContext(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), key, "value"))
	request.AddCookie(&http.Cookie{Name: "wga_bionic", Value: "on"})

	ctx := templutils.ContextFromRequest(request)
	if got := ctx.Value(key); got != "value" {
		t.Fatalf("expected preserved value %q, got %q", "value", got)
	}
	if !templutils.GetBionicReading(ctx) {
		t.Fatal("expected bionic reading to be enabled")
	}

	cancelled, cancel := context.WithCancel(request.Context())
	cancel()
	cancelledRequest := request.WithContext(cancelled)
	if err := templutils.ContextFromRequest(cancelledRequest).Err(); err != context.Canceled {
		t.Fatalf("expected cancelled context, got %v", err)
	}
}
