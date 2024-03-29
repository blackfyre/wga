package sitemap

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/sabloger/sitemap-generator/smg"
)

func GenerateSiteMap(app *pocketbase.PocketBase) {
	isDevelopment := os.Getenv("WGA_ENV") == "development"

	index := smg.NewSitemapIndex(isDevelopment)
	index.SetSitemapIndexName("web_gallery_of_art")
	index.SetHostname(os.Getenv("WGA_PROTOCOL") + "://" + os.Getenv("WGA_HOSTNAME"))
	index.SetOutputPath("./wga_sitemap")
	index.SetServerURI("/sitemaps/")

	index.SetCompress(!isDevelopment)

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

func generateArtistMap(app *pocketbase.PocketBase, index *smg.SitemapIndex) {
	now := time.Now().UTC()
	artistSitemap := index.NewSitemap()
	artistSitemap.SetName("artists")
	artistSitemap.SetLastMod(&now)

	records, err := app.Dao().FindRecordsByFilter(
		"artists",
		"published = true",
		"+name",
		0,
		0,
	)

	if err != nil {
		app.Logger().Error("Error fetching artists for sitemap", err)
	}

	for _, m := range records {

		updatedAtTime := m.GetUpdated().Time()

		err := artistSitemap.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/artist/%s-%s", m.GetString("slug"), m.GetId()),
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
	now := time.Now().UTC()
	artistSitemap := index.NewSitemap()
	artistSitemap.SetName("artworks")
	artistSitemap.SetLastMod(&now)

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

		err := artistSitemap.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/artist/%s-%s/%s-%s", author.GetString("slug"), author.GetId(), utils.Slugify(m.GetString("title")), m.GetId()),
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
