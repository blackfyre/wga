package agentcontent

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

const (
	publicationDirectoryName = "agent-content"
	versionsDirectoryName    = "versions"
	currentFilename          = "current"
	llmsFilename             = "llms.txt"
	manifestFilename         = "manifest.json"
)

var (
	publicationMu  = sync.Mutex{}
	htmlTagPattern = regexp.MustCompile(`<[^>]*>`)
)

// PublicationResult describes one generated agent-content publication.
type PublicationResult struct {
	ArtistCount   int
	ArtworkCount  int
	ExcludedCount int
	Directory     string
	CleanupErr    error
}

type publicationManifest struct {
	Resources map[string]string `json:"resources"`
}

// Resource is one generated public representation and its canonical HTML URL.
type Resource struct {
	Content      []byte
	CanonicalURL string
}

// Directory returns the durable root for versioned agent-content publications.
func Directory(app core.App) string {
	return filepath.Join(app.DataDir(), publicationDirectoryName)
}

// CurrentDirectory resolves the complete publication selected by the atomic
// current-version marker.
func CurrentDirectory(app core.App) (string, error) {
	root := Directory(app)
	data, err := os.ReadFile(filepath.Join(root, currentFilename))
	if err != nil {
		return "", fmt.Errorf("read current agent-content publication: %w", err)
	}
	version := strings.TrimSpace(string(data))
	if version == "" || filepath.Base(version) != version || version == "." || version == ".." {
		return "", fmt.Errorf("invalid current agent-content publication")
	}
	current := filepath.Join(root, versionsDirectoryName, version)
	info, err := os.Stat(current)
	if err != nil {
		return "", fmt.Errorf("stat current agent-content publication: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("current agent-content publication is not a directory")
	}
	return current, nil
}

// ReadCurrent reads a resource from one complete selected publication without
// consulting PocketBase. It retries once if pruning races a marker read.
func ReadCurrent(app core.App, relative string) (Resource, error) {
	if !fs.ValidPath(relative) {
		return Resource{}, fs.ErrNotExist
	}
	for attempt := 0; attempt < 2; attempt++ {
		current, err := CurrentDirectory(app)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && attempt == 0 {
				continue
			}
			return Resource{}, err
		}
		manifestData, err := os.ReadFile(filepath.Join(current, manifestFilename))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && attempt == 0 {
				continue
			}
			return Resource{}, fmt.Errorf("read agent-content manifest: %w", err)
		}
		var manifest publicationManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return Resource{}, fmt.Errorf("parse agent-content manifest: %w", err)
		}
		canonical, ok := manifest.Resources[relative]
		if !ok {
			return Resource{}, fs.ErrNotExist
		}
		content, err := os.ReadFile(filepath.Join(current, filepath.FromSlash(relative)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && attempt == 0 {
				continue
			}
			return Resource{}, fmt.Errorf("read generated agent content: %w", err)
		}
		return Resource{Content: content, CanonicalURL: canonical}, nil
	}
	return Resource{}, fs.ErrNotExist
}

