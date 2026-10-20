package utils

import "github.com/pocketbase/pocketbase/core"

// KeepResponsePrivate marks a response rendered from visitor state, such as a
// remembered preference cookie, as never shareable. The public catalogue cache
// middleware leaves an explicit Cache-Control header unchanged.
func KeepResponsePrivate(c *core.RequestEvent) {
	c.Response.Header().Set("Cache-Control", "private, no-store")
}
