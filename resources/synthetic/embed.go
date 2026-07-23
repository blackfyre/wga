package synthetic

import "embed"

// Files contains the migration bootstrap database with its storage assets.
//
//go:embed wga-test.sqlite storage
var Files embed.FS