// Publish generates, validates, and atomically selects a complete set of
// session-independent agent resources. Failures before the marker switch leave
// the previous publication selected.
func Publish(app core.App, publicURL config.PublicURL) (PublicationResult, error) {
	publicationMu.Lock()
	defer publicationMu.Unlock()

	root := Directory(app)
	versions := filepath.Join(root, versionsDirectoryName)
	if err := os.MkdirAll(versions, 0o755); err != nil {
		return PublicationResult{}, fmt.Errorf("create agent-content publication directory: %w", err)
	}
	staging, err := os.MkdirTemp(versions, ".staging-")
	if err != nil {
		return PublicationResult{}, fmt.Errorf("create agent-content staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	result, expected, err := generatePublication(app, publicURL, staging)
	if err != nil {
		return PublicationResult{}, err
	}
	if err := validatePublication(staging, expected); err != nil {
		return PublicationResult{}, err
	}

	version := "publication-" + strings.TrimPrefix(filepath.Base(staging), ".staging-")
	published := filepath.Join(versions, version)
	if err := os.Rename(staging, published); err != nil {
		return PublicationResult{}, fmt.Errorf("publish staged agent content: %w", err)
	}
	if err := selectPublication(root, version); err != nil {
		_ = os.RemoveAll(published)
		return PublicationResult{}, err
	}

	result.Directory = published
	result.CleanupErr = prunePublications(versions, version)
	return result, nil
}

type artworkProjection struct {
	record    *core.Record
	author    *core.Record
	canonical string
	link      Link
}

func generatePublication(app core.App, publicURL config.PublicURL, staging string) (PublicationResult, map[string]struct{}, error) {
	baseURL := strings.TrimRight(publicURL.String(), "/")
	expected := map[string]struct{}{llmsFilename: {}, manifestFilename: {}}
	manifest := publicationManifest{Resources: map[string]string{llmsFilename: baseURL + "/llms.txt"}}
	if err := os.WriteFile(filepath.Join(staging, llmsFilename), renderLLMs(baseURL), 0o644); err != nil {
		return PublicationResult{}, nil, fmt.Errorf("write staged llms.txt: %w", err)
	}

	artistRecords, err := app.FindRecordsByFilter(constants.CollectionArtists, "published = true", "+name", 0, 0)
	if err != nil {
		return PublicationResult{}, nil, fmt.Errorf("fetch artists for agent content: %w", err)
	}
	artworkRecords, err := app.FindRecordsByFilter(constants.CollectionArtworks, "published = true", "+title", 0, 0)
	if err != nil {
		return PublicationResult{}, nil, fmt.Errorf("fetch artworks for agent content: %w", err)
	}

	artists := make(map[string]*core.Record, len(artistRecords))
	result := PublicationResult{}
	for _, record := range artistRecords {
		if !validRecord(record, "name") {
			result.ExcludedCount++
			continue
		}
		artists[record.Id] = record
	}

	artworks := make([]artworkProjection, 0, len(artworkRecords))
	linksByArtist := make(map[string][]Link)
	for _, record := range artworkRecords {
		authorIDs := record.GetStringSlice("author")
		if !validRecord(record, "title") || len(authorIDs) == 0 {
			result.ExcludedCount++
			continue
		}
		author, ok := artists[authorIDs[0]]
		if !ok {
			result.ExcludedCount++
			continue
		}
		canonical := baseURL + urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
			ArtistName: author.GetString("name"), ArtworkTitle: record.GetString("title"),
			ArtistId: author.Id, ArtworkId: record.Id,
		})
		link := Link{Label: record.GetString("title"), URL: canonical}
		artworks = append(artworks, artworkProjection{record: record, author: author, canonical: canonical, link: link})
		linksByArtist[author.Id] = append(linksByArtist[author.Id], link)
	}
	for artistID, links := range linksByArtist {
		sort.Slice(links, func(left, right int) bool {
			if links[left].URL != links[right].URL {
				return links[left].URL < links[right].URL
			}
			return links[left].Label < links[right].Label
		})
		linksByArtist[artistID] = links[:min(len(links), MaxRelatedLinks+1)]
	}

	artistDir := filepath.Join(staging, "agents", "artists")
	artworkDir := filepath.Join(staging, "agents", "artworks")
	if err := os.MkdirAll(artistDir, 0o755); err != nil {
		return PublicationResult{}, nil, fmt.Errorf("create staged artist directory: %w", err)
	}
	if err := os.MkdirAll(artworkDir, 0o755); err != nil {
		return PublicationResult{}, nil, fmt.Errorf("create staged artwork directory: %w", err)
	}

	attribution := Link{Label: "Web Gallery of Art", URL: baseURL + "/pages/about"}
	for _, record := range artistRecords {
		if _, ok := artists[record.Id]; !ok {
			continue
		}
		school, err := relationNames(app, constants.CollectionSchools, record.GetStringSlice("school"))
		if err != nil {
			return PublicationResult{}, nil, fmt.Errorf("load schools for artist %s: %w", record.Id, err)
		}
		content, err := RenderArtist(Artist{
			Name: record.GetString("name"), CanonicalURL: baseURL + urlutils.GenerateArtistUrlFromRecord(record),
			Lifespan: artistLifespan(record), Profession: record.GetString("profession"), School: school,
			Biography: plainText(record.GetString("bio")), Attribution: attribution,
			RelatedArtworks: linksByArtist[record.Id][:min(len(linksByArtist[record.Id]), MaxRelatedLinks)],
		})
		if err != nil {
			return PublicationResult{}, nil, fmt.Errorf("render artist %s: %w", record.Id, err)
		}
		relative := filepath.Join("agents", "artists", record.Id+".md")
		if err := os.WriteFile(filepath.Join(staging, relative), content, 0o644); err != nil {
			return PublicationResult{}, nil, fmt.Errorf("write staged artist %s: %w", record.Id, err)
		}
		expected[filepath.ToSlash(relative)] = struct{}{}
		manifest.Resources[filepath.ToSlash(relative)] = baseURL + urlutils.GenerateArtistUrlFromRecord(record)
		result.ArtistCount++
	}

	for _, projection := range artworks {
		location, dimensions := artworkLocationAndDimensions(projection.record.GetString("comment"))
		related := make([]Link, 0, len(linksByArtist[projection.author.Id])-1)
		for _, link := range linksByArtist[projection.author.Id] {
			if link.URL != projection.canonical {
				related = append(related, link)
			}
		}
		content, err := RenderArtwork(Artwork{
			Title: projection.record.GetString("title"), CanonicalURL: projection.canonical,
			Artist: Link{Label: projection.author.GetString("name"), URL: baseURL + urlutils.GenerateArtistUrlFromRecord(projection.author)},
			Date:   artworkDate(projection.record), Technique: projection.record.GetString("technique"),
			Dimensions: dimensions, Location: location, Commentary: projection.record.GetString("source_comment"),
			Attribution: attribution, RelatedArtworks: related,
		})
		if err != nil {
			return PublicationResult{}, nil, fmt.Errorf("render artwork %s: %w", projection.record.Id, err)
		}
		relative := filepath.Join("agents", "artworks", projection.record.Id+".md")
		if err := os.WriteFile(filepath.Join(staging, relative), content, 0o644); err != nil {
			return PublicationResult{}, nil, fmt.Errorf("write staged artwork %s: %w", projection.record.Id, err)
		}
		expected[filepath.ToSlash(relative)] = struct{}{}
		manifest.Resources[filepath.ToSlash(relative)] = projection.canonical
		result.ArtworkCount++
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return PublicationResult{}, nil, fmt.Errorf("marshal agent-content manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')
	if err := os.WriteFile(filepath.Join(staging, manifestFilename), manifestData, 0o644); err != nil {
		return PublicationResult{}, nil, fmt.Errorf("write staged agent-content manifest: %w", err)
	}

	return result, expected, nil
}

func validRecord(record *core.Record, labelField string) bool {
	if record == nil || record.Id == "" || filepath.Base(record.Id) != record.Id {
		return false
	}
	return strings.TrimSpace(record.GetString(labelField)) != ""
}

func relationNames(app core.App, collection string, ids []string) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}
	records, err := app.FindRecordsByIds(collection, ids)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(records))
	for _, record := range records {
		if name := strings.TrimSpace(record.GetString("name")); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", "), nil
}

