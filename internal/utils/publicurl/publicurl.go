package publicurl

import "github.com/blackfyre/wga/internal/config"

var configured config.PublicURL

// Configure sets the public URL used to resolve application asset URLs.
func Configure(value config.PublicURL) {
	configured = value
}

// Resolve returns an absolute public URL for path.
func Resolve(path string) string {
	return configured.Resolve(path)
}
