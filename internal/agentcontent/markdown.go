// Package agentcontent renders bounded, session-independent public catalogue
// projections for cooperative agents.
package agentcontent

import (
	"fmt"
	"html"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// MaxRelatedLinks bounds collection fan-out in one generated document.
	MaxRelatedLinks  = 12
	maxHeadingRunes  = 300
	maxMetadataRunes = 500
	maxProseRunes    = 4000
	maxURLRunes      = 2048
)

// Link is a public, canonical link with a human-readable label.
type Link struct {
	Label string
	URL   string
}

// Artist is the complete set of public artist fields eligible for generated
// Markdown. It intentionally has no session, administrative, or persistence
// metadata fields.
type Artist struct {
	Name            string
	CanonicalURL    string
	Lifespan        string
	Profession      string
	School          string
	Biography       string
	Attribution     Link
	RelatedArtworks []Link
}

// Artwork is the complete set of public artwork fields eligible for generated
// Markdown. It intentionally has no session, administrative, or persistence
// metadata fields.
type Artwork struct {
	Title           string
	CanonicalURL    string
	Artist          Link
	Date            string
	Technique       string
	Dimensions      string
	Location        string
	Commentary      string
	Attribution     Link
	RelatedArtworks []Link
}

// RenderArtist renders one deterministic public artist document.
func RenderArtist(artist Artist) ([]byte, error) {
	name := markdownText(artist.Name, maxHeadingRunes)
	if name == "" {
		return nil, fmt.Errorf("artist name is required")
	}
	canonical, err := canonicalURL(artist.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("artist canonical URL: %w", err)
	}
	attribution, err := markdownLink(artist.Attribution, "")
	if err != nil {
		return nil, fmt.Errorf("artist attribution: %w", err)
	}
	related, err := canonicalRelatedLinks(artist.RelatedArtworks, canonical)
	if err != nil {
		return nil, fmt.Errorf("artist related artworks: %w", err)
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# %s\n\n[View the canonical artist record](<%s>)\n", name, canonical)
	writeMetadata(&output, []metadata{
		{label: "Lifespan", value: artist.Lifespan},
		{label: "Profession", value: artist.Profession},
		{label: "School", value: artist.School},
	})
	writeProse(&output, "Biography", artist.Biography)
	writeLinks(&output, "Related artworks", related)
	fmt.Fprintf(&output, "\n## Attribution\n\n%s\n", attribution)
	return []byte(output.String()), nil
}

// RenderArtwork renders one deterministic public artwork document.
func RenderArtwork(artwork Artwork) ([]byte, error) {
	title := markdownText(artwork.Title, maxHeadingRunes)
	if title == "" {
		return nil, fmt.Errorf("artwork title is required")
	}
	canonical, err := canonicalURL(artwork.CanonicalURL)
	if err != nil {
		return nil, fmt.Errorf("artwork canonical URL: %w", err)
	}
	canonicalRecord, _ := url.Parse(canonical)
	artist, err := markdownLink(artwork.Artist, canonicalRecord.Host)
	if err != nil {
		return nil, fmt.Errorf("artwork artist: %w", err)
	}
	attribution, err := markdownLink(artwork.Attribution, "")
	if err != nil {
		return nil, fmt.Errorf("artwork attribution: %w", err)
	}
	related, err := canonicalRelatedLinks(artwork.RelatedArtworks, canonical)
	if err != nil {
		return nil, fmt.Errorf("artwork related artworks: %w", err)
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# %s\n\n[View the canonical artwork record](<%s>)\n", title, canonical)
	output.WriteString("\n## Artwork\n")
	fmt.Fprintf(&output, "\n- Artist: %s", artist)
	for _, item := range []metadata{
		{label: "Date", value: artwork.Date},
		{label: "Technique", value: artwork.Technique},
		{label: "Dimensions", value: artwork.Dimensions},
		{label: "Location", value: artwork.Location},
	} {
		if value := markdownText(item.value, maxMetadataRunes); value != "" {
			fmt.Fprintf(&output, "\n- %s: %s", item.label, value)
		}
	}
	output.WriteString("\n")
	writeProse(&output, "Commentary", artwork.Commentary)
	writeLinks(&output, "Related artworks", related)
	fmt.Fprintf(&output, "\n## Attribution\n\n%s\n", attribution)
	return []byte(output.String()), nil
}

type metadata struct {
	label string
	value string
}

func writeMetadata(output *strings.Builder, items []metadata) {
	wroteHeading := false
	for _, item := range items {
		value := markdownText(item.value, maxMetadataRunes)
		if value == "" {
			continue
		}
		if !wroteHeading {
			output.WriteString("\n## Artist\n")
			wroteHeading = true
		}
		fmt.Fprintf(output, "\n- %s: %s", item.label, value)
	}
	if wroteHeading {
		output.WriteString("\n")
	}
}

func writeProse(output *strings.Builder, heading string, prose string) {
	value := markdownText(prose, maxProseRunes)
	if value != "" {
		fmt.Fprintf(output, "\n## %s\n\n%s\n", heading, value)
	}
}

func writeLinks(output *strings.Builder, heading string, links []string) {
	if len(links) == 0 {
		return
	}
	fmt.Fprintf(output, "\n## %s\n", heading)
	for _, link := range links {
		fmt.Fprintf(output, "\n- %s", link)
	}
	output.WriteString("\n")
}

func canonicalRelatedLinks(links []Link, parentURL string) ([]string, error) {
	parent, _ := url.Parse(parentURL)
	bounded := slices.Clone(links)
	slices.SortFunc(bounded, func(left Link, right Link) int {
		if compared := strings.Compare(left.URL, right.URL); compared != 0 {
			return compared
		}
		return strings.Compare(left.Label, right.Label)
	})
	if len(bounded) > MaxRelatedLinks {
		bounded = bounded[:MaxRelatedLinks]
	}

	rendered := make([]string, 0, len(bounded))
	for index, link := range bounded {
		value, err := markdownLink(link, parent.Host)
		if err != nil {
			return nil, fmt.Errorf("link %d: %w", index, err)
		}
		rendered = append(rendered, value)
	}
	return rendered, nil
}

func markdownLink(link Link, requiredHost string) (string, error) {
	label := markdownText(link.Label, maxHeadingRunes)
	if label == "" {
		return "", fmt.Errorf("link label is required")
	}
	destination, err := canonicalURL(link.URL)
	if err != nil {
		return "", err
	}
	parsed, _ := url.Parse(destination)
	if requiredHost != "" && !strings.EqualFold(parsed.Host, requiredHost) {
		return "", fmt.Errorf("link must use canonical host %q", requiredHost)
	}
	return fmt.Sprintf("[%s](<%s>)", label, destination), nil
}

func canonicalURL(raw string) (string, error) {
	if utf8.RuneCountInString(raw) > maxURLRunes || strings.ContainsAny(raw, "\r\n<>") {
		return "", fmt.Errorf("URL is not a bounded HTTP URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("URL must be an absolute canonical HTTP URL")
	}
	if parsed.Path == "/_" || strings.HasPrefix(parsed.Path, "/_/") || parsed.Path == "/api" || strings.HasPrefix(parsed.Path, "/api/") {
		return "", fmt.Errorf("URL must not target an administrative route")
	}
	return parsed.String(), nil
}

func markdownText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit-1]) + "…"
	}
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"|", "\\|",
	)
	return html.EscapeString(replacer.Replace(value))
}
