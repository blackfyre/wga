package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/url"
	"sync/atomic"
)

// EntryScriptPath is the stable path of the browser entry module.
const EntryScriptPath = "/assets/js/app.js"

var entryScriptURL atomic.Value

// ConfigureEntryScript versions the entry module URL with a digest of the
// served js/app.js. A new release then changes the URL the layout references,
// so a cached entry module from an earlier release is never reused, whatever
// browser TTL an intermediary applies. A missing file leaves the URL
// unversioned.
func ConfigureEntryScript(public fs.FS) {
	content, err := fs.ReadFile(public, "js/app.js")
	if err != nil {
		entryScriptURL.Store(EntryScriptPath)
		return
	}
	digest := sha256.Sum256(content)
	entryScriptURL.Store(EntryScriptPath + "?v=" + url.QueryEscape(hex.EncodeToString(digest[:6])))
}

// EntryScriptURL returns the layout's entry module URL.
func EntryScriptURL() string {
	if value, ok := entryScriptURL.Load().(string); ok {
		return value
	}
	return EntryScriptPath
}
