package agentcontent

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/generatedpublication"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const (
	publicationDirectoryName = "agent-content"
	llmsFilename             = "llms.txt"
	manifestFilename         = "manifest.json"
)

var (
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
	Resources     map[string]string   `json:"resources"`
	AcceptedPaths map[string][]string `json:"accepted_paths,omitempty"`
}

// Resource is one generated public representation and its canonical HTML URL.
type Resource struct {
	Content       []byte
	CanonicalURL  string
	AcceptedPaths []string
}

// CurrentDirectory resolves the complete publication selected by the atomic
// current-version marker.
func CurrentDirectory(app core.App) (string, error) {
	publication, err := generatedpublication.CurrentDirectory(app)
	if err != nil {
		return "", err
	}
	current := filepath.Join(publication, publicationDirectoryName)
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
		content, err := os.ReadFile(filepath.Join(current, filepath.FromSlash(relative)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && attempt == 0 {
				continue
			}
			if errors.Is(err, fs.ErrNotExist) {
				return Resource{}, fs.ErrNotExist
			}
			return Resource{}, fmt.Errorf("read generated agent content: %w", err)
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
		acceptedPaths := manifest.AcceptedPaths[relative]
		if len(acceptedPaths) == 0 {
			if parsed, err := url.Parse(canonical); err == nil && parsed.Path != "" {
				acceptedPaths = []string{parsed.Path}
			}
		}
		return Resource{Content: content, CanonicalURL: canonical, AcceptedPaths: acceptedPaths}, nil
	}
	return Resource{}, fs.ErrNotExist
}

// Generate writes and validates agent resources inside a shared unpublished
// staging directory. The sitemap workflow selects the complete publication.
func Generate(app core.App, publicURL config.PublicURL, staging string) (PublicationResult, error) {
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return PublicationResult{}, fmt.Errorf("create agent-content staging directory: %w", err)
	}
	result, expected, err := generatePublication(app, publicURL, staging)
	if err != nil {
		return PublicationResult{}, err
	}
	if err := validatePublication(staging, expected); err != nil {
		return PublicationResult{}, err
	}

	result.Directory = staging
	return result, nil
}

type artworkProjection struct {
	record        *core.Record
	author        *core.Record
	canonical     string
	acceptedPaths []string
	link          Link
}

func generatePublication(app core.App, publicURL config.PublicURL, staging string) (PublicationResult, map[string]struct{}, error) {
	baseURL := strings.TrimRight(publicURL.String(), "/")
	expected := map[string]struct{}{llmsFilename: {}, manifestFilename: {}}
	manifest := publicationManifest{
		Resources:     map[string]string{llmsFilename: baseURL + "/llms.txt"},
		AcceptedPaths: map[string][]string{llmsFilename: {"/llms.txt"}},
	}
	if err := os.WriteFile(filepath.Join(staging, llmsFilename), renderLLMs(baseURL), 0o644); err != nil {
		return PublicationResult{}, nil, fmt.Errorf("write staged llms.txt: %w", err)
	}

	artistRecords, err := repositories.NewArtistRecordRepository(app).ListPublishedArtists()
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
		var author *core.Record
		eligibleAuthors := make([]*core.Record, 0, len(authorIDs))
		acceptedPaths := make([]string, 0, len(authorIDs))
		for _, authorID := range authorIDs {
			if eligible := artists[authorID]; eligible != nil {
				if author == nil {
					author = eligible
				}
				eligibleAuthors = append(eligibleAuthors, eligible)
				acceptedPaths = append(acceptedPaths, canonicalArtworkPath(eligible, record))
			}
		}
		if author == nil {
			result.ExcludedCount++
			continue
		}
		canonical := baseURL + canonicalArtworkPath(author, record)
		link := Link{Label: record.GetString("title"), URL: canonical}
		artworks = append(artworks, artworkProjection{record: record, author: author, canonical: canonical, acceptedPaths: acceptedPaths, link: link})
		for _, eligible := range eligibleAuthors {
			linksByArtist[eligible.Id] = append(linksByArtist[eligible.Id], Link{
				Label: record.GetString("title"),
				URL:   baseURL + canonicalArtworkPath(eligible, record),
			})
		}
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
			Name: record.GetString("name"), CanonicalURL: baseURL + canonicalArtistPath(record),
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
		manifest.Resources[filepath.ToSlash(relative)] = baseURL + canonicalArtistPath(record)
		manifest.AcceptedPaths[filepath.ToSlash(relative)] = []string{canonicalArtistPath(record)}
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
			Artist: Link{Label: projection.author.GetString("name"), URL: baseURL + canonicalArtistPath(projection.author)},
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
		manifest.AcceptedPaths[filepath.ToSlash(relative)] = projection.acceptedPaths
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

func canonicalArtistPath(record *core.Record) string {
	return "/artists/" + utils.GenerateArtistSlug(record)
}

func canonicalArtworkPath(artist *core.Record, artwork *core.Record) string {
	return canonicalArtistPath(artist) + "/" + utils.Slugify(artwork.GetString("title")) + "-" + artwork.Id
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
	format := func(year int, exact bool) string {
		if year <= 0 {
			return "?"
		}
		value := strconv.Itoa(year)
		if !exact {
			value = "c. " + value
		}
		return value
	}
	return format(birth, record.GetBool("exact_year_of_birth")) + "–" + format(death, record.GetBool("exact_year_of_death"))
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
