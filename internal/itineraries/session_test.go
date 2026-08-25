package itineraries

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewTokenIsOpaqueAndUnique(t *testing.T) {
	first, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if first == "" || second == "" {
		t.Fatal("tokens must not be empty")
	}
	if first == second {
		t.Fatal("two tokens must not collide")
	}
	if len(first) != 43 {
		t.Errorf("token length = %d, want 43 (32 bytes base64url)", len(first))
	}
	if !ValidToken(first) {
		t.Error("a freshly issued token must be valid")
	}
}

func TestValidTokenRejectsMalformed(t *testing.T) {
	if ValidToken("") {
		t.Error("empty token must be invalid")
	}
	if ValidToken("not-base64!") {
		t.Error("non-alphabet token must be invalid")
	}
	if ValidToken("short") {
		t.Error("short token must be invalid")
	}
	if ValidToken(base64.RawURLEncoding.EncodeToString(make([]byte, 16))) {
		t.Error("16-byte token must be invalid (want exactly 32 bytes)")
	}
	// A padded encoding is not the canonical unpadded form.
	padded := base64.URLEncoding.EncodeToString(make([]byte, sessionTokenBytes))
	if ValidToken(padded) {
		t.Error("padded token must be invalid")
	}
}

func TestValidTokenRejectsNonCanonicalTrailingBits(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	// 32 bytes -> 43 base64url chars; the final char carries 4 data bits with
	// two zero padding bits in the low positions. Setting those low bits
	// produces a value that decodes to the same bytes but is not canonical.
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	b := []byte(token)
	last := b[len(b)-1]
	idx := strings.IndexByte(alphabet, last)
	mutated := string(b[:len(b)-1]) + string(alphabet[idx|0x03])

	if mutated == token {
		t.Fatal("mutation failed to alter the token")
	}
	if _, err := base64.RawURLEncoding.DecodeString(mutated); err != nil {
		t.Fatalf("mutated token should still decode: %v", err)
	}
	if ValidToken(mutated) {
		t.Error("non-canonical trailing bits must be rejected")
	}
}

func TestOwnerDigestIsDeterministicAndDistinct(t *testing.T) {
	digest := OwnerDigest("token-a")
	if digest != OwnerDigest("token-a") {
		t.Error("digest must be deterministic")
	}
	if digest == OwnerDigest("token-b") {
		t.Error("different tokens must yield different digests")
	}
	// A tampered token must map to a different owner (and therefore a
	// different, empty draft), never another visitor's draft.
	if digest == OwnerDigest("tampered") {
		t.Error("tampered token must not resolve to the same owner")
	}
}

func TestCSRFTokenBinding(t *testing.T) {
	secret := "opaque-bearer"
	token := CSRFToken(secret)
	if token == "" {
		t.Fatal("CSRF token must not be empty")
	}
	if token == CSRFToken("other-secret") {
		t.Error("different secrets must yield different CSRF tokens")
	}
	if !ValidCSRF(secret, token) {
		t.Error("valid token must verify")
	}
	if ValidCSRF(secret, "tampered") {
		t.Error("tampered token must not verify")
	}
	if ValidCSRF("", token) || ValidCSRF(secret, "") {
		t.Error("empty secret or submitted token must not verify")
	}
}

func TestTokenFromRequestEnumeratesStrictly(t *testing.T) {
	newRequest := func(cookies ...*http.Cookie) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/itineraries/new", nil)
		for _, cookie := range cookies {
			r.AddCookie(cookie)
		}
		return r
	}
	valid, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	if got := TokenFromRequest(newRequest(), ProductionSessionCookieName); got != "" {
		t.Errorf("no cookie = %q, want empty", got)
	}
	if got := TokenFromRequest(newRequest(&http.Cookie{Name: ProductionSessionCookieName, Value: "bogus"}), ProductionSessionCookieName); got != "" {
		t.Errorf("invalid value = %q, want empty", got)
	}
	if got := TokenFromRequest(newRequest(
		&http.Cookie{Name: ProductionSessionCookieName, Value: valid},
		&http.Cookie{Name: ProductionSessionCookieName, Value: valid},
	), ProductionSessionCookieName); got != "" {
		t.Errorf("duplicate same-name cookies = %q, want empty", got)
	}
	if got := TokenFromRequest(newRequest(&http.Cookie{Name: ProductionSessionCookieName, Value: valid}), ProductionSessionCookieName); got != valid {
		t.Errorf("single valid cookie = %q, want %q", got, valid)
	}
	// The legacy name is ignored.
	if got := TokenFromRequest(newRequest(&http.Cookie{Name: LegacySessionCookieName, Value: valid}), ProductionSessionCookieName); got != "" {
		t.Errorf("legacy cookie must be ignored, got %q", got)
	}
}

