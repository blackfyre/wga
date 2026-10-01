package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFooterBottomRowOffersBackToTopBesidePreferences(t *testing.T) {
	var output bytes.Buffer
	if err := Footer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render footer: %v", err)
	}
	rendered := output.String()

	start := strings.Index(rendered, `<a href="#top" data-wga-back-to-top`)
	if start < 0 {
		t.Fatal("footer missing the plain #top link")
	}
	end := strings.Index(rendered[start:], "</a>")
	if end < 0 {
		t.Fatal("back-to-top link is not closed")
	}
	link := rendered[start : start+end]
	if !strings.HasSuffix(link, ">↑ BACK TO TOP") {
		t.Errorf("back-to-top link must read ↑ BACK TO TOP, got %q", link)
	}
	if strings.Contains(link, "hx-") {
		t.Error("back-to-top must be an ordinary in-page link")
	}
	if preferences := strings.Index(rendered, "data-wga-preferences-control"); preferences < start {
		t.Error("back-to-top link must sit beside the preferences control in the bottom row")
	}
}
