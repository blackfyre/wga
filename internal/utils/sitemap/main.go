package sitemap

import (
	"fmt"
	"log"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils/url"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/sabloger/sitemap-generator/smg"
)

// setupSitemapIndex initializes and configures a SitemapIndex object.
// It sets the SitemapIndex name, hostname, output path, server URI, and compression settings.
// The SitemapIndex object is then returned.
func setupSitemapIndex(config config.Sitemap) *smg.SitemapIndex {
	isDevelopment := config.Environment.IsDevelopment()
	index := smg.NewSitemapIndex(isDevelopment)
	index.SetSitemapIndexName("web_gallery_of_art")
	index.SetHostname(config.PublicURL.String())
	index.SetOutputPath("./wga_sitemap")
	index.SetServerURI("/sitemaps/")

	index.SetCompress(!isDevelopment)

	return index
}

func GenerateSiteMap(app *pocketbase.PocketBase, config config.Sitemap) {

	index := setupSitemapIndex(config)

	generateArtistMap(app, index)
	generateArtworksMap(app, index)

	// Save func saves the xml files and returns more than one filename in case of split large files.
	filenames, err := index.Save()
	if err != nil {
		app.Logger().Error("Unable to save sitemap", "error", err)
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

func fetchArtistsForSitemap(app *pocketbase.PocketBase) ([]*core.Record, error) {
	return app.FindRecordsByFilter(
		constants.CollectionArtists,
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
		app.Logger().Error("Error fetching artists for sitemap", "error", err)
	}

	for _, m := range records {

		updatedAtTime := m.GetDateTime("updated").Time()

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

	records, err := app.FindRecordsByFilter(
		constants.CollectionArtworks,
		"published = true",
		"+title",
		0,
		0,
	)

	if err != nil {
		log.Fatal("Unable to Save Sitemap:", err)
	}
	authors, err := fetchArtworkAuthors(app, records)
	if err != nil {
		app.Logger().Error("Error loading artwork authors for sitemap", "error", err)
		return
	}

	for _, m := range records {
		authorIDs := m.GetStringSlice("author")
		if len(authorIDs) == 0 {
			app.Logger().Error("Artwork has no author", "id", m.Id)
			continue
		}
		author, ok := authors[authorIDs[0]]
		if !ok {
			app.Logger().Error("Artwork author is missing", "id", m.Id, "author_id", authorIDs[0])
			continue
		}

		updatedAtTime := m.GetDateTime("updated")
		lastMod := updatedAtTime.Time()

		err := sitemap.Add(&smg.SitemapLoc{
			Loc: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistName:   author.GetString("name"),
				ArtistId:     author.GetString("id"),
				ArtworkId:    m.GetString("id"),
				ArtworkTitle: m.GetString("title"),
			}),
			LastMod:    &lastMod,
			ChangeFreq: smg.Monthly,
			Priority:   0.8,
		})

		if err != nil {
			app.Logger().Error("Unable to save sitemap", "error", err)
			return
		}
	}
}

func fetchArtworkAuthors(app *pocketbase.PocketBase, artworks []*core.Record) (map[string]*core.Record, error) {
	uniqueIDs := map[string]struct{}{}
	for _, artwork := range artworks {
		for _, authorID := range artwork.GetStringSlice("author") {
			uniqueIDs[authorID] = struct{}{}
		}
	}

	ids := make([]string, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}

	authorsByID := make(map[string]*core.Record, len(ids))
	for start := 0; start < len(ids); start += 100 {
		end := min(start+100, len(ids))
		authors, err := app.FindRecordsByIds(constants.CollectionArtists, ids[start:end])
		if err != nil {
			return nil, err
		}
		for _, author := range authors {
			authorsByID[author.Id] = author
		}
	}

	return authorsByID, nil
}
