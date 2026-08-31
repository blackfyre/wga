package components

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"testing"
)

func TestKeyboardLayerRendersCompleteScreenRegistry(t *testing.T) {
	expected := []KeyboardScreen{
		{Key: "O", Num: "00", Label: "HOME", Href: "/"},
		{Key: "A", Num: "01", Label: "ARTIST INDEX", Href: "/artists"},
		{Key: "P", Num: "04", Label: "POSTCARD SERVICE", Href: "/postcard"},
		{Key: "E", Num: "06", Label: "GENERAL CONTENT", Href: "/pages/privacy-policy"},
		{Key: "I", Num: "07", Label: "INSPIRATION", Href: "/inspire"},
		{Key: "S", Num: "08", Label: "SEARCH RESULTS", Href: "/search"},
		{Key: "D", Num: "09", Label: "DUAL MODE", Href: "/dual-mode"},
		{Key: "G", Num: "10", Label: "GUESTBOOK", Href: "/guestbook"},
		{Key: "C", Num: "11", Label: "CONTRIBUTORS", Href: "/contributors"},
		{Key: "W", Num: "12", Label: "ARTWORK SEARCH", Href: "/artworks"},
		{Key: "Y", Num: "13", Label: "GLOSSARY", Href: "/glossary"},
		{Key: "B", Num: "14", Label: "ABOUT THE COLLECTION", Href: "/pages/about"},
		{Key: "N", Num: "15", Label: "STATISTICS", Href: "/statistics"},
		{Key: "V", Num: "16", Label: "VISITOR ITINERARIES", Href: "/itineraries"},
		{Key: "R", Num: "17", Label: "ITINERARY BUILDER", Href: "/itineraries/new"},
		{Key: "M", Num: "19", Label: "TIMELINE EXPLORER", Href: "/timeline"},
		{Key: "U", Num: "20", Label: "GUIDED TOURS", Href: "/tours"},
	}

	if len(keyboardScreens) != len(expected) {
		t.Fatalf("expected %d screens, got %d", len(expected), len(keyboardScreens))
	}
	for index, screen := range expected {
		if keyboardScreens[index] != screen {
			t.Fatalf("screen %d: expected %+v, got %+v", index, screen, keyboardScreens[index])
		}
	}

	keys := map[string]bool{}
	numbers := map[string]bool{}
	routes := map[string]bool{}
	for _, screen := range keyboardScreens {
		if !regexp.MustCompile(`^[0-9]{2}$`).MatchString(screen.Num) {
			t.Errorf("screen %q has non-two-digit number %q", screen.Label, screen.Num)
		}
		if keys[screen.Key] || numbers[screen.Num] || routes[screen.Href] {
			t.Errorf("screen %q duplicates a key, number, or route", screen.Label)
		}
		if strings.ContainsAny(screen.Key, "JKHL") {
			t.Errorf("screen %q uses reserved movement key %q", screen.Label, screen.Key)
		}
		keys[screen.Key] = true
		numbers[screen.Num] = true
		routes[screen.Href] = true
	}

	var output bytes.Buffer
	if err := KeyboardLayer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render keyboard layer: %v", err)
	}
	rendered := output.String()
	payload := keyboardPayload(t, rendered)
	if len(payload) != len(keyboardScreens) {
		t.Fatalf("expected %d JSON screens, got %d", len(keyboardScreens), len(payload))
	}
	if count := strings.Count(rendered, `data-kbd-item="section"`); count != len(keyboardScreens) {
		t.Fatalf("expected %d palette screens, got %d", len(keyboardScreens), count)
	}
	helpScreens := regexp.MustCompile(`<kbd>([A-Z])</kbd> ([0-9]{2}) · ([A-Z ]+)`).FindAllStringSubmatch(rendered, -1)
	if len(helpScreens) != len(keyboardScreens) {
		t.Fatalf("expected %d help screens, got %d", len(keyboardScreens), len(helpScreens))
	}
	for index, screen := range keyboardScreens {
		if payload[index] != screen {
			t.Errorf("JSON screen %d: expected %+v, got %+v", index, screen, payload[index])
		}
		palette := `href="` + screen.Href + `" data-kbd-item="section" data-kbd-key="` + screen.Key + `" data-kbd-num="` + screen.Num + `" data-kbd-label="` + screen.Label + `" data-kbd-href="` + screen.Href + `"`
		if !strings.Contains(rendered, palette) {
			t.Errorf("palette does not render %+v", screen)
		}
		help := helpScreens[index]
		if help[1] != screen.Key || help[2] != screen.Num || help[3] != screen.Label {
			t.Errorf("help screen %d: expected %+v, got %q %q %q", index, screen, help[1], help[2], help[3])
		}
	}
	for _, required := range []string{
		`id="keyboard-palette" class="wga-kbd-dialog" aria-labelledby="keyboard-palette-title"`,
		`id="keyboard-help" class="wga-kbd-dialog" aria-labelledby="keyboard-help-title"`,
		`aria-label="Close Go to"`,
		`aria-label="Close shortcuts"`,
		`data-kbd-modifier`,
		`FOCUS VISIBLE SEARCH; OPENS MOBILE NAVIGATION FIRST`,
		`CLEAR A MARKER, BLUR SEARCH, OR CLOSE A SAFE TRANSIENT SURFACE`,
		`GUIDED TOURS: WHILE READING A TOUR`,
		`TURNS TO THE PREVIOUS PAGE`,
		`TURNS TO THE NEXT PAGE`,
		`NOT A GLOBAL SHORTCUT`,
		`IGNORED IN EDITABLE CONTROLS`,
		`WHILE THE ARTWORK VIEWER OWNS THE KEYBOARD`,
		`ARTWORK VIEWER: OPEN AN ARTWORK IMAGE, THEN USE ITS VISIBLE CLOSE CONTROL OR`,
	} {
		if !strings.Contains(strings.ToUpper(rendered), strings.ToUpper(required)) {
			t.Errorf("keyboard layer is missing %q", required)
		}
	}
}

