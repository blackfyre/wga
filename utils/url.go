package utils

import (
	"os"
	"strings"
)

func AssetUrl(path string) string {

	protocol := os.Getenv("WGA_PROTOCOL")
	hostname := os.Getenv("WGA_HOSTNAME")

	// if the path beings with a slash, remove it
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return protocol + "://" + hostname + path
}

// ExtractIdFromString extracts the ID from a string.
func ExtractIdFromString(s string) string {
	parts := strings.Split(s, "-")
	return parts[len(parts)-1]
}
