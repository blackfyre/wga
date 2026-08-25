package itineraries

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
)

const (
	// ProductionSessionCookieName is the host-only __Host- session cookie name
	// used under the production HTTPS policy. The __Host- prefix requires the
	// cookie to be Secure, Path=/, and to carry no Domain attribute.
	ProductionSessionCookieName = "__Host-wga_itinerary"

	// LegacySessionCookieName is the pre-remediation cookie name. It is never
	// read: a request carrying only this cookie is treated as cookie-less and
	// issued a fresh token under the active policy name.
	LegacySessionCookieName = "wga_itinerary"

	// DevelopmentSessionCookieName is the non-Secure cookie name used under the
	// explicitly opted-in HTTP development policy. It is distinct from both the
	// production __Host- name and the legacy name.
	DevelopmentSessionCookieName = "wga_itinerary_dev"

	// sessionTokenBytes is the entropy of the opaque bearer token.
	sessionTokenBytes = 32

	// csrfPurpose domain-separates the synchroniser token from any other HMAC
	// derived from the same secret bearer token.
	csrfPurpose = "wga-itinerary-sync-v1"
)

// NewToken returns a crypto-random, base64url-encoded opaque bearer token.
func NewToken() (string, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// ValidToken reports whether token is the canonical base64url encoding of
// exactly sessionTokenBytes bytes. Padded, non-canonical, and any other length
// are rejected.
func ValidToken(token string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	if len(raw) != sessionTokenBytes {
		return false
	}

	return base64.RawURLEncoding.EncodeToString(raw) == token
}

// OwnerDigest derives the persisted owner capability from an opaque bearer
// token. Only the SHA-256 digest is stored; a tampered token therefore maps to
// a different, empty draft rather than another visitor's data.
func OwnerDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// SessionCookie builds the anonymous session cookie for the supplied bearer
// token under the named cookie policy. It is always HttpOnly, SameSite=Lax,
// Path=/, and host-only (no Domain attribute); Secure follows the policy.
func SessionCookie(token string, name string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}

// TokenFromRequest returns the single valid bearer token carried by the named
// session cookie. Zero same-name cookies, multiple same-name cookies, or any
// value that is not a canonical 32-byte base64url token are treated as absent
// and yield an empty string so the caller issues a fresh token.
func TokenFromRequest(r *http.Request, name string) string {
	var values []string
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			values = append(values, cookie.Value)
		}
	}
	if len(values) != 1 || !ValidToken(values[0]) {
		return ""
	}

	return values[0]
}

// CSRFToken derives the synchroniser token bound to the session secret via
// HMAC-SHA256 with a fixed purpose string. It is safe to embed in forms.
func CSRFToken(secret string) string {
	if secret == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(csrfPurpose))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ValidCSRF compares a submitted synchroniser token against the token derived
// from the session secret using a constant-time comparison.
func ValidCSRF(secret string, submitted string) bool {
	if secret == "" || submitted == "" {
		return false
	}

	expected := CSRFToken(secret)
	return hmac.Equal([]byte(expected), []byte(submitted))
}

// SameOrigin reports whether a state-changing request's Host and Origin/Referer
// match the configured canonical origin using default-port-aware semantics. The
// Host header must always match the canonical host; when an Origin or Referer
// header is present it must also match the canonical scheme and host. A request
// carrying neither still relies on the HMAC-bound synchroniser token, which
// non-browser clients cannot obtain.
func SameOrigin(r *http.Request, canonical *url.URL) bool {
	if canonical == nil {
		return false
	}

	scheme := strings.ToLower(canonical.Scheme)
	wantHost, ok := normalizeHost(scheme, canonical.Host)
	if !ok {
		return false
	}

	requestHost, ok := normalizeHost(scheme, r.Host)
	if !ok || requestHost != wantHost {
		return false
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		return originMatches(scheme, wantHost, origin)
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return originMatches(scheme, wantHost, referer)
	}

	return true
}

func originMatches(scheme string, wantHost string, raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Scheme, scheme) {
		return false
	}

	host, ok := normalizeHost(scheme, parsed.Host)
	if !ok {
		return false
	}

	return host == wantHost
}

// normalizeHost lowercases a host and makes its default port explicit for the
// supplied scheme, so "example.com" and "example.com:443" compare equal under
// https, and "example.com" and "example.com:80" compare equal under http.
// A host whose port is not a valid number does not split and therefore never
// compares equal to a well-formed canonical host.
func normalizeHost(scheme string, host string) (string, bool) {
	parsed := &url.URL{Scheme: scheme, Host: host}
	hostname := parsed.Hostname()
	if hostname == "" {
		return "", false
	}

	port := parsed.Port()
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return "", false
		}
	}

	return strings.ToLower(hostname) + ":" + port, true
}
