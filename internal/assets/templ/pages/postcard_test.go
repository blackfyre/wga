package pages

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/blackfyre/wga/internal/assets/templ/components"
)

func TestPostcardDialogsCarryHeaderRuleDismissal(t *testing.T) {
	compose := renderPostcard(t, PostcardComposeDialog(PostcardComposeView{ImageID: "artwork-id", Image: "/image.jpg", Title: "Work", ArtistFilingName: "Artist, Filing"}))
	confirmation := renderPostcard(t, PostcardConfirmationDialog(PostcardConfirmationView{MaskedRecipient: "r••••@example.test", ViewURL: "/postcard?token=opaque", Expires: "22 September 2026"}))

	for name, html := range map[string]string{"compose": compose, "confirmation": confirmation} {
		for _, expected := range []string{`data-dialog-close`, `data-dialog-initial-focus`, `aria-label="Close"`, `method="dialog"`, `class="modal-backdrop`, `border-b border-base-content`} {
			if !strings.Contains(html, expected) {
				t.Fatalf("%s dialog missing header-rule dismissal marker %s", name, expected)
			}
		}
		if !strings.Contains(html, "✕") {
			t.Fatalf("%s dialog missing visible ✕ dismissal", name)
		}
	}
}

func TestPostcardDialogsLabelVisibleHeadingsWithUniqueIDs(t *testing.T) {
	compose := renderPostcard(t, PostcardComposeDialog(PostcardComposeView{ImageID: "artwork-id", Image: "/image.jpg", Title: "Work", ArtistFilingName: "Artist, Filing"}))
	confirmation := renderPostcard(t, PostcardConfirmationDialog(PostcardConfirmationView{MaskedRecipient: "r••••@example.test", ViewURL: "/postcard?token=opaque", Expires: "22 September 2026"}))

	titleTag := regexp.MustCompile(`<h1[^>]*>`)
	for name, tc := range map[string]struct {
		html  string
		title string
		id    string
	}{
		"compose":      {compose, "Send a postcard", "postcard-compose-title"},
		"confirmation": {confirmation, "Postcard queued", "postcard-confirmation-title"},
	} {
		if !strings.Contains(tc.html, tc.title) {
			t.Fatalf("%s dialog missing visible heading text %q", name, tc.title)
		}
		tag := titleTag.FindString(tc.html)
		if tag == "" {
			t.Fatalf("%s dialog missing h1 heading", name)
		}
		if !strings.Contains(tag, "data-dialog-title") {
			t.Fatalf("%s heading is not marked data-dialog-title", name)
		}
		if !strings.Contains(tag, `id="`+tc.id+`"`) {
			t.Fatalf("%s heading missing unique id %s", name, tc.id)
		}
		if strings.Count(tc.html, `id="`+tc.id+`"`) != 1 {
			t.Fatalf("%s heading id %s is duplicated", name, tc.id)
		}
	}
}

func renderPostcard(t *testing.T, component templ.Component) string {
	t.Helper()
	var output bytes.Buffer
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatalf("render postcard dialog: %v", err)
	}
	return output.String()
}

func TestPostcardComposeAndConfirmationProgressivelyEnhance(t *testing.T) {
	var compose bytes.Buffer
	if err := PostcardComposePage(PostcardComposeView{ImageID: "artwork-id", Image: "/image.jpg", Title: "Work", ArtistFilingName: "Artist, Filing"}).Render(context.Background(), &compose); err != nil {
		t.Fatal(err)
	}
	html := compose.String()
	for _, fragment := range []string{`action="/postcard"`, `method="post"`, `hx-post="/postcard"`, `name="recipient"`, `maxlength="300"`, `name="include_music"`, `name="name"`, `name="email"`, "Artist, Filing"} {
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
	if err := PostcardPage(PostcardView{Title: "Work", ArtistFilingName: "Artist, Filing", Image: "/image.jpg", SenderName: "Sender", Message: "Hello"}).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	html := output.String()
	if !strings.Contains(html, "Artist, Filing") {
		t.Fatal("recipient must render the artist filing name")
	}
	if strings.Contains(html, "PERIOD MUSIC") || strings.Contains(html, "Period music was included") {
		t.Fatal("recipient must not render or claim music without a matching published song")
	}

	var withMusic bytes.Buffer
	if err := PostcardPage(PostcardView{Title: "Work", ArtistFilingName: "Artist, Filing", Image: "/image.jpg", SenderName: "Sender", Message: "Hello", Music: components.MusicPeriodCard{SongID: "song1234567890a", Piece: "Fantasia", PlayerURL: "/player?song=song1234567890a"}}).Render(context.Background(), &withMusic); err != nil {
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
