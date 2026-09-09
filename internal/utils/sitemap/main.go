package sitemap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blackfyre/wga/internal/agentcontent"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/generatedpublication"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
	"github.com/sabloger/sitemap-generator/smg"
)

const (
	directoryName       = "sitemap"
	legacyDirectoryName = "sitemaps"
	indexFilename       = "sitemap.xml"
	childPath           = "/sitemap/"
	xslPath             = "/sitemap.xsl"
)

var generationMu sync.Mutex

// Result describes one sitemap publication attempt.
type Result struct {
	URLCount           int
	ExcludedCount      int
	AgentArtistCount   int
	AgentArtworkCount  int
	AgentExcludedCount int
	IndexPath          string
	CleanupErr         error
}

func CurrentDirectory(app core.App) (string, error) {
	publication, err := generatedpublication.CurrentDirectory(app)
	if err != nil {
		if errors.Is(err, generatedpublication.ErrNoCurrentPublication) {
			return legacyCurrentDirectory(app)
		}
		return "", err
	}
	return filepath.Join(publication, directoryName), nil
}

func legacyCurrentDirectory(app core.App) (string, error) {
	legacy := filepath.Join(app.DataDir(), legacyDirectoryName)
	info, err := os.Stat(filepath.Join(legacy, indexFilename))
	if err != nil {
		return "", fmt.Errorf("stat legacy sitemap publication: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("legacy sitemap index is not a file")
	}
	return legacy, nil
}

func ReadCurrent(app core.App, relative string) ([]byte, error) {
	return readCurrent(app, relative, CurrentDirectory)
}

func readCurrent(app core.App, relative string, resolveCurrent func(core.App) (string, error)) ([]byte, error) {
	if !fs.ValidPath(relative) {
		return nil, fs.ErrNotExist
	}
	for attempt := 0; attempt < 2; attempt++ {
		current, err := resolveCurrent(app)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && attempt == 0 {
				continue
			}
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(current, filepath.FromSlash(relative)))
		if errors.Is(err, fs.ErrNotExist) && attempt == 0 {
			continue
		}
		return data, err
	}
	return nil, fs.ErrNotExist
}

// GenerateSiteMap creates and publishes a complete sitemap set. A failed run
// leaves the currently published index untouched.
func GenerateSiteMap(app core.App, sitemapConfig config.Sitemap) (result Result, err error) {
	generationMu.Lock()
	defer generationMu.Unlock()
	publicationLock, err := generatedpublication.AcquireLock(app)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		err = errors.Join(err, publicationLock.Close())
	}()

	generation := time.Now().UTC().Format("20060102T150405.000000000")
	version := "publication-" + generation
	staging, err := generatedpublication.NewStaging(app)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(staging)
	stagingDir := filepath.Join(staging, directoryName)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create sitemap staging directory: %w", err)
	}

	index := setupSitemapIndex(sitemapConfig, stagingDir)
	artistMap := setupSitemap("artists-"+generation, index)
	artworkMap := setupSitemap("artworks-"+generation, index)

	artistURLs, artistExcluded, err := generateArtistMap(app, artistMap)
	if err != nil {
		return Result{}, err
	}
	artworkURLs, artworkExcluded, err := generateArtworksMap(app, artworkMap)
	if err != nil {
		return Result{}, err
	}

	filename, err := index.Save()
	if err != nil {
		return Result{}, fmt.Errorf("save sitemap index: %w", err)
	}
	if filename != indexFilename {
		return Result{}, fmt.Errorf("unexpected sitemap index filename %q", filename)
	}
	if err := attachStylesheet(stagingDir); err != nil {
		return Result{}, err
	}

	_, err = validate(stagingDir, sitemapConfig)
	if err != nil {
		return Result{}, err
	}
	agentResult, err := agentcontent.Generate(app, sitemapConfig.PublicURL, filepath.Join(staging, agentcontent.PublicationDirectoryName))
	if err != nil {
		return Result{}, fmt.Errorf("generate agent content: %w", err)
	}
	published, err := generatedpublication.Publish(app, staging, version)
	if err != nil {
		return Result{}, err
	}

	result = Result{
		URLCount:           artistURLs + artworkURLs,
		ExcludedCount:      artistExcluded + artworkExcluded,
		AgentArtistCount:   agentResult.ArtistCount,
		AgentArtworkCount:  agentResult.ArtworkCount,
		AgentExcludedCount: agentResult.ExcludedCount,
		IndexPath:          filepath.Join(published, directoryName, indexFilename),
	}
	result.CleanupErr = errors.Join(generatedpublication.Prune(app, version), agentResult.CleanupErr)

	return result, nil
}

func attachStylesheet(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("list staged sitemap files for stylesheet: %w", err)
	}
	processingInstruction := []byte(`<?xml-stylesheet type="text/xsl" href="` + xslPath + `"?>`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xml") {
			continue
		}
		filename := filepath.Join(directory, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read staged sitemap %q for stylesheet: %w", entry.Name(), err)
		}
		declarationEnd := bytes.Index(data, []byte("?>"))
		if declarationEnd < 0 {
			return fmt.Errorf("staged sitemap %q has no XML declaration", entry.Name())
		}
		styled := append([]byte{}, data[:declarationEnd+2]...)
		styled = append(styled, '\n')
		styled = append(styled, processingInstruction...)
		styled = append(styled, '\n')
		styled = append(styled, data[declarationEnd+2:]...)
		if err := os.WriteFile(filename, styled, 0o644); err != nil {
			return fmt.Errorf("write stylesheet reference to sitemap %q: %w", entry.Name(), err)
		}
	}
	return nil
}

