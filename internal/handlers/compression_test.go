package handlers

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/handlers/health"
	"github.com/blackfyre/wga/internal/handlers/landing"
	"github.com/blackfyre/wga/internal/handlers/static"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func compressionTestHandler(t *testing.T, contentType string, body []byte, etag string) http.Handler {
	t.Helper()
	wrap, err := newCompressionWrapper()
	if err != nil {
		t.Fatalf("create compression wrapper: %v", err)
	}
	return wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		if etag != "" {
			w.Header().Set("ETag", etag)
		}
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
	}))
}

func compressionRequest(handler http.Handler, acceptEncoding string, header map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if acceptEncoding != "" {
		request.Header.Set("Accept-Encoding", acceptEncoding)
	}
	for name, value := range header {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func gunzip(t *testing.T, body []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("open gzip body: %v", err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	return decoded
}

func TestCompressionCompressesTextAndKeepsBodyIntact(t *testing.T) {
	body := []byte(strings.Repeat("<p>Web Gallery of Art</p>", 200))
	handler := compressionTestHandler(t, "text/html; charset=utf-8", body, `"abc"`)

	response := compressionRequest(handler, "gzip", nil)

	if got := response.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := response.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Errorf("Vary = %q, want Accept-Encoding", got)
	}
	if got := response.Header().Get("Content-Length"); got == "5200" {
		t.Errorf("Content-Length = %s, the uncompressed length", got)
	}
	if got := response.Header().Get("ETag"); got != `"abc-gzip"` {
		t.Errorf("ETag = %q, want encoding-suffixed tag", got)
	}
	if response.Body.Len() >= len(body) {
		t.Errorf("compressed size %d is not smaller than %d", response.Body.Len(), len(body))
	}
	if decoded := gunzip(t, response.Body.Bytes()); !bytes.Equal(decoded, body) {
		t.Error("decompressed body differs from the uncompressed response")
	}
}

func TestCompressionLeavesResponsesUncompressed(t *testing.T) {
	text := []byte(strings.Repeat("plain text ", 400))
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x42}, 8192)...)
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0x42}, 8192)...)

	cases := []struct {
		name           string
		contentType    string
		body           []byte
		acceptEncoding string
		header         map[string]string
	}{
		{name: "client without compression", contentType: "text/html", body: text},
		{name: "small body", contentType: "text/plain; charset=utf-8", body: []byte("ok"), acceptEncoding: "gzip"},
		{name: "jpeg image", contentType: "image/jpeg", body: jpeg, acceptEncoding: "gzip, zstd"},
		{name: "png image", contentType: "image/png", body: png, acceptEncoding: "gzip"},
		{name: "unknown binary", contentType: "application/octet-stream", body: text, acceptEncoding: "gzip"},
		{name: "range request", contentType: "text/plain", body: text, acceptEncoding: "gzip", header: map[string]string{"Range": "bytes=0-1999"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := compressionRequest(compressionTestHandler(t, tc.contentType, tc.body, ""), tc.acceptEncoding, tc.header)

			if got := response.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding = %q, want none", got)
			}
			want := tc.body
			if tc.header["Range"] != "" {
				if response.Code != http.StatusPartialContent {
					t.Fatalf("status = %d, want 206", response.Code)
				}
				want = tc.body[:2000]
			}
			if !bytes.Equal(response.Body.Bytes(), want) {
				t.Error("body is not byte-exact")
			}
		})
	}
}

func TestCompressibleContentTypes(t *testing.T) {
	cases := map[string]bool{
		"text/html; charset=utf-8":     true,
		"text/markdown; charset=utf-8": true,
		"text/css":                     true,
		"application/javascript":       true,
		"application/json":             true,
		"application/ld+json":          true,
		"application/manifest+json":    true,
		"application/xml":              true,
		"application/rss+xml":          true,
		"image/svg+xml":                true,
		"image/jpeg":                   false,
		"image/webp":                   false,
		"image/avif":                   false,
		"audio/mpeg":                   false,
		"video/mp4":                    false,
		"application/zip":              false,
		"application/octet-stream":     false,
		"":                             false,
		"not a media type;;":           false,
	}
	for contentType, want := range cases {
		if got := isCompressibleContentType(contentType); got != want {
			t.Errorf("isCompressibleContentType(%q) = %v, want %v", contentType, got, want)
		}
	}
}

// TestRegisteredCompressionWrapsTheServedHandler proves that serve-time
// registration compresses a real page while the health probe stays plain.
func TestRegisteredCompressionWrapsTheServedHandler(t *testing.T) {
	app := newTrustedHeadRouterTestApp(t)
	if err := registerResponseCompression(app); err != nil {
		t.Fatalf("register compression: %v", err)
	}
	landing.RegisterHandlers(app)
	health.RegisterHandlers(app)
	static.RegisterHandlers(app)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router, Server: &http.Server{}}
	if err := app.OnServe().Trigger(serveEvent, func(se *core.ServeEvent) error {
		mux, err := se.Router.BuildMux()
		if err != nil {
			return err
		}
		se.Server.Handler = mux
		return nil
	}); err != nil {
		t.Fatalf("trigger serve: %v", err)
	}

	page := compressionRequest(serveEvent.Server.Handler, "gzip", map[string]string{"Accept": "text/html"})
	if page.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", page.Code)
	}
	if got := page.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("GET / Content-Encoding = %q, want gzip", got)
	}
	if decoded := gunzip(t, page.Body.Bytes()); !bytes.Contains(decoded, []byte("<html")) {
		t.Error("decompressed homepage is not an HTML document")
	}

	probe := httptest.NewRequest(http.MethodGet, "/health", nil)
	probe.Header.Set("Accept-Encoding", "gzip")
	probeResponse := httptest.NewRecorder()
	serveEvent.Server.Handler.ServeHTTP(probeResponse, probe)
	if got := probeResponse.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("GET /health Content-Encoding = %q, want none", got)
	}
	if got := probeResponse.Body.String(); got != "ok" {
		t.Errorf("GET /health body = %q, want ok", got)
	}
}

func TestCompressionHonoursWildcardAcceptEncoding(t *testing.T) {
	body := []byte(strings.Repeat("<p>Web Gallery of Art</p>", 200))
	cases := map[string]string{
		"*":                     "gzip",
		"br, *;q=0.5":           "gzip",
		"*;q=0":                 "",
		"gzip;q=0, *":           "zstd",
		"gzip;q=0, zstd;q=0, *": "",
		"x-gzip;q=0, *":         "zstd",
		"identity":              "",
		"zstd, *":               "zstd",
	}
	for acceptEncoding, want := range cases {
		response := compressionRequest(compressionTestHandler(t, "text/html", body, ""), acceptEncoding, nil)
		if got := response.Header().Get("Content-Encoding"); got != want {
			t.Errorf("Accept-Encoding %q: Content-Encoding = %q, want %q", acceptEncoding, got, want)
		}
	}
}

func TestCompressionReadsEveryAcceptEncodingLine(t *testing.T) {
	body := []byte(strings.Repeat("<p>Web Gallery of Art</p>", 200))
	handler := compressionTestHandler(t, "text/html", body, "")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Add("Accept-Encoding", "br")
	request.Header.Add("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip from the second header line", got)
	}
}
