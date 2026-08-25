package itineraries

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
)

// CookiePolicy describes how the anonymous session cookie is issued for one
// transport mode. HttpOnly, SameSite=Lax, Path=/, and the host-only (no
// Domain) attribute are fixed by the workflow's SessionCookie builder; only the
// name and the Secure flag vary between modes.
type CookiePolicy struct {
	// Name is the cookie name. The production policy must use the __Host-
	// prefixed name; the development policy must use a distinct name.
	Name string
	// Secure marks the cookie Secure. It is always true for the production
	// policy and false for the explicitly opted-in HTTP development policy.
	Secure bool
}

// TrustedClientID resolves the trusted client identity for a request. It
// returns ok=false when the identity is absent or invalid, in which case
// guarded state creation and publication must fail closed. The raw identity is
// hashed before use in admission limits and must never be logged.
type TrustedClientID func(*http.Request) (string, bool)

// SecurityPolicy is the explicit, validated security configuration for the
// itineraries handlers. The serial integration owner constructs it from the
// central configuration (canonical public URL and environment) plus an
// explicitly configured trusted-proxy identity parser.
type SecurityPolicy struct {
	// CanonicalOrigin is the canonical public origin (scheme://host[:port])
	// against which request Host and Origin/Referer headers are validated.
	CanonicalOrigin string
	// Production is the HTTPS cookie policy. It must use the __Host- prefix
	// and be Secure.
	Production CookiePolicy
	// Development, when opted in (non-empty Name), is the HTTP cookie policy
	// used when the canonical origin scheme is http. It must use a distinct
	// name and is never selected from backend TLS state.
	Development CookiePolicy
	// TrustedClientID resolves the trusted client identity. It is required.
	TrustedClientID TrustedClientID
}

// Validate reports whether the policy is complete and internally consistent.
// RegisterHandlers fails closed on any validation error.
func (p SecurityPolicy) Validate() error {
	canonical, err := parseCanonicalOrigin(p.CanonicalOrigin)
	if err != nil {
		return err
	}

	if p.Production.Name != itineraryworkflow.ProductionSessionCookieName {
		return fmt.Errorf("itinerary security policy: production cookie name must be %q", itineraryworkflow.ProductionSessionCookieName)
	}
	if !p.Production.Secure {
		return fmt.Errorf("itinerary security policy: production cookie must be Secure")
	}

	switch canonical.Scheme {
	case "https":
		// Production policy is authoritative; a development policy is optional.
	case "http":
		if p.Development.Name == "" {
			return fmt.Errorf("itinerary security policy: http canonical origin requires an opted-in development cookie policy")
		}
		if p.Development.Name == p.Production.Name {
			return fmt.Errorf("itinerary security policy: development cookie name must differ from production")
		}
		if p.Development.Secure {
			return fmt.Errorf("itinerary security policy: development cookie must not be Secure")
		}
	}

	if p.TrustedClientID == nil {
		return fmt.Errorf("itinerary security policy: TrustedClientID resolver is required")
	}

	return nil
}

// canonicalURL returns the parsed canonical origin, or an error.
func (p SecurityPolicy) canonicalURL() (*url.URL, error) {
	return parseCanonicalOrigin(p.CanonicalOrigin)
}

// activeCookiePolicy returns the cookie policy for the canonical origin's
// scheme. The choice is driven entirely by configured state, never by whether
// the backend observed TLS on the request.
func (p SecurityPolicy) activeCookiePolicy() (CookiePolicy, error) {
	canonical, err := p.canonicalURL()
	if err != nil {
		return CookiePolicy{}, err
	}

	if canonical.Scheme == "https" {
		return p.Production, nil
	}

	return p.Development, nil
}

// ActiveCookie returns the cookie policy selected for the canonical origin's
// scheme. The central projection middleware uses it to issue and read the
// visitor's anonymous session cookie on public document requests.
func ActiveCookie(policy SecurityPolicy) (CookiePolicy, error) {
	return policy.activeCookiePolicy()
}

func parseCanonicalOrigin(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, fmt.Errorf("itinerary security policy: canonical origin is required")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("itinerary security policy: invalid canonical origin %q", raw)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("itinerary security policy: canonical origin scheme must be http or https")
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("itinerary security policy: canonical origin must include a host")
	}
	if parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("itinerary security policy: canonical origin must be a bare scheme://host[:port] origin")
	}

	parsed.Scheme = scheme
	return parsed, nil
}
