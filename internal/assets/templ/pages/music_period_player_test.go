package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMusicPeriodPlayerPageRendersNativeNoAutoplayControls(t *testing.T) {
	var output bytes.Buffer
	view := MusicPeriodPlayerView{
		SongID:     "72d6bb922f76aea",
		Composer:   "Sweelinck",
		Piece:      "Sweelinck — Fantasia chromatica",
		Source:     "/api/files/music_song/song1234567890a/fantasia.mp3",
		WindowName: "wga-period-music",
	}
	if err := MusicPeriodPlayerPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render music player: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`data-wga-music-player`,
		`data-wga-music-song="72d6bb922f76aea"`,
		`data-wga-music-piece="Sweelinck — Fantasia chromatica"`,
		`data-wga-music-window="wga-period-music"`,
		`controls`,
		`preload="metadata"`,
		`src="/api/files/music_song/song1234567890a/fantasia.mp3"`,
		"Sweelinck",
		"Playback starts only when you use the audio controls.",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered player does not contain %q", expected)
		}
	}
	if strings.Contains(strings.ToLower(rendered), "autoplay") {
		t.Error("rendered player must not contain autoplay")
	}
}
