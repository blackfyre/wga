package requestprotection

import (
	"net/http"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   Profile
	}{
		{name: "artist search GET", method: http.MethodGet, path: "/artists", want: ProfileSearch},
		{name: "artist search HEAD", method: http.MethodHead, path: "/artists", want: ProfileSearch},
		{name: "artwork search", method: http.MethodGet, path: "/artworks", want: ProfileSearch},
		{name: "artwork results", method: http.MethodGet, path: "/artworks/results", want: ProfileSearch},
		{name: "dual fragment", method: http.MethodGet, path: "/dual-mode", want: ProfileFragment},
		{name: "artist detail", method: http.MethodGet, path: "/artists/claude-monet", want: ProfileDetail},
		{name: "artwork detail", method: http.MethodGet, path: "/artists/claude-monet/water-lilies", want: ProfileDetail},
		{name: "agent artist detail", method: http.MethodGet, path: "/agents/artists/artist-id.md", want: ProfileDetail},
		{name: "agent artwork detail", method: http.MethodHead, path: "/agents/artworks/artwork-id.md", want: ProfileDetail},
		{name: "health", method: http.MethodGet, path: "/health", want: ProfileExempt},
		{name: "static asset", method: http.MethodHead, path: "/assets/css/style.css", want: ProfileExempt},
		{name: "sitemap index", method: http.MethodGet, path: "/sitemap.xml", want: ProfileExempt},
		{name: "sitemap stylesheet", method: http.MethodGet, path: "/sitemap.xsl", want: ProfileExempt},
		{name: "sitemap shard", method: http.MethodGet, path: "/sitemap/artists-1.xml", want: ProfileExempt},
		{name: "robots", method: http.MethodGet, path: "/robots.txt", want: ProfileExempt},
		{name: "admin root", method: http.MethodGet, path: "/_/", want: ProfileExempt},
		{name: "admin asset", method: http.MethodGet, path: "/_/assets/app.js", want: ProfileExempt},
		{name: "write method", method: http.MethodPost, path: "/artists", want: ProfileUnclassified},
		{name: "dual lookup remains unclassified", method: http.MethodGet, path: "/dual-mode/lookup", want: ProfileUnclassified},
		{name: "selection remains unclassified", method: http.MethodGet, path: "/artists/name/selections/id", want: ProfileUnclassified},
		{name: "unknown route", method: http.MethodGet, path: "/future-route", want: ProfileUnclassified},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Classify(test.method, test.path); got != test.want {
				t.Fatalf("Classify(%q, %q) = %q; want %q", test.method, test.path, got, test.want)
			}
		})
	}
}

func TestCanonicalHostAllows(t *testing.T) {
	policy, err := NewCanonicalHost("https://beta.wga.hu")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		profile Profile
		host    string
		want    bool
	}{
		{name: "canonical host", profile: ProfileSearch, host: "beta.wga.hu", want: true},
		{name: "host case", profile: ProfileFragment, host: "BETA.WGA.HU", want: true},
		{name: "trailing dot", profile: ProfileDetail, host: "beta.wga.hu.", want: true},
		{name: "explicit default port", profile: ProfileDetail, host: "beta.wga.hu:443", want: true},
		{name: "wrong port", profile: ProfileSearch, host: "beta.wga.hu:80", want: false},
		{name: "deployment host protected", profile: ProfileSearch, host: "wga-production.up.railway.app", want: false},
		{name: "deployment host exempt", profile: ProfileExempt, host: "wga-production.up.railway.app", want: true},
		{name: "deployment host unclassified", profile: ProfileUnclassified, host: "wga-production.up.railway.app", want: true},
		{name: "empty protected host", profile: ProfileDetail, host: "", want: false},
		{name: "userinfo authority", profile: ProfileDetail, host: "user@beta.wga.hu", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := policy.Allows(test.profile, test.host); got != test.want {
				t.Fatalf("Allows(%q, %q) = %t; want %t", test.profile, test.host, got, test.want)
			}
		})
	}
}

func TestCanonicalHostHonoursConfiguredPort(t *testing.T) {
	policy, err := NewCanonicalHost("http://localhost:8090")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		host string
		want bool
	}{
		{host: "localhost:8090", want: true},
		{host: "LOCALHOST:8090", want: true},
		{host: "localhost", want: false},
		{host: "localhost:80", want: false},
	} {
		if got := policy.Allows(ProfileSearch, test.host); got != test.want {
			t.Errorf("Allows(search, %q) = %t; want %t", test.host, got, test.want)
		}
	}
}

func TestNewCanonicalHostRejectsInvalidPublicURL(t *testing.T) {
	for _, publicURL := range []string{"", "beta.wga.hu", "ftp://beta.wga.hu", "https://user@beta.wga.hu", "https://beta.wga.hu/path"} {
		t.Run(publicURL, func(t *testing.T) {
			if _, err := NewCanonicalHost(publicURL); err == nil {
				t.Fatalf("NewCanonicalHost(%q) succeeded; want error", publicURL)
			}
		})
	}
}
