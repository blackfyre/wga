package postcards

import (
	"strings"
	"testing"
)

func TestSanitiseMessageAllowsOnlyPostcardFormatting(t *testing.T) {
	message := SanitiseMessage(`<p class="lead" onclick="alert(1)">Hello <strong>wide</strong> <b style="color:red">world</b>.</p><ul data-x="1"><li>One</li></ul><a href="https://example.test">link</a><script>alert(2)</script>`)

	for _, expected := range []string{"<p>Hello wide <b>world</b>.</p>", "<ul><li>One</li></ul>", "link"} {
		if !strings.Contains(message.HTML(), expected) {
			t.Errorf("sanitised HTML missing %q: %s", expected, message.HTML())
		}
	}
	for _, forbidden := range []string{"class=", "onclick=", "style=", "data-x=", "<strong", "<a", "<script", "alert(2)"} {
		if strings.Contains(message.HTML(), forbidden) {
			t.Errorf("sanitised HTML contains %q: %s", forbidden, message.HTML())
		}
	}
}

func TestSanitiseMessageCountsVisibleUnicodeText(t *testing.T) {
	message := SanitiseMessage("<p>Árvíz <b>😀</b> &amp; tea</p>")
	if message.Text() != "Árvíz 😀 & tea" {
		t.Fatalf("text = %q, want %q", message.Text(), "Árvíz 😀 & tea")
	}
	if got := message.RuneCount(); got != 13 {
		t.Fatalf("rune count = %d, want 13", got)
	}
}

func TestValidateQueueInputCanonicalisesMessageBeforePersistence(t *testing.T) {
	input := QueueInput{
		SenderName:  "Sender",
		SenderEmail: "sender@example.test",
		ImageID:     "artwork-id",
		Message:     `<p onclick="alert(1)">Hello <i>there</i>.</p>`,
	}
	if err := validateQueueInput(&input); err != nil {
		t.Fatal(err)
	}
	if input.Message != "<p>Hello <i>there</i>.</p>" {
		t.Fatalf("message = %q", input.Message)
	}
}
