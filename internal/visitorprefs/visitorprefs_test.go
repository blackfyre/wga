package visitorprefs

import (
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"
)

func TestFallbackView(t *testing.T) {
	consent := neturl.PathEscape(`{"categories":["necessary","preferences"]}`)
	for _, test := range []struct {
		name    string
		consent string
		prefs   string
		want    string
	}{
		{name: "nothing stored", consent: consent, want: ViewGrid},
		{name: "remembered list", consent: consent, prefs: neturl.PathEscape(`{"view":"list"}`), want: ViewList},
		{name: "remembered grid", consent: consent, prefs: neturl.PathEscape(`{"view":"grid"}`), want: ViewGrid},
		{name: "no consent", prefs: neturl.PathEscape(`{"view":"list"}`), want: ViewGrid},
		{name: "essential consent only", consent: neturl.PathEscape(`{"categories":["necessary"]}`), prefs: neturl.PathEscape(`{"view":"list"}`), want: ViewGrid},
		{name: "invalid view", consent: consent, prefs: neturl.PathEscape(`{"view":"table"}`), want: ViewGrid},
		{name: "wrong type", consent: consent, prefs: neturl.PathEscape(`{"view":1}`), want: ViewGrid},
		{name: "malformed cookie", consent: consent, prefs: "%zz", want: ViewGrid},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/artists", nil)
			if test.consent != "" {
				request.AddCookie(&http.Cookie{Name: ConsentCookieName, Value: test.consent})
			}
			if test.prefs != "" {
				request.AddCookie(&http.Cookie{Name: CookieName, Value: test.prefs})
			}
			if got := FallbackView(request); got != test.want {
				t.Fatalf("FallbackView() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEmitView(t *testing.T) {
	for _, test := range []struct {
		view, fallback string
		want           bool
	}{
		{ViewList, ViewGrid, true},
		{ViewList, ViewList, true},
		{ViewGrid, ViewGrid, false},
		{ViewGrid, "", false},
		{ViewGrid, ViewList, true},
		{"", ViewList, false},
	} {
		if got := EmitView(test.view, test.fallback); got != test.want {
			t.Errorf("EmitView(%q, %q) = %v, want %v", test.view, test.fallback, got, test.want)
		}
	}
}
