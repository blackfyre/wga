package handlers

import (
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/klauspost/compress/gzhttp"
	"github.com/pocketbase/pocketbase/core"
)

// compressionMinSize is the smallest body worth compressing; below it the
// encoding overhead outweighs the saving.
const compressionMinSize = 1024

// compressionETagSuffix marks the tag of a compressed representation so it
// differs from the uncompressed one. gzhttp writes "-gzip" for gzip and
// rewrites it to "-zstd" for zstd; static.matchingETag accepts both forms.
const compressionETagSuffix = "-gzip"

// newCompressionWrapper returns the handler wrapper that compresses text
// responses. Already-compressed media, partial content, responses that chose
// their own encoding, and bodies under compressionMinSize pass through.
func newCompressionWrapper() (func(http.Handler) http.HandlerFunc, error) {
	compress, err := gzhttp.NewWrapper(
		gzhttp.MinSize(compressionMinSize),
		gzhttp.ContentTypeFilter(isCompressibleContentType),
		gzhttp.SuffixETag(compressionETagSuffix),
	)
	if err != nil {
		return nil, err
	}
	return func(next http.Handler) http.HandlerFunc {
		compressed := compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(compressingWriter{ResponseWriter: w}, r)
		}))
		return func(w http.ResponseWriter, r *http.Request) {
			if values := r.Header.Values("Accept-Encoding"); acceptsAnyEncoding(values) {
				// gzhttp selects only explicitly named codings and reads one
				// header line; a wildcard naming neither gzip nor zstd allows both.
				r.Header.Set("Accept-Encoding", strings.Join(append(values, "gzip"), ", "))
			}
			compressed(w, r)
		}
	}, nil
}

// compressingWriter keeps file responses inside the compressor. PocketBase's
// ResponseWriter.ReadFrom, which http.ServeContent reaches through io.Copy,
// unwraps writers until it finds an io.ReaderFrom; without this type it would
// unwrap past the compressor to the connection and send embedded CSS and
// JavaScript uncompressed. Unwrap stays available for http.ResponseController
// flushing and deadlines.
type compressingWriter struct {
	http.ResponseWriter
}

// ReadFrom copies through Write so the compressor sees every byte.
func (w compressingWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(writerOnly{w.ResponseWriter}, r)
}

// Unwrap returns the compressor's writer.
func (w compressingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// writerOnly hides every optional interface so io.Copy uses Write.
type writerOnly struct {
	io.Writer
}

// registerResponseCompression wraps the built HTTP handler once, outside every
// router middleware, so route middleware keeps seeing uncompressed writes.
// PocketBase assigns Server.Handler in the final OnServe step, which runs
// inside se.Next(); the listener starts serving only after the hook returns.
func registerResponseCompression(app core.App) error {
	wrap, err := newCompressionWrapper()
	if err != nil {
		return err
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := se.Next(); err != nil {
			return err
		}
		if se.Server.Handler != nil {
			se.Server.Handler = wrap(se.Server.Handler)
		}
		return nil
	})
	return nil
}

// isCompressibleContentType reports whether a response media type is text
// that benefits from compression. Unknown or missing types are not
// compressed, so binary files never gain an encoding by accident.
func isCompressibleContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType == "text/event-stream" {
		return false
	}
	switch {
	case strings.HasPrefix(mediaType, "text/"):
		return true
	case strings.HasSuffix(mediaType, "+json"), strings.HasSuffix(mediaType, "+xml"):
		return true
	}
	switch mediaType {
	case "application/json", "application/javascript", "application/xml", "image/svg+xml":
		return true
	}
	return false
}

// acceptsAnyEncoding reports whether Accept-Encoding allows any coding through
// a "*" entry with a non-zero quality while naming neither gzip nor zstd, which
// keep their own explicit preference.
func acceptsAnyEncoding(values []string) bool {
	wildcard := false
	for _, value := range values {
		for _, entry := range strings.Split(value, ",") {
			coding, params, _ := strings.Cut(strings.TrimSpace(entry), ";")
			coding = strings.ToLower(strings.TrimSpace(coding))
			switch coding {
			case "gzip", "x-gzip", "zstd":
				return false
			case "*":
				wildcard = !zeroQuality(params)
			}
		}
	}
	return wildcard
}

// zeroQuality reports whether Accept-Encoding parameters carry q=0.
func zeroQuality(params string) bool {
	for _, param := range strings.Split(params, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if ok && strings.EqualFold(strings.TrimSpace(name), "q") {
			q, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			return err == nil && q == 0
		}
	}
	return false
}
