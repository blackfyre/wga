package utils

import (
	"strings"

	"github.com/blackfyre/wga/internal/config"
)

var publicURL config.PublicURL

func ConfigurePublicURL(value config.PublicURL) {
	publicURL = value
}

func AssetUrl(path string) string {
	return publicURL.Resolve(path)
}

// ExtractIdFromString extracts the ID from a string.
func ExtractIdFromString(s string) string {
	parts := strings.Split(s, "-")
	return parts[len(parts)-1]
}