func setupSitemapIndex(sitemapConfig config.Sitemap, outputPath string) *smg.SitemapIndex {
	index := smg.NewSitemapIndex(false)
	index.SetSitemapIndexName(strings.TrimSuffix(indexFilename, ".xml"))
	index.SetHostname(sitemapConfig.PublicURL.String())
	index.SetOutputPath(outputPath)
	index.SetServerURI(childPath)
	index.SetCompress(false)
	return index
}

func setupSitemap(name string, index *smg.SitemapIndex) *smg.Sitemap {
	now := time.Now().UTC()
	sitemap := index.NewSitemap()
	sitemap.SetName(name)
	sitemap.SetLastMod(&now)
	return sitemap
}

func generateArtistMap(app core.App, sitemap *smg.Sitemap) (int, int, error) {
	records, err := app.FindRecordsByFilter(constants.CollectionArtists, "published = true", "+name", 0, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("fetch artists for sitemap: %w", err)
	}

	count := 0
	excluded := 0
	for _, record := range records {
		if record.Id == "" || strings.TrimSpace(record.GetString("name")) == "" {
			excluded++
			continue
		}

		updated := record.GetDateTime("updated").Time()
		if err := sitemap.Add(&smg.SitemapLoc{
			Loc:        urlutils.GenerateArtistUrlFromRecord(record),
			LastMod:    &updated,
			ChangeFreq: smg.Monthly,
			Priority:   0.8,
		}); err != nil {
			return 0, 0, fmt.Errorf("add artist %s to sitemap: %w", record.Id, err)
		}
		count++
	}

	return count, excluded, nil
}

func generateArtworksMap(app core.App, sitemap *smg.Sitemap) (int, int, error) {
	records, err := app.FindRecordsByFilter(constants.CollectionArtworks, "published = true", "+title", 0, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("fetch artworks for sitemap: %w", err)
	}
	authors, err := fetchArtworkAuthors(app, records)
	if err != nil {
		return 0, 0, fmt.Errorf("load artwork authors for sitemap: %w", err)
	}

	count := 0
	excluded := 0
	for _, record := range records {
		authorIDs := record.GetStringSlice("author")
		if record.Id == "" || strings.TrimSpace(record.GetString("title")) == "" || len(authorIDs) == 0 {
			excluded++
			continue
		}
		author, ok := authors[authorIDs[0]]
		if !ok || !author.GetBool("published") || strings.TrimSpace(author.GetString("name")) == "" {
			excluded++
			continue
		}

		updated := record.GetDateTime("updated").Time()
		if err := sitemap.Add(&smg.SitemapLoc{
			Loc: urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
				ArtistName:   author.GetString("name"),
				ArtistId:     author.Id,
				ArtworkId:    record.Id,
				ArtworkTitle: record.GetString("title"),
			}),
			LastMod:    &updated,
			ChangeFreq: smg.Monthly,
			Priority:   0.8,
		}); err != nil {
			return 0, 0, fmt.Errorf("add artwork %s to sitemap: %w", record.Id, err)
		}
		count++
	}

	return count, excluded, nil
}

func fetchArtworkAuthors(app core.App, artworks []*core.Record) (map[string]*core.Record, error) {
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

type sitemapIndex struct {
	XMLName xml.Name            `xml:"sitemapindex"`
	Maps    []sitemapIndexEntry `xml:"sitemap"`
}

type sitemapIndexEntry struct {
	Loc string `xml:"loc"`
}

type sitemapURLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

func validate(stagingDir string, sitemapConfig config.Sitemap) (map[string]struct{}, error) {
	data, err := os.ReadFile(filepath.Join(stagingDir, indexFilename))
	if err != nil {
		return nil, fmt.Errorf("read staged sitemap index: %w", err)
	}

	var index sitemapIndex
	if err := xml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse staged sitemap index: %w", err)
	}
	if index.XMLName.Local != "sitemapindex" || len(index.Maps) == 0 {
		return nil, fmt.Errorf("staged sitemap index has no child maps")
	}

	publicURL, err := url.Parse(sitemapConfig.PublicURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse public URL: %w", err)
	}
	children := make(map[string]struct{}, len(index.Maps))
	for _, entry := range index.Maps {
		location, err := url.Parse(entry.Loc)
		if err != nil || location.Scheme != publicURL.Scheme || location.Host != publicURL.Host || !strings.HasPrefix(location.Path, childPath) {
			return nil, fmt.Errorf("invalid child sitemap URL %q", entry.Loc)
		}
		filename := path.Base(location.Path)
		if filename == "." || filename == indexFilename || !strings.HasSuffix(filename, ".xml") {
			return nil, fmt.Errorf("invalid child sitemap filename %q", filename)
		}
		if _, exists := children[filename]; exists {
			return nil, fmt.Errorf("duplicate child sitemap filename %q", filename)
		}
		childData, err := os.ReadFile(filepath.Join(stagingDir, filename))
		if err != nil {
			return nil, fmt.Errorf("read staged child sitemap %q: %w", filename, err)
		}
		var child sitemapURLSet
		if err := xml.Unmarshal(childData, &child); err != nil || child.XMLName.Local != "urlset" {
			return nil, fmt.Errorf("parse staged child sitemap %q: %w", filename, err)
		}
		children[filename] = struct{}{}
	}

	entries, err := os.ReadDir(stagingDir)
	if err != nil {
		return nil, fmt.Errorf("list staged sitemap files: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == indexFilename {
			continue
		}
		if _, ok := children[entry.Name()]; !ok {
			return nil, fmt.Errorf("staged sitemap file %q is not indexed", entry.Name())
		}
	}

	return children, nil
}
