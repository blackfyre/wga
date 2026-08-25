// Package requesttrust resolves a request's trusted client identity according
// to an explicitly configured source policy. The resolved identity is the
// canonical IP address used only for anonymous-write admission limits; it is
// hashed by callers and must never be persisted or logged in raw form.
package requesttrust

import (
	"net"
	"net/http"
	"strings"
)

// Source selects how the trusted client identity is resolved.
type Source string

// Supported client-identity sources.
const (
	// SourceDirect parses the socket peer (RemoteAddr) and ignores every
	// forwarding header. It is the safe default for local development where the
	// application is reached directly.
	SourceDirect Source = "direct"

	// SourceRailway follows the production Railway-edge contract: it requires
	// exactly one syntactically valid X-Railway-Edge marker plus exactly one
	// parseable X-Real-IP address and ignores X-Forwarded-For. Anything else
	// fails closed.
	SourceRailway Source = "railway"
)

// Resolver returns the trusted client identity for a request. ok is false when
// the identity is absent or invalid, in which case guarded state creation and
// publication must fail closed.
type Resolver func(*http.Request) (string, bool)

// New returns the resolver for the configured source. An unrecognised source
// resolves to the fail-closed direct resolver, matching the validation already
// performed by config.LoadFrom.
func New(source Source) Resolver {
	if source == SourceRailway {
		return resolveRailway
	}

	return resolveDirect
}

// resolveDirect canonicalises the socket peer address and ignores forwarded
// headers entirely.
func resolveDirect(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}

	return canonicaliseIP(r.RemoteAddr)
}

// canonicaliseIP extracts and canonicalises the host portion of a RemoteAddr
// style value. A bare IP without a port is also accepted.
func canonicaliseIP(remoteAddr string) (string, bool) {
	host := remoteAddr
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsedHost
	}

	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return "", false
	}

	return ip.String(), true
}

// resolveRailway requires Railway's edge marker and a single parseable client
// address. X-Forwarded-For is ignored.
func resolveRailway(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}

	if !validRailwayEdge(r.Header) {
		return "", false
	}

	return singleRealIP(r.Header)
}

// validRailwayEdge reports whether the request carries exactly one
// syntactically valid X-Railway-Edge marker. A valid marker is a single
// non-empty token without whitespace or commas.
func validRailwayEdge(header http.Header) bool {
	markers := header.Values("X-Railway-Edge")
	if len(markers) != 1 {
		return false
	}

	return validEdgeMarker(markers[0])
}

// validEdgeMarker reports whether marker is a single non-empty token of
// permitted characters.
func validEdgeMarker(marker string) bool {
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return false
	}

	for index := 0; index < len(marker); index++ {
		character := marker[index]
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-' || character == '_' || character == '.':
		default:
			return false
		}
	}

	return true
}

// singleRealIP returns the single parseable X-Real-IP address, or ok=false when
// the header is absent, duplicated, or not a valid IP.
func singleRealIP(header http.Header) (string, bool) {
	values := header.Values("X-Real-IP")
	if len(values) != 1 {
		return "", false
	}

	ip := net.ParseIP(strings.TrimSpace(values[0]))
	if ip == nil {
		return "", false
	}

	return ip.String(), true
}
