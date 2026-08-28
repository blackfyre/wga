package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func newRequestEvent(t *testing.T, target string) *core.RequestEvent {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, target, nil)
	return &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: httptest.NewRecorder(),
		},
	}
}

func TestRequestsMainContentArea(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{name: "shell id", target: "mc-area", want: true},
		{name: "shell selector", target: "#mc-area", want: true},
		{name: "shell id with whitespace", target: "  mc-area  ", want: true},
		{name: "feature local", target: "timeline", want: false},
		{name: "dual block", target: "dual-area", want: false},
		{name: "dual pane", target: "dual-left", want: false},
		{name: "empty target", target: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/timeline", nil)
			if test.target != "" {
				req.Header.Set("HX-Target", test.target)
			}

			event := &core.RequestEvent{
				Event: router.Event{
					Request:  req,
					Response: httptest.NewRecorder(),
				},
			}

			if got := RequestsMainContentArea(event); got != test.want {
				t.Errorf("RequestsMainContentArea(%q) = %v, want %v", test.target, got, test.want)
			}
		})
	}
}

func TestErrorHelpersRenderSharedShellWithExactStatuses(t *testing.T) {
	tests := []struct {
		name      string
		render    func(*core.RequestEvent) error
		status    int
		fragments []string
	}{
		{
			name:   "not found",
			render: NotFoundError,
			status: http.StatusNotFound,
			fragments: []string{
				"This record is not in the collection.",
				"RETURN TO THE GALLERY",
				"BROWSE ALL ARTWORKS",
			},
		},
		{
			name:   "server fault",
			render: func(event *core.RequestEvent) error {
				return ServerFaultError(event)
			},
			status: http.StatusInternalServerError,
			fragments: []string{
				"The archive could not complete that request.",
				"RETURN TO THE GALLERY",
			},
		},
		{
			name:   "bad request",
			render: BadRequestError,
			status: http.StatusBadRequest,
			fragments: []string{
				"That request is not supported.",
				"RETURN TO THE GALLERY",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := newRequestEvent(t, "/pages/example")
			if err := test.render(event); err != nil {
				t.Fatalf("render error helper: %v", err)
			}

			recorder := event.Response.(*httptest.ResponseRecorder)
			if got := recorder.Code; got != test.status {
				t.Errorf("status = %d, want %d", got, test.status)
			}
			if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", got)
			}
			body := recorder.Body.String()
			for _, fragment := range test.fragments {
				if !strings.Contains(body, fragment) {
					t.Errorf("response does not contain %q", fragment)
				}
			}
			if strings.Contains(body, "Internal server error") {
				t.Errorf("response must not return a bare internal server error")
			}
		})
	}
}

func TestServerFaultErrorRecordsOptionalFailure(t *testing.T) {
	event := newRequestEvent(t, "/pages/example")
	cause := errors.New("render failed")

	if err := ServerFaultError(event, ServerFailure{Category: "page_render", Cause: cause}); err != nil {
		t.Fatalf("render server fault: %v", err)
	}

	failure, ok := ServerFailureFrom(event)
	if !ok {
		t.Fatal("expected server failure metadata")
	}
	if failure.Category != "page_render" {
		t.Errorf("category = %q, want page_render", failure.Category)
	}
	if failure.Cause != cause {
		t.Errorf("cause = %v, want %v", failure.Cause, cause)
	}

	recorder := event.Response.(*httptest.ResponseRecorder)
	if got := recorder.Code; got != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestErrorHelpersHonourBionicReadingContext(t *testing.T) {
	tests := []struct {
		name      string
		cookie    string
		wantState string
	}{
		{name: "disabled without cookie", wantState: `aria-checked="false"`},
		{name: "enabled with on cookie", cookie: "on", wantState: `aria-checked="true"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pages/example", nil)
			if test.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "wga_bionic", Value: test.cookie})
			}

			event := &core.RequestEvent{
				Event: router.Event{
					Request:  req,
					Response: httptest.NewRecorder(),
				},
			}

			if err := NotFoundError(event); err != nil {
				t.Fatalf("render not found helper: %v", err)
			}

			recorder := event.Response.(*httptest.ResponseRecorder)
			if !strings.Contains(recorder.Body.String(), test.wantState) {
				t.Errorf("bionic toggle does not reflect the request context, want %s", test.wantState)
			}
		})
	}
}
