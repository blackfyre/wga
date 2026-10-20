package handlers

import (
	"net/http"
	"testing"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPublicCatalogueCachePolicy(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		url     string
		headers map[string]string
		status  int
		want    string
	}{
		{name: "anonymous catalogue document", method: http.MethodGet, url: "/artworks", status: http.StatusOK, want: publicCatalogueCacheControl},
		{name: "landing page", method: http.MethodGet, url: "/", status: http.StatusOK, want: publicCatalogueCacheControl},
		{name: "nested catalogue path", method: http.MethodGet, url: "/artists/some-artist-r1", status: http.StatusOK, want: publicCatalogueCacheControl},
		{name: "head request", method: http.MethodHead, url: "/artworks", status: http.StatusOK, want: publicCatalogueCacheControl},
		{name: "htmx fragment", method: http.MethodGet, url: "/artworks", headers: map[string]string{"HX-Request": "true"}, status: http.StatusOK, want: "private, no-store"},
		{name: "response that sets a cookie", method: http.MethodGet, url: "/artworks/cookie", status: http.StatusOK},
		{name: "visitor-state response", method: http.MethodGet, url: "/artworks/private", status: http.StatusOK, want: "private, no-store"},
		{name: "error response", method: http.MethodGet, url: "/artworks/missing", status: http.StatusNotFound},
		{name: "non-catalogue route", method: http.MethodGet, url: "/feedback", status: http.StatusOK},
		{name: "prefix that is not a family", method: http.MethodGet, url: "/artworksx", status: http.StatusOK},
		{name: "wildcard accept", method: http.MethodGet, url: "/artworks", headers: map[string]string{"Accept": "*/*"}, status: http.StatusOK},
		{name: "htmx fragment with wildcard accept", method: http.MethodGet, url: "/artworks", headers: map[string]string{"Accept": "*/*", "HX-Request": "true"}, status: http.StatusOK, want: "private, no-store"},
		{name: "markdown preference", method: http.MethodGet, url: "/artworks", headers: map[string]string{"Accept": "text/markdown, text/html;q=0.5"}, status: http.StatusOK},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var gotStatus int
			headers := map[string]string{"Accept": "text/html"}
			for name, value := range test.headers {
				headers[name] = value
			}
			scenario := tests.ApiScenario{
				Name:            test.name,
				Method:          test.method,
				URL:             test.url,
				Headers:         headers,
				ExpectedStatus:  test.status,
				ExpectedContent: []string{""},
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := testutils.NewTestApp(t)
					registerPublicCatalogueCache(app)
					app.OnServe().BindFunc(func(se *core.ServeEvent) error {
						ok := func(e *core.RequestEvent) error { return e.String(http.StatusOK, "ok") }
						for _, path := range []string{"/{$}", "/artworks", "/artists/{slug}", "/feedback", "/artworksx"} {
							se.Router.GET(path, ok)
						}
						se.Router.GET("/artworks/cookie", func(e *core.RequestEvent) error {
							e.SetCookie(&http.Cookie{Name: "session", Value: "v"})
							return e.String(http.StatusOK, "ok")
						})
						se.Router.GET("/artworks/private", func(e *core.RequestEvent) error {
							utils.KeepResponsePrivate(e)
							return e.String(http.StatusOK, "ok")
						})
						se.Router.GET("/artworks/missing", func(e *core.RequestEvent) error {
							return e.String(http.StatusNotFound, "missing")
						})
						se.Router.BindFunc(func(e *core.RequestEvent) error {
							err := e.Next()
							gotStatus = e.Status()
							return err
						})
						return se.Next()
					})
					return app
				},
				AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
					if got := response.Header.Get("Cache-Control"); got != test.want {
						t.Errorf("Cache-Control = %q; want %q", got, test.want)
					}
					if gotStatus != test.status {
						t.Errorf("tracked status = %d; want %d", gotStatus, test.status)
					}
				},
			}
			scenario.Test(t)
		})
	}
}
