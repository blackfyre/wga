package handlers

import (
	"github.com/blackfyre/wga/internal/handlers/artists"
	"github.com/blackfyre/wga/internal/handlers/artworks"
	"github.com/blackfyre/wga/internal/handlers/contributors"
	"github.com/blackfyre/wga/internal/handlers/dual"
	"github.com/blackfyre/wga/internal/handlers/feedback"
	"github.com/blackfyre/wga/internal/handlers/guestbook"
	"github.com/blackfyre/wga/internal/handlers/inspire"
	"github.com/blackfyre/wga/internal/handlers/landing"
	"github.com/blackfyre/wga/internal/handlers/static"
	"github.com/blackfyre/wga/internal/handlers/statistics"

	"github.com/blackfyre/wga/internal/handlers/postcards"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase"
)

// RegisterHandlers registers all the handlers for the application.
// It takes a pointer to a PocketBase instance and initializes the cache.
// The cache is used to store frequently accessed data for faster access.
// The cache is automatically cleaned up every 30 minutes.
func RegisterHandlers(app *pocketbase.PocketBase) {

	app.Logger().Debug("Registering route handlers...")
	p := bluemonday.NewPolicy()

	feedback.RegisterHandlers(app)
	// registerMusicHandlers(app)
	guestbook.RegisterHandlers(app)
	artists.RegisterHandlers(app)
	postcards.RegisterPostcardHandlers(app, p)
	contributors.RegisterHandlers(app)
	static.RegisterHandlers(app)
	artworks.RegisterArtworksHandlers(app)
	inspire.RegisterHandlers(app)
	landing.RegisterHandlers(app)
	statistics.RegisterHandlers(app)
	dual.RegisterHandlers(app)
}
