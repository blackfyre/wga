package utils

import (
	"regexp"
	"testing"
	"testing/fstest"
)

func TestConfigureEntryScriptVersionsByContent(t *testing.T) {
	t.Cleanup(func() { entryScriptURL.Store(EntryScriptPath) })

	ConfigureEntryScript(fstest.MapFS{"js/app.js": {Data: []byte("release one")}})
	first := EntryScriptURL()
	if !regexp.MustCompile(`^/assets/js/app\.js\?v=[0-9a-f]{12}$`).MatchString(first) {
		t.Fatalf("EntryScriptURL() = %q", first)
	}

	ConfigureEntryScript(fstest.MapFS{"js/app.js": {Data: []byte("release one")}})
	if got := EntryScriptURL(); got != first {
		t.Fatalf("same content changed URL: %q then %q", first, got)
	}

	ConfigureEntryScript(fstest.MapFS{"js/app.js": {Data: []byte("release two")}})
	if got := EntryScriptURL(); got == first {
		t.Fatalf("changed content kept URL %q", got)
	}

	ConfigureEntryScript(fstest.MapFS{})
	if got := EntryScriptURL(); got != EntryScriptPath {
		t.Fatalf("missing entry URL = %q, want %q", got, EntryScriptPath)
	}
}