func TestSameOriginChecks(t *testing.T) {
	canonical := mustParseOrigin(t, "https://gallery.example")
	insecure := mustParseOrigin(t, "http://gallery.example")

	newRequest := func(origin, referer, host string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "http://"+host+"/itineraries/draft/add", nil)
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if referer != "" {
			r.Header.Set("Referer", referer)
		}
		return r
	}

	// HTTPS canonical origin.
	if !SameOrigin(newRequest("https://gallery.example", "", "gallery.example"), canonical) {
		t.Error("https Origin must be accepted against an https canonical origin")
	}
	if SameOrigin(newRequest("http://gallery.example", "", "gallery.example"), canonical) {
		t.Error("http Origin must be rejected against an https canonical origin")
	}
	if SameOrigin(newRequest("https://evil.example", "", "gallery.example"), canonical) {
		t.Error("mismatched Origin host must be rejected")
	}
	if SameOrigin(newRequest("https://gallery.example", "", "evil.example"), canonical) {
		t.Error("mismatched request Host must be rejected")
	}
	if !SameOrigin(newRequest("", "https://gallery.example/itineraries", "gallery.example"), canonical) {
		t.Error("https Referer must be accepted when Origin is absent")
	}
	if SameOrigin(newRequest("not a url", "", "gallery.example"), canonical) {
		t.Error("malformed Origin must be rejected")
	}
	if !SameOrigin(newRequest("", "", "gallery.example"), canonical) {
		t.Error("absent Origin and Referer fall back to the synchroniser token")
	}

	// Default-port-aware semantics.
	if !SameOrigin(newRequest("https://gallery.example:443", "", "gallery.example:443"), canonical) {
		t.Error("https://host:443 must equal the default-port canonical origin")
	}
	if !SameOrigin(newRequest("http://gallery.example:80", "", "gallery.example:80"), insecure) {
		t.Error("http://host:80 must equal the default-port canonical origin")
	}

	// HTTP canonical origin.
	if !SameOrigin(newRequest("http://gallery.example", "", "gallery.example"), insecure) {
		t.Error("http Origin must be accepted against an http canonical origin")
	}
	if SameOrigin(newRequest("https://gallery.example", "", "gallery.example"), insecure) {
		t.Error("https Origin must be rejected against an http canonical origin")
	}

	if SameOrigin(newRequest("https://gallery.example", "", "gallery.example"), nil) {
		t.Error("nil canonical origin must fail closed")
	}
}

func TestSessionCookieShape(t *testing.T) {
	cookie := SessionCookie("token", ProductionSessionCookieName, true)
	if cookie.Name != ProductionSessionCookieName {
		t.Errorf("cookie name = %q, want %q", cookie.Name, ProductionSessionCookieName)
	}
	if cookie.Value != "token" {
		t.Errorf("cookie value = %q, want token", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if !cookie.Secure {
		t.Error("Secure must follow the explicit registration policy")
	}
	if cookie.Path != "/" {
		t.Errorf("cookie path = %q, want /", cookie.Path)
	}
	if cookie.Domain != "" {
		t.Error("session cookie must be host-only (no Domain attribute)")
	}

	insecure := SessionCookie("token", "wga_itinerary_dev", false)
	if insecure.Secure {
		t.Error("development policy cookie must not be Secure")
	}
	if insecure.Name == ProductionSessionCookieName {
		t.Error("development cookie name must differ from production")
	}
}

func mustParseOrigin(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse origin %q: %v", raw, err)
	}
	return parsed
}
