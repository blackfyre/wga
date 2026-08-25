package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/components"
)

func TestPostcardComposeAndConfirmationProgressivelyEnhance(t *testing.T) {
	var compose bytes.Buffer
	if err := PostcardComposePage(PostcardComposeView{ImageID: "artwork-id", Image: "/image.jpg", Title: "Work", Author: "Artist"}).Render(context.Background(), &compose); err != nil {
		t.Fatal(err)
	}
	html := compose.String()
	for _, fragment := range []string{`action="/postcard"`, `method="post"`, `hx-post="/postcard"`, `name="recipient"`, `maxlength="300"`, `name="include_music"`, `name="name"`, `name="email"`} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("compose missing %s", fragment)
		}
	}
	var confirmation bytes.Buffer
	if err := PostcardConfirmationPage(PostcardConfirmationView{MaskedRecipient: "r••••@example.test", ViewURL: "/postcard?token=opaque", Expires: "22 September 2026"}).Render(context.Background(), &confirmation); err != nil {
		t.Fatal(err)
	}
	output := confirmation.String()
	if !strings.Contains(output, "Postcard queued") || !strings.Contains(output, "r••••@example.test") || !strings.Contains(output, "token=opaque") {
		t.Fatal("confirmation omits queued state, masked recipient, or recipient URL")
	}
}

func TestPostcardRecipientRendersMusicOnlyWithMatchingPublishedSong(t *testing.T) {
	var output bytes.Buffer
	if err := PostcardPage(PostcardView{Title: "Work", Author: "Artist", Image: "/image.jpg", SenderName: "Sender", Message: "Hello"}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	html := output.String()
	if strings.Contains(html, "PERIOD MUSIC") || strings.Contains(html, "Period music was included") {
		t.Fatal("recipient must not render or claim music without a matching published song")
	}

	var withMusic bytes.Buffer
	if err := PostcardPage(PostcardView{Title: "Work", Author: "Artist", Image: "/image.jpg", SenderName: "Sender", Message: "Hello", Music: components.MusicPeriodCard{SongID: "song1234567890a", Piece: "Fantasia", PlayerURL: "/player?song=song1234567890a"}}).Render(context.Background(), &withMusic); err != nil {
		t.Fatal(err)
	}
	musicHTML := withMusic.String()
	for _, expected := range []string{"PERIOD MUSIC", `target="wga-period-music"`, `href="/player?song=song1234567890a"`} {
		if !strings.Contains(musicHTML, expected) {
			t.Fatalf("music card missing %s", expected)
		}
	}
	if strings.Contains(musicHTML, "autoplay") {
		t.Fatal("recipient music must not autoplay")
	}
}
