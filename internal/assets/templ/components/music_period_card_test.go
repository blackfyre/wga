package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMusicPeriodPlayerCardRetainsOrdinaryNamedLinkFallback(t *testing.T) {
	var output bytes.Buffer
	playerURL := "/player?song=72d6bb922f76aea"
	if err := MusicPeriodPlayerCard("72d6bb922f76aea", "Fantasia", playerURL).Render(context.Background(), &output); err != nil {
		t.Fatalf("render player card: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`<a href="/player?`,
		`target="wga-period-music"`,
		`data-wga-music="72d6bb922f76aea"`,
		`data-wga-music-control`,
		`data-wga-music-state="idle"`,
		`role="status"`,
		`data-wga-music-blocked`,
		`hidden`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered card does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{"onclick=", "hx-", "autoplay", `rel="noopener"`, `rel="noreferrer"`} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("rendered card must not contain %q", forbidden)
		}
	}
	if strings.Contains(rendered, "aria-pressed") {
		t.Error("ordinary fallback link must not use button-only aria-pressed state")
	}
}
