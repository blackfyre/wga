package synthetic

import "embed"

// Files contains the development and staging seed database with its storage assets.
//
//go:embed wga-test.sqlite storage
var Files embed.FS
