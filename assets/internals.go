package assets

import (
	"embed"
)

//go:embed "reference/*" "views/*"
var InternalFiles embed.FS
