package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestGenerateCurrentRelativePageUrl(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "path without query",
			url:  "/artworks",
			want: "/artworks",
		},
		{
			name: "path with query",
			url:  "/pages/privacy-policy?mode=preview",
			want: "/pages/privacy-policy?mode=preview",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, test.url, nil)
			if err != nil {
				t.Fatalf("failed to build request: %v", err)
			}

			event := &core.RequestEvent{
				Event: router.Event{
					Request:  req,
					Response: httptest.NewRecorder(),
				},
			}

			if got := GenerateCurrentRelativePageUrl(event); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
