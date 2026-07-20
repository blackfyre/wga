package utils

import (
	"strings"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/utils/publicurl"
)

func ConfigurePublicURL(value config.PublicURL) {
	publicurl.Configure(value)
}

func AssetUrl(path string) string {
	return publicurl.Resolve(path)
}

// ExtractIdFromString extracts the ID from a string.
func ExtractIdFromString(s string) string {
	parts := strings.Split(s, "-")
	return parts[len(parts)-1]
}
