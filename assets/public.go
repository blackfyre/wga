package assets

import (
	"embed"
)

//go:embed "public/css/*" "public/js/*" "public/images/*" "public/fonts/*" "public/404.html"
var PublicFiles embed.FS
