package repositories

import (
	"sync/atomic"

	"github.com/pocketbase/pocketbase/core"
)

const artworkCatalogueRevisionKey = "artworks:catalogue:revision"

func artworkCatalogueRevisionCounter(app core.App) *atomic.Uint64 {
	return app.Store().GetOrSet(artworkCatalogueRevisionKey, func() any {
		return &atomic.Uint64{}
	}).(*atomic.Uint64)
}

// ArtworkCatalogueRevision returns the process-local artwork mutation revision.
func ArtworkCatalogueRevision(app core.App) uint64 {
	return artworkCatalogueRevisionCounter(app).Load()
}

// AdvanceArtworkCatalogueRevision marks page selections made before an artwork
// mutation as candidates for revalidation.
func AdvanceArtworkCatalogueRevision(app core.App) {
	artworkCatalogueRevisionCounter(app).Add(1)
}
