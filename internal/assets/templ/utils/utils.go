package utils

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/blackfyre/wga/internal/utils/publicurl"
)

type ContextKey string

// trustedHeadMarkupContextKey is a private typed key for operator-trusted
// markup rendered verbatim in the document head. It is intentionally distinct
// from the generic ContextKey so the executable-content trust boundary cannot
// collide with ordinary string context values.
type trustedHeadMarkupContextKey struct{}

type requestPathContextKey struct{}
type htmxRequestContextKey struct{}

const defaultDescription = "Explore European artists and artworks from the 3rd century to the early 20th in the Web Gallery of Art."

var TitleKey ContextKey = "title"
var DescriptionKey ContextKey = "description"
var EnvironmentKey ContextKey = "environment"
var OgTitleKey ContextKey = "og:title"
var OgDescriptionKey ContextKey = "og:description"
var OgImageKey ContextKey = "og:image"
var OgUrlKey ContextKey = "og:url"
var OgTypeKey ContextKey = "og:type"
var OgSiteNameKey ContextKey = "og:site_name"
var OgImageAltKey ContextKey = "og:image:alt"
var TwitterCardKey ContextKey = "twitter:card"
var TwitterSiteKey ContextKey = "twitter:site"
var TwitterCreatorKey ContextKey = "twitter:creator"
var TwitterTitleKey ContextKey = "twitter:title"
var TwitterDescriptionKey ContextKey = "twitter:description"
var TwitterImageKey ContextKey = "twitter:image"
var TwitterImageAltKey ContextKey = "twitter:image:alt"
var CanonicalUrlKey ContextKey = "canonical:url"
var AlternateMarkdownURLKey ContextKey = "alternate:markdown:url"

func ContextFromRequest(request *http.Request) context.Context {
	if request == nil {
		return context.WithValue(context.Background(), requestPathContextKey{}, "")
	}

	ctx := context.WithValue(request.Context(), requestPathContextKey{}, request.URL.Path)
	return context.WithValue(ctx, htmxRequestContextKey{}, request.Header.Get("HX-Request") == "true")
}

// IsHTMXRequest reports whether the render context belongs to an HTMX request.
func IsHTMXRequest(c context.Context) bool {
	value, _ := c.Value(htmxRequestContextKey{}).(bool)
	return value
}

// RequestPath returns the request URL path captured for shared template rendering.
func RequestPath(c context.Context) string {
	path, _ := c.Value(requestPathContextKey{}).(string)
	return normalizePath(path)
}

// IsPathActive reports whether candidate is the most specific destination that
// owns the request path. A root destination owns only the root; all other
// destinations own their path and children on a path boundary. Public artist
// artwork records are an alias for the ARTWORKS shell destination.
func IsPathActive(c context.Context, candidate string, destinations []string) bool {
	current := navigationPath(RequestPath(c))
	candidate = normalizePath(candidate)
	if current == "" || candidate == "" {
		return false
	}

	allDestinations := append([]string{}, destinations...)
	allDestinations = append(allDestinations, candidate)

	active := ""
	for _, destination := range allDestinations {
		destination = normalizePath(destination)
		if !pathOwns(destination, current) {
			continue
		}
		if len(destination) > len(active) {
			active = destination
		}
	}

	return candidate == active
}

// navigationPath resolves public record routes to their top-level shell
// destination. Singular and plural artist aliases share this ownership: artist
// records and selections belong to ARTISTS, while artwork records belong to
// ARTWORKS.
func navigationPath(value string) string {
	value = normalizePath(value)
	parts := strings.Split(strings.Trim(value, "/"), "/")
	if len(parts) == 3 && (parts[0] == "artist" || parts[0] == "artists") && parts[2] != "selections" {
		return "/artworks"
	}
	if len(parts) >= 2 && (parts[0] == "artist" || parts[0] == "artists") {
		return "/artists"
	}

	return value
}

func normalizePath(value string) string {
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	if value != "/" {
		value = strings.TrimRight(value, "/")
	}
	return value
}

func pathOwns(destination string, current string) bool {
	if destination == "/" {
		return current == "/"
	}

	return current == destination || strings.HasPrefix(current, destination+"/")
}

// WithTrustedHeadMarkup returns a context carrying the supplied operator-trusted
// head markup. The value is immutable and intended only for the shared layout's
// dedicated raw rendering boundary; it is never parsed, sanitised, or logged.
func WithTrustedHeadMarkup(c context.Context, markup string) context.Context {
	return context.WithValue(c, trustedHeadMarkupContextKey{}, markup)
}

// GetTrustedHeadMarkup returns the trusted head markup stored in the context, or
// an empty string when no value was stored. Empty content must not render.
func GetTrustedHeadMarkup(c context.Context) string {
	markup, _ := c.Value(trustedHeadMarkupContextKey{}).(string)
	return markup
}

func AssetUrl(path string) string {
	return publicurl.Resolve(path)
}

// GetTitle retrieves the title from the context.
// If the title is found, it returns the title as a string.
// If the title is not found, it returns an empty string.
func GetTitle(c context.Context) string {
	if v, ok := c.Value(TitleKey).(string); ok && strings.TrimSpace(v) != "" {
		return v + " - WGA"
	}

	return "Web Gallery of Art"
}

// GetCanonicalUrl returns the canonical URL from the given context.
// If the canonical URL is found in the context, it is returned.
// Otherwise, an empty string is returned.
func GetCanonicalUrl(c context.Context) string {
	if v, ok := c.Value(CanonicalUrlKey).(string); ok {
		return v
	}

	return ""
}

