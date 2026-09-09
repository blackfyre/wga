package handlers

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/agentcontent"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestPrefersMarkdown(t *testing.T) {
	tests := []struct {
		accept string
		want   bool
	}{
		{accept: "text/markdown", want: true},
		{accept: "Text/Markdown; charset=utf-8", want: true},
		{accept: "text/markdown, text/html", want: true},
		{accept: "text/markdown;q=0.5, text/html;q=0.9", want: false},
		{accept: "text/markdown;q=0.5, text/*;q=0.9", want: false},
		{accept: "text/markdown;q=0.5, */*;q=0.9", want: false},
		{accept: "text/markdown;q=0, */*;q=1", want: false},
		{accept: "text/markdown;q=0.9, text/*;q=0.5", want: true},
		{accept: "text/html,application/xhtml+xml,*/*", want: false},
		{accept: "text/*", want: false},
		{accept: "*/*", want: false},
		{accept: "", want: false},
		{accept: "text/markdown;q=invalid", want: false},
	}
	for _, test := range tests {
		if got := prefersMarkdown(test.accept); got != test.want {
			t.Errorf("prefersMarkdown(%q) = %t, want %t", test.accept, got, test.want)
		}
	}
}

func TestMarkdownNegotiationPrecedesDocumentWork(t *testing.T) {
	path := "/artists/authoritative-name-artistone000001"
	event, recorder := markdownNegotiationEvent(path, "text/markdown")
	nextCalled := false
	next := func() error {
		nextCalled = true
		event.Response.Header().Set("Set-Cookie", "must-not-be-set=1")
		return nil
	}

	err := negotiateGeneratedMarkdown(nil, event, func(core.App, string) (agentcontent.Resource, error) {
		return agentcontent.Resource{CanonicalURL: "https://gallery.example" + path}, nil
	}, next)
	if err != nil {
		t.Fatal(err)
	}
	if nextCalled {
		t.Fatal("document middleware was invoked before Markdown redirect")
	}
	if recorder.Code != http.StatusTemporaryRedirect || recorder.Header().Get("Location") != "/agents/artists/artistone000001.md" {
		t.Fatalf("response = %d Location %q", recorder.Code, recorder.Header().Get("Location"))
	}
	if recorder.Header().Get("Set-Cookie") != "" {
		t.Fatal("Markdown redirect set a session cookie")
	}
	if !strings.Contains(recorder.Header().Get("Vary"), "Accept") {
		t.Fatalf("Vary = %q", recorder.Header().Get("Vary"))
	}
}

func TestMarkdownNegotiationCombinesAcceptFieldLines(t *testing.T) {
	path := "/artists/authoritative-name-artistone000001"
	event, recorder := markdownNegotiationEvent(path, "text/markdown;q=0.5")
	event.Request.Header.Add("Accept", "text/html;q=1")
	nextCalled := false

	err := negotiateGeneratedMarkdown(nil, event, func(core.App, string) (agentcontent.Resource, error) {
		return agentcontent.Resource{CanonicalURL: "https://gallery.example" + path}, nil
	}, func() error {
		nextCalled = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !nextCalled || recorder.Code == http.StatusTemporaryRedirect {
		t.Fatal("combined Accept preference did not preserve HTML")
	}
}

func TestMarkdownNegotiationAcceptsPublicCoauthorRoute(t *testing.T) {
	requested := "/artists/second-author-artisttwo000001/shared-work-workone00000001"
	event, recorder := markdownNegotiationEvent(requested, "text/markdown")
	nextCalled := false

	err := negotiateGeneratedMarkdown(nil, event, func(core.App, string) (agentcontent.Resource, error) {
		return agentcontent.Resource{
			CanonicalURL: "https://gallery.example/artists/first-author-artistone000001/shared-work-workone00000001",
			AcceptedPaths: []string{
				"/artists/first-author-artistone000001/shared-work-workone00000001",
				requested,
			},
		}, nil
	}, func() error {
		nextCalled = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if nextCalled {
		t.Fatal("public coauthor route continued into HTML work")
	}
	if recorder.Code != http.StatusTemporaryRedirect || recorder.Header().Get("Location") != "/agents/artworks/workone00000001.md" {
		t.Fatalf("response = %d Location %q", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestMarkdownNegotiationPreservesRecordValidation(t *testing.T) {
	for _, test := range []struct {
		name      string
		path      string
		canonical string
		readErr   error
	}{
		{name: "missing publication", path: "/artists/missing-missing0000001", readErr: fs.ErrNotExist},
		{name: "noncanonical alias", path: "/artists/alias-artistone000001", canonical: "https://gallery.example/artists/authoritative-name-artistone000001"},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, recorder := markdownNegotiationEvent(test.path, "text/markdown")
			nextCalled := false
			next := func() error { nextCalled = true; return nil }
			err := negotiateGeneratedMarkdown(nil, event, func(core.App, string) (agentcontent.Resource, error) {
				return agentcontent.Resource{CanonicalURL: test.canonical}, test.readErr
			}, next)
			if err != nil {
				t.Fatal(err)
			}
			if !nextCalled {
				t.Fatal("normal record validation was bypassed")
			}
			if recorder.Code == http.StatusTemporaryRedirect {
				t.Fatal("invalid record request redirected to generated Markdown")
			}
		})
	}
}

func markdownNegotiationEvent(path string, accept string) (*core.RequestEvent, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Accept", accept)
	recorder := httptest.NewRecorder()
	return &core.RequestEvent{Event: router.Event{Request: request, Response: recorder}}, recorder
}
