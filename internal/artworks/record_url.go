package artworks

import (
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

// RecordPath returns the canonical public record path of an artwork under one
// of its authors. The artist segment uses the author's stored slug, which the
// importer disambiguates when two artists' names normalise to the same slug,
// so it must not be rebuilt from the name. The artwork record route redirects
// any other spelling to this path.
func RecordPath(artist *core.Record, artwork *core.Record) string {
	return "/artists/" + utils.GenerateArtistSlug(artist) + "/" + utils.Slugify(artwork.GetString("title")) + "-" + artwork.Id
}