// GetAlternateMarkdownURL returns the generated Markdown alternate URL stored
// in the render context, or an empty string when the page has no alternate.
func GetAlternateMarkdownURL(c context.Context) string {
	if v, ok := c.Value(AlternateMarkdownURLKey).(string); ok {
		return v
	}

	return ""
}

// GetDescription retrieves the description value from the context.
// If the value is found and is of type string, it is returned.
// Otherwise, an empty string is returned.
func GetDescription(c context.Context) string {
	if v, ok := c.Value(DescriptionKey).(string); ok && strings.TrimSpace(v) != "" {
		return v
	}

	return defaultDescription
}

// GetEnvironment returns the environment value from the given context.
// If the environment value is not found in the context, it returns "dev" as the default value.
func GetEnvironment(c context.Context) string {
	if v, ok := c.Value(EnvironmentKey).(string); ok {
		return v
	}

	return "dev"
}

func GetOpenGraphTags(c context.Context) map[string]string {
	ogTags := map[string]string{
		"og:title":       GetTitle(c),
		"og:description": GetDescription(c),
		"og:type":        "website",
		"og:site_name":   "Web Gallery of Art",
	}

	if v, ok := c.Value(OgTitleKey).(string); ok && strings.TrimSpace(v) != "" {
		ogTags["og:title"] = v
	}

	if v, ok := c.Value(OgDescriptionKey).(string); ok && strings.TrimSpace(v) != "" {
		ogTags["og:description"] = v
	}

	if v, ok := c.Value(OgImageKey).(string); ok && strings.TrimSpace(v) != "" {
		ogTags["og:image"] = v
		ogTags["og:image:alt"] = GetTitle(c)
	} else {
		ogTags["og:image"] = AssetUrl("/assets/images/smo_cover_1080x1080.png")
		ogTags["og:image:alt"] = "Web Gallery of Art"
		ogTags["og:image:width"] = "1080"
		ogTags["og:image:height"] = "1080"
	}
	if v, ok := c.Value(OgImageAltKey).(string); ok && v != "" {
		ogTags["og:image:alt"] = v
	}

	if v, ok := c.Value(OgUrlKey).(string); ok && strings.TrimSpace(v) != "" {
		ogTags["og:url"] = v
	} else if canonical := GetCanonicalUrl(c); canonical != "" {
		ogTags["og:url"] = canonical
	}

	if v, ok := c.Value(OgTypeKey).(string); ok && strings.TrimSpace(v) != "" {
		ogTags["og:type"] = v
	} else {
		ogTags["og:type"] = "website"
	}

	if v, ok := c.Value(OgSiteNameKey).(string); ok && strings.TrimSpace(v) != "" {
		ogTags["og:site_name"] = v
	}

	return ogTags
}

func GetTwitterTags(c context.Context) map[string]string {
	twitterTags := map[string]string{
		"twitter:title":       GetTitle(c),
		"twitter:description": GetDescription(c),
	}

	if v, ok := c.Value(TwitterCardKey).(string); ok {
		twitterTags["twitter:card"] = v
	} else {
		twitterTags["twitter:card"] = "summary_large_image"
	}

	if v, ok := c.Value(TwitterSiteKey).(string); ok {
		twitterTags["twitter:site"] = v
	}

	if v, ok := c.Value(TwitterCreatorKey).(string); ok {
		twitterTags["twitter:creator"] = v
	}

	if v, ok := c.Value(TwitterTitleKey).(string); ok && strings.TrimSpace(v) != "" {
		twitterTags["twitter:title"] = v
	}

	if v, ok := c.Value(TwitterDescriptionKey).(string); ok && strings.TrimSpace(v) != "" {
		twitterTags["twitter:description"] = v
	}

	if v, ok := c.Value(TwitterImageKey).(string); ok && strings.TrimSpace(v) != "" {
		twitterTags["twitter:image"] = v
		twitterTags["twitter:image:alt"] = GetTitle(c)
	} else {
		twitterTags["twitter:image"] = AssetUrl("/assets/images/smo_cover_1080x1080.png")
		twitterTags["twitter:image:alt"] = "Web Gallery of Art"
	}
	if v, ok := c.Value(TwitterImageAltKey).(string); ok && v != "" {
		twitterTags["twitter:image:alt"] = v
	}

	return twitterTags
}

// DecorateContext decorates the given context with a key-value pair.
// It returns a new context with the provided key-value pair added.
func DecorateContext(c context.Context, k ContextKey, v string) context.Context {

	if k == TitleKey || k == OgTitleKey || k == TwitterTitleKey {
		cwv := context.WithValue(c, TitleKey, v)
		cwv = context.WithValue(cwv, OgTitleKey, v)
		cwv = context.WithValue(cwv, TwitterTitleKey, v)
		return cwv
	}

	if k == DescriptionKey || k == OgDescriptionKey || k == TwitterDescriptionKey {

		v = StripHtmlTags(v)

		if len(v) > 160 {
			v = v[:157] + "..."
		}

		cwv := context.WithValue(c, DescriptionKey, v)
		cwv = context.WithValue(cwv, OgDescriptionKey, v)
		cwv = context.WithValue(cwv, TwitterDescriptionKey, v)
		return cwv
	}

	if k == OgImageKey || k == TwitterImageKey {
		cwv := context.WithValue(c, OgImageKey, v)
		cwv = context.WithValue(cwv, TwitterImageKey, v)
		return cwv
	}

	if k == OgUrlKey || k == CanonicalUrlKey {
		cwv := context.WithValue(c, OgUrlKey, v)
		cwv = context.WithValue(cwv, CanonicalUrlKey, v)
		return cwv
	}

	return context.WithValue(c, k, v)
}

func StripHtmlTags(s string) string {
	// remove all html tags
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}