func artistLifespan(record *core.Record) string {
	birth := record.GetInt("year_of_birth")
	death := record.GetInt("year_of_death")
	if birth <= 0 && death <= 0 {
		return ""
	}
	format := func(year int, exact string) string {
		if year <= 0 {
			return "?"
		}
		value := strconv.Itoa(year)
		if exact == "no" {
			value = "c. " + value
		}
		return value
	}
	return format(birth, record.GetString("exact_year_of_birth")) + "–" + format(death, record.GetString("exact_year_of_death"))
}

func artworkDate(record *core.Record) string {
	if timeframe := strings.TrimSpace(record.GetString("timeframe_text")); timeframe != "" {
		return timeframe
	}
	start := record.GetInt("date_start")
	end := record.GetInt("date_end")
	if start <= 0 && end <= 0 {
		return ""
	}
	value := strconv.Itoa(max(start, end))
	if start > 0 && end > 0 && start != end {
		value = strconv.Itoa(start) + "–" + strconv.Itoa(end)
	}
	if record.GetBool("is_circa") {
		value = "c. " + value
	}
	if qualifier := strings.TrimSpace(record.GetString("date_qualifier")); qualifier != "" {
		value = qualifier + " " + value
	}
	return value
}

func artworkLocationAndDimensions(comment string) (string, string) {
	parts := strings.Split(plainText(comment), " · ")
	if len(parts) < 3 {
		return "", ""
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
}

func plainText(value string) string {
	return html.UnescapeString(htmlTagPattern.ReplaceAllString(value, " "))
}

func renderLLMs(baseURL string) []byte {
	return []byte(fmt.Sprintf(`# Web Gallery of Art

The canonical Web Gallery of Art catalogue is available at <%s/>.

## Discovery

- Sitemap: <%s/sitemap.xml>
- Artist Markdown: %s/agents/artists/{id}.md
- Artwork Markdown: %s/agents/artworks/{id}.md

## Usage

Use the sitemap to discover current canonical records, then request the corresponding generated Markdown resource. These representations contain bounded public catalogue fields and canonical links; this document does not embed the complete catalogue.
`, baseURL, baseURL, baseURL, baseURL))
}

func validatePublication(staging string, expected map[string]struct{}) error {
	seen := make(map[string]struct{}, len(expected))
	err := filepath.WalkDir(staging, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("staged agent content contains symlink")
		}
		relative, err := filepath.Rel(staging, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, ok := expected[relative]; !ok {
			return fmt.Errorf("unexpected staged agent-content file %q", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("staged agent-content file %q is empty", relative)
		}
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return fmt.Errorf("validate staged agent content: %w", err)
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("validate staged agent content: found %d files, expected %d", len(seen), len(expected))
	}
	return nil
}

func selectPublication(root, version string) error {
	marker, err := os.CreateTemp(root, ".current-")
	if err != nil {
		return fmt.Errorf("create agent-content publication marker: %w", err)
	}
	markerPath := marker.Name()
	defer os.Remove(markerPath)
	if _, err := marker.WriteString(version + "\n"); err != nil {
		_ = marker.Close()
		return fmt.Errorf("write agent-content publication marker: %w", err)
	}
	if err := marker.Sync(); err != nil {
		_ = marker.Close()
		return fmt.Errorf("sync agent-content publication marker: %w", err)
	}
	if err := marker.Close(); err != nil {
		return fmt.Errorf("close agent-content publication marker: %w", err)
	}
	if err := os.Rename(markerPath, filepath.Join(root, currentFilename)); err != nil {
		return fmt.Errorf("select agent-content publication: %w", err)
	}
	return nil
}

func prunePublications(versions, current string) error {
	entries, err := os.ReadDir(versions)
	if err != nil {
		return fmt.Errorf("list agent-content publications: %w", err)
	}
	var cleanupErr error
	for _, entry := range entries {
		if entry.Name() == current || !entry.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(versions, entry.Name())); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove stale agent-content publication: %w", err))
		}
	}
	return cleanupErr
}
