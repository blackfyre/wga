package handlers

import (
	"github.com/blackfyre/wga/handlers/artist"
	"github.com/blackfyre/wga/handlers/artists"
	"github.com/blackfyre/wga/handlers/artworks"
	"github.com/blackfyre/wga/handlers/dual"
	"github.com/blackfyre/wga/handlers/feedback"
	"github.com/blackfyre/wga/handlers/inspire"
	"github.com/blackfyre/wga/handlers/postcards"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
)

// RegisterHandlers registers all the handlers for the application.
// It takes a pointer to a PocketBase instance and initializes the cache.
// The cache is used to store frequently accessed data for faster access.
// The cache is automatically cleaned up every 30 minutes.
func RegisterHandlers(app *pocketbase.PocketBase) {

	p := bluemonday.NewPolicy()

	feedback.RegisterHandlers(app)
	registerMusicHandlers(app)
	registerGuestbookHandlers(app)
	artist.RegisterHandlers(app)
	artists.RegisterHandlers(app)
	postcards.RegisterPostcardHandlers(app, p)
	registerContributors(app)
	registerStatic(app)
	artworks.RegisterArtworksHandlers(app)
	inspire.RegisterHandlers(app)
	registerHome(app)
	dual.RegisterHandlers(app)
}