func TestKeyboardHelpDocumentsTourPageTurns(t *testing.T) {
	var output bytes.Buffer
	if err := KeyboardLayer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render keyboard layer: %v", err)
	}
	rendered := output.String()
	upper := strings.ToUpper(rendered)

	for _, required := range []string{
		`GUIDED TOURS: WHILE READING A TOUR`,
		`TURNS TO THE PREVIOUS PAGE`,
		`TURNS TO THE NEXT PAGE`,
		`NOT A GLOBAL SHORTCUT`,
		`IGNORED IN EDITABLE CONTROLS`,
		`WHILE THE ARTWORK VIEWER OWNS THE KEYBOARD`,
	} {
		if !strings.Contains(upper, required) {
			t.Errorf("keyboard help is missing %q", required)
		}
	}

	for _, key := range []string{`<kbd>←</kbd>`, `<kbd>→</kbd>`} {
		if !strings.Contains(rendered, key) {
			t.Errorf("keyboard help is missing the tour page-turn key %q", key)
		}
	}

	if strings.Contains(upper, "NO TOUR PAGE-TURN SHORTCUT") {
		t.Errorf("keyboard help still contains the superseded statement %q", "no tour page-turn shortcut is currently available")
	}
}

func TestKeyboardLayerExposesHelpFromKeyboardBar(t *testing.T) {
	var output bytes.Buffer
	if err := KeyboardLayer().Render(context.Background(), &output); err != nil {
		t.Fatalf("render keyboard layer: %v", err)
	}

	rendered := output.String()
	if count := strings.Count(rendered, `data-keyboard-help`); count != 1 {
		t.Fatalf("expected one keyboard-bar help control, got %d", count)
	}
	if !strings.Contains(rendered, `? ALL KEYS`) {
		t.Fatal("expected the keyboard bar help control")
	}
}

func keyboardPayload(t *testing.T, rendered string) []KeyboardScreen {
	t.Helper()
	matches := regexp.MustCompile(`data-json="([^"]*)"`).FindStringSubmatch(rendered)
	if len(matches) != 2 {
		t.Fatal("keyboard screen JSON payload is missing")
	}

	var payload []KeyboardScreen
	if err := json.Unmarshal([]byte(html.UnescapeString(matches[1])), &payload); err != nil {
		t.Fatalf("decode keyboard screen JSON: %v", err)
	}

	return payload
}
