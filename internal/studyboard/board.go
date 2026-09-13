// Package studyboard owns the transient Study Board state contract.
package studyboard

import (
	"regexp"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// MaxWorks is the fixed capacity of one Study Board.
	MaxWorks        = 12
	lookupChunkSize = 200
)

var recordIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)

// Artwork is the deliberately small public projection needed before the
// comparison views are assembled. Unpublished records never enter this type.
type Artwork struct {
	ID    string
	Title string
}

// Board is one validated, ordered, duplicate-free transient board.
type Board struct {
	Works []Artwork
}

// IDs returns a copy of the board's canonical ordered identifiers.
func (b Board) IDs() []string {
	ids := make([]string, len(b.Works))
	for i, work := range b.Works {
		ids[i] = work.ID
	}
	return ids
}

// AtCapacity reports whether another distinct work can be appended.
func (b Board) AtCapacity() bool {
	return len(b.Works) >= MaxWorks
}

// Resolve validates an untrusted comma-separated board value against currently
// published artwork records while retaining the request order.
func Resolve(app core.App, raw string) (Board, error) {
	candidates := candidateIDs(raw)
	works := make([]Artwork, 0, min(len(candidates), MaxWorks))

	for start := 0; start < len(candidates) && len(works) < MaxWorks; start += lookupChunkSize {
		end := min(start+lookupChunkSize, len(candidates))
		records, err := app.FindRecordsByIds(constants.CollectionArtworks, candidates[start:end])
		if err != nil {
			return Board{}, err
		}

		published := make(map[string]*core.Record, len(records))
		for _, record := range records {
			if record.GetBool("published") {
				published[record.Id] = record
			}
		}

		for _, id := range candidates[start:end] {
			record, ok := published[id]
			if !ok {
				continue
			}
			works = append(works, Artwork{ID: id, Title: record.GetString("title")})
			if len(works) == MaxWorks {
				break
			}
		}
	}

	return Board{Works: works}, nil
}

func candidateIDs(raw string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		id := strings.TrimSpace(value)
		if !recordIDPattern.MatchString(id) {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
