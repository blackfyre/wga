package assets

import (
	"embed"
)

//go:embed "public/*"
var PublicFiles embed.FS
