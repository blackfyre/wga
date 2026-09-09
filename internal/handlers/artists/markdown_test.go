package artists

import (
	"testing"
)

func TestGeneratedMarkdownPath(t *testing.T) {
	if got := generatedMarkdownPath("artists", "synthetic-artist-artistone000001"); got != "/agents/artists/artistone000001.md" {
		t.Fatalf("generatedMarkdownPath = %q", got)
	}
}
