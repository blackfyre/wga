package sitemap

import (
	"fmt"
    "github.com/blackfyre/wga/utils/url"
    "log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"github.com/sabloger/sitemap-generator/smg"
)

// setupSitemapIndex initializes and configures a SitemapIndex object.
// It sets the SitemapIndex name, hostname, output path, server URI, and compression settings.
// The SitemapIndex object is then returned.
func setupSitemapIndex() *smg.SitemapIndex {
	isDevelopment := os.Getenv("WGA_ENV") == "development"
	index := smg.NewSitemapIndex(isDevelopment)
	index.SetSitemapIndexName("web_gallery_of_art")
	index.SetHostname(os.Getenv("WGA_PROTOCOL") + "://" + os.Getenv("WGA_HOSTNAME"))
	index.SetOutputPath("./wga_sitemap")
	index.SetServerURI("/sitemaps/")

	index.SetCompress(!isDevelopment)

	return index
}

func GenerateSiteMap(app *pocketbase.PocketBase) {

	index := setupSitemapIndex()

	generateArtistMap(app, index)
	generateArtworksMap(app, index)

	// Save func saves the xml files and returns more than one filename in case of split large files.
	filenames, err := index.Save()
	if err != nil {
		app.Logger().Error("Unable to Save Sitemap:", err)
		return
	}
	for _, filename := range filenames {
		app.Logger().Info(fmt.Sprintf("Sitemap saved to %c", filename))
	}
}

func setupSitemap(name string, index *smg.SitemapIndex) *smg.Sitemap {
	now := time.Now().UTC()
	sitemap := index.NewSitemap()
	sitemap.SetName(name)
	sitemap.SetLastMod(&now)
	return sitemap
}

func fetchArtistsForSitemap(app *pocketbase.PocketBase) ([]*models.Record, error) {
	return app.Dao().FindRecordsByFilter(
		"artists",
		"published = true",
		"+name",
		0,
		0,
	)
}

func generateArtistMap(app *pocketbase.PocketBase, index *smg.SitemapIndex) {
	sitemap := setupSitemap("artists", index)

	records, err := fetchArtistsForSitemap(app)

	if err != nil {
		app.Logger().Error("Error fetching artists for sitemap", err)
	}

	for _, m := range records {

		updatedAtTime := m.GetUpdated().Time()

		err := sitemap.Add(&smg.SitemapLoc{
			Loc:        url.GenerateArtistUrlFromRecord(m),
			LastMod:    &updatedAtTime,
			ChangeFreq: smg.Monthly,
			Priority:   0.8,
		})

		if err != nil {
			log.Fatal("Unable to Save Sitemap:", err)
		}
	}
}

func generateArtworksMap(app *pocketbase.PocketBase, index *smg.SitemapIndex) {
	sitemap := setupSitemap("artworks", index)

	records, err := app.Dao().FindRecordsByFilter(
		"artworks",
		"published = true",
		"+title",
		0,
		0,
	)

	if err != nil {
		log.Fatal("Unable to Save Sitemap:", err)
	}

	for _, m := range records {

		if errs := app.Dao().ExpandRecord(m, []string{"author"}, nil); len(errs) > 0 {
			app.Logger().Error("Error expanding record", "err", errs)
			// we should log the failed items, still waiting for pb logs
			continue // we're skipping failed items
		}

		author := m.ExpandedOne("author")

		if author == nil {
			//every item in the db should have an author
			// log those items which don't for fixing
			app.Logger().Error("Error expanding record, no author found", "id", m.GetId())
			continue
		}

		updatedAtTime := m.GetUpdated().Time()

		err := sitemap.Add(&smg.SitemapLoc{
			Loc: url.GenerateArtworkUrl(url.ArtworkUrlDTO{
				ArtistName: author.GetString("name"),
				ArtistId: author.GetId(),
				ArtworkId: m.GetId(),
				ArtworkTitle: m.GetString("title"),
			}),
			LastMod:    &updatedAtTime,
			ChangeFreq: smg.Monthly,
			Priority:   0.8,
		})

		if err != nil {
			app.Logger().Error("Unable to Save Sitemap:", err)
			return
		}
	}
}
