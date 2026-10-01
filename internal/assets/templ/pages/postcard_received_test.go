package pages

import (
	"regexp"
	"strings"
	"testing"
)

func receivedPostcardView() PostcardView {
	return PostcardView{
		Title: "Work", ArtistFilingName: "Artist, Filing", Image: "/image.jpg", SenderName: "Sender", Message: "<p>Hello</p>",
		Technique: "Oil on canvas", Dimensions: "101 x 201 cm", Location: "Mauritshuis, The Hague",
		RecordURL: "/artists/artist-a1/work-w1", ComposeURL: "/postcard/send?awid=w1",
	}
}

func TestPostcardReceivedLinksBackIntoTheCollection(t *testing.T) {
	html := renderPostcard(t, PostcardBlock(receivedPostcardView()))

	for _, pattern := range []string{
		`<h2[^>]*><a href="/artists/artist-a1/work-w1"[^>]*>Work</a></h2>`,
		`<a href="/artists/artist-a1/work-w1"[^>]*>VIEW IN GALLERY →</a>`,
		`<a href="/postcard/send\?awid=w1"[^>]*>SEND YOUR OWN →</a>`,
		`<a href="/artists"[^>]*>BROWSE THE ARCHIVE →</a>`,
	} {
		if !regexp.MustCompile(pattern).MatchString(html) {
			t.Fatalf("received postcard missing link %s", pattern)
		}
	}
	// Card links are full navigations: an un-pushed HTMX request would leave
	// the address bar on the bearer URL and would not work without JavaScript.
	section := html[strings.Index(html, `id="postcard-view"`):]
	if strings.Contains(section, "hx-get") || strings.Contains(section, "hx-post") {
		t.Fatal("received postcard links must be ordinary navigations")
	}
}

func TestPostcardReceivedShowsRecordDetails(t *testing.T) {
	html := renderPostcard(t, PostcardBlock(receivedPostcardView()))

	for _, expected := range []string{
		`<span data-postcard-material>Oil on canvas, 101 x 201 cm</span>`,
		`<span data-postcard-location>Mauritshuis, The Hague</span>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("received postcard missing %s", expected)
		}
	}
}

func TestPostcardReceivedOmitsMissingRecordDetails(t *testing.T) {
	cases := map[string]struct {
		technique, dimensions, location string
		material                        string
	}{
		"no dimensions or location": {technique: "Oil on canvas", material: "Oil on canvas"},
		"dimensions only":           {dimensions: "101 x 201 cm", material: "101 x 201 cm"},
		"nothing":                   {},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			view := receivedPostcardView()
			view.Technique, view.Dimensions, view.Location = tc.technique, tc.dimensions, tc.location
			html := renderPostcard(t, PostcardBlock(view))

			if strings.Contains(html, "data-postcard-location") {
				t.Fatal("received postcard must omit an absent location")
			}
			if tc.material == "" {
				if strings.Contains(html, "data-postcard-material") {
					t.Fatal("received postcard must omit an absent material line")
				}
				if regexp.MustCompile(`Artist, Filing\s*<br>`).MatchString(html) {
					t.Fatal("received postcard must not render an empty detail line")
				}
				return
			}
			if !strings.Contains(html, `<span data-postcard-material>`+tc.material+`</span>`) {
				t.Fatalf("received postcard material line = want %q", tc.material)
			}
		})
	}
}

func TestPostcardComposerCountsCharactersLikeTheDesign(t *testing.T) {
	for _, tc := range []struct {
		name          string
		messageLength int
		counter       string
		overLimit     bool
	}{
		{name: "empty composer", counter: "300 CHARACTERS LEFT"},
		{name: "redisplayed message", messageLength: 11, counter: "289 CHARACTERS LEFT"},
		{name: "one character left", messageLength: 299, counter: "1 CHARACTER LEFT"},
		{name: "at the limit", messageLength: 300, counter: "0 CHARACTERS LEFT"},
		{name: "over the limit", messageLength: 305, counter: "5 CHARACTERS OVER THE LIMIT", overLimit: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := renderPostcard(t, PostcardComposeBlock(PostcardComposeView{ImageID: "artwork-id", Image: "/image.jpg", Title: "Work", ArtistFilingName: "Artist, Filing", Recipients: []string{""}, MessageLength: tc.messageLength}))

			pattern := `<label id="message-label" for="message">MESSAGE</label>\s*<span aria-hidden="true"[^>]*>—</span>\s*<span id="message-count"[^>]*class="([^"]*)"[^>]*>` + regexp.QuoteMeta(tc.counter) + `</span>`
			match := regexp.MustCompile(pattern).FindStringSubmatch(html)
			if match == nil {
				t.Fatalf("composer must read MESSAGE — %s before enhancement", tc.counter)
			}
			if got := strings.Contains(match[1], "text-wga-error"); got != tc.overLimit {
				t.Fatalf("counter error styling = %v, want %v (class %q)", got, tc.overLimit, match[1])
			}
		})
	}
}
