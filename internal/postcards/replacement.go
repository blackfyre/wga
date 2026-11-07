package postcards

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// ReplacementQueryMinRunes and ReplacementQueryMaxRunes bound the title
	// search offered when a draft's selected work becomes unavailable.
	ReplacementQueryMinRunes = 2
	ReplacementQueryMaxRunes = 100
	replacementLimit         = 6
	replacementScanLimit     = 24
)

// ErrReplacementQueryInvalid indicates a replacement search shorter or longer
// than the accepted bounds.
var ErrReplacementQueryInvalid = errors.New("postcard replacement query is outside its bounds")

// ReplacementArtwork is a published work a draft can switch to without losing
// its entered values.
type ReplacementArtwork struct {
	ID               string
	Title            string
	ArtistFilingName string
}

// FindReplacementArtworks returns up to six published works whose title
// contains the query and which have a published, complete author identity,
// so every result can open the composer.
func FindReplacementArtworks(app core.App, query string) ([]ReplacementArtwork, error) {
	query = strings.TrimSpace(query)
	if count := utf8.RuneCountInString(query); count < ReplacementQueryMinRunes || count > ReplacementQueryMaxRunes {
		return nil, ErrReplacementQueryInvalid
	}
	records, err := app.FindRecordsByFilter(constants.CollectionArtworks, "published = true && title ~ {:query}", "title", replacementScanLimit, 0, dbx.Params{"query": query})
	if err != nil {
		return nil, fmt.Errorf("find postcard replacement artworks: %w", err)
	}
	results := make([]ReplacementArtwork, 0, replacementLimit)
	for _, record := range records {
		author, err := PublishedAuthor(app, record)
		if errors.Is(err, ErrArtistIdentityUnavailable) {
			continue
		}
		if err != nil {
			return nil, err
		}
		results = append(results, ReplacementArtwork{ID: record.Id, Title: record.GetString("title"), ArtistFilingName: author.GetString("filing_name")})
		if len(results) == replacementLimit {
			break
		}
	}
	return results, nil
}
