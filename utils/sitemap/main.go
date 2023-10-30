package sitemap

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/sabloger/sitemap-generator/smg"
)

func GenerateSiteMap(app *pocketbase.PocketBase) {

	index := smg.NewSitemapIndex(app.IsDebug())
	index.SetSitemapIndexName("web_gallery_of_art")
	index.SetHostname(os.Getenv("WGA_PROTOCOL") + "://" + os.Getenv("WGA_HOSTNAME"))
	index.SetOutputPath("./wga_sitemap")
	index.SetServerURI("/sitemaps/")

	index.SetCompress(!app.IsDebug())

	generateArtistMap(app, index)
	generateArtworksMap(app, index)

	// Save func saves the xml files and returns more than one filename in case of split large files.
	filenames, err := index.Save()
	if err != nil {
		log.Fatal("Unable to Save Sitemap:", err)
	}
	for i, filename := range filenames {
		fmt.Println("file no.", i+1, filename)
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
		log.Fatal("Unable to Save Sitemap:", err)
	}

	for _, m := range records {

		updatedAtTime := m.GetUpdated().Time()

		err := artistSitemap.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/artist/%s", m.GetString("slug")),
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
			fmt.Printf("failed to expand: %v", errs)
			// we should log the failed items, still waiting for pb logs
			continue // we're skipping failed items
		}

		author := m.ExpandedOne("author")

		if author == nil {
			//every item in the db should have an author
			// log those items which dont for fixing
			continue
		}

		updatedAtTime := m.GetUpdated().Time()

		err := artistSitemap.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/artist/%s/%s", author.GetString("slug"), m.GetId()),
			LastMod:    &updatedAtTime,
			ChangeFreq: smg.Monthly,
			Priority:   0.8,
		})

		if err != nil {
			log.Fatal("Unable to Save Sitemap:", err)
		}
	}
}
