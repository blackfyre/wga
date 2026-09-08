// Package requestprotection owns admission policy for public catalogue reads.
package requestprotection

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Profile identifies the independent admission budget used by a route.
type Profile string

// Route profiles. Unclassified routes deliberately remain outside protection
// until they are explicitly assigned.
const (
	ProfileUnclassified Profile = "unclassified"
	ProfileExempt       Profile = "exempt"
	ProfileSearch       Profile = "search"
	ProfileFragment     Profile = "fragment"
	ProfileDetail       Profile = "detail"
)

// Protected reports whether the profile requires canonical ingress and read
// admission.
func (p Profile) Protected() bool {
	return p == ProfileSearch || p == ProfileFragment || p == ProfileDetail
}

// Classify assigns GET and HEAD request paths to an admission profile. Other
// methods and unknown paths remain unclassified so write workflows and newly
// added routes are not silently brought under read admission.
func Classify(method string, path string) Profile {
	if method != http.MethodGet && method != http.MethodHead {
		return ProfileUnclassified
	}

	switch path {
	case "/artists", "/artworks", "/artworks/results":
		return ProfileSearch
	case "/dual-mode":
		return ProfileFragment
	case "/health", "/robots.txt", "/sitemap.xml", "/sitemap.xsl":
		return ProfileExempt
	}

	if hasPathPrefix(path, "/assets/") || hasPathPrefix(path, "/sitemap/") || hasPathPrefix(path, "/_/") {
		return ProfileExempt
	}
	if isArtistDetail(path) || isAgentDetail(path) {
		return ProfileDetail
	}

	return ProfileUnclassified
}

func hasPathPrefix(path string, prefix string) bool {
	return path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix)
}

func isArtistDetail(path string) bool {
	const prefix = "/artists/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	segments := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(segments) < 1 || len(segments) > 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
	}
	return true
}

func isAgentDetail(path string) bool {
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segments) != 3 || segments[0] != "agents" || (segments[1] != "artists" && segments[1] != "artworks") {
		return false
	}

	name := segments[2]
	return strings.HasSuffix(name, ".md") && len(name) > len(".md")
}

// CanonicalHost validates protected requests against the configured public
// authority while allowing exempt and unclassified routes on deployment hosts.
type CanonicalHost struct {
	host        string
	port        string
	defaultPort string
}

// NewCanonicalHost constructs a canonical-host policy from the configured
// absolute public URL.
func NewCanonicalHost(publicURL string) (CanonicalHost, error) {
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return CanonicalHost{}, fmt.Errorf("public URL must be an absolute origin")
	}

	host, port, err := parseAuthority(parsed.Host)
	if err != nil {
		return CanonicalHost{}, fmt.Errorf("invalid public URL authority: %w", err)
	}

	defaultPort := ""
	switch strings.ToLower(parsed.Scheme) {
	case "http":
		defaultPort = "80"
	case "https":
		defaultPort = "443"
	default:
		return CanonicalHost{}, fmt.Errorf("public URL scheme must be http or https")
	}

	return CanonicalHost{host: host, port: port, defaultPort: defaultPort}, nil
}

// Allows reports whether a request host may access the classified route.
func (h CanonicalHost) Allows(profile Profile, requestHost string) bool {
	if !profile.Protected() {
		return true
	}

	host, port, err := parseAuthority(requestHost)
	if err != nil || host != h.host {
		return false
	}

	canonicalPort := h.port
	if canonicalPort == "" {
		canonicalPort = h.defaultPort
	}
	requestPort := port
	if requestPort == "" {
		requestPort = h.defaultPort
	}
	return requestPort == canonicalPort
}

func parseAuthority(authority string) (string, string, error) {
	parsed, err := url.Parse("//" + authority)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", fmt.Errorf("invalid host authority")
	}

	host := strings.ToLower(parsed.Hostname())
	if strings.HasSuffix(host, ".") {
		host = strings.TrimSuffix(host, ".")
	}
	if host == "" || strings.HasSuffix(host, ".") {
		return "", "", fmt.Errorf("invalid host name")
	}

	return host, parsed.Port(), nil
}
