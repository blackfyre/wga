package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// RelatedWorkBasis is one of the four visitor-selectable related-work bases.
// The value doubles as the URL query value, so it must stay short and stable.
type RelatedWorkBasis string

const (
	// RelatedByArtist surfaces other published works by the same artist(s).
	RelatedByArtist RelatedWorkBasis = "artist"
	// RelatedByCollection surfaces other published works sharing the artwork's
	// canonical current-location (museum) relation.
	RelatedByCollection RelatedWorkBasis = "collection"
	// RelatedByPalette surfaces published works ranked by image-derived colour
	// signature distance. Implemented in a later serial task.
	RelatedByPalette RelatedWorkBasis = "palette"
	// RelatedByPeriod surfaces other artists' published works catalogued within
	// forty years of the artwork. Implemented in a later serial task.
	RelatedByPeriod RelatedWorkBasis = "period"
)

// DefaultRelatedWorkBasis is the basis applied when a request omits or supplies
// an unknown basis. BY ARTIST is the ordinary record default.
const DefaultRelatedWorkBasis = RelatedByArtist

// relatedWorksLimit caps the related works returned for any basis.
const relatedWorksLimit = 4

// relatedPaletteCandidateLimit bounds the number of profiled candidates the
// palette resolver loads before ranking in memory. It comfortably covers the
// documented current producer profiled-record count (52,865) while guaranteeing
// the palette query can never materialise an unbounded candidate table.
const relatedPaletteCandidateLimit = 60000

// relatedPeriodWindow is the inclusive year window around the artwork's known
// creation date used by SAME PERIOD.
const relatedPeriodWindow = 40

// colourSignatureSpace is the only signature space the resolver treats as a
// comparable producer colour profile. Signatures in any other space are ignored
// rather than compared across incompatible histograms.
const colourSignatureSpace = "oklab-hcl-12x3x4"

// colourSignature is the persisted producer colour signature: a fixed-space,
// weighted histogram over the oklab-hcl bins.
type colourSignature struct {
	Space string `json:"space"`
	Bins  []int  `json:"bins"`
}

// ParseRelatedWorkBasis normalises an arbitrary query value to a basis. Unknown
// values fall back to the default so a malformed URL never breaks the record.
func ParseRelatedWorkBasis(raw string) RelatedWorkBasis {
	switch RelatedWorkBasis(raw) {
	case RelatedByArtist, RelatedByCollection, RelatedByPalette, RelatedByPeriod:
		return RelatedWorkBasis(raw)
	default:
		return DefaultRelatedWorkBasis
	}
}

// IsDefault reports whether the basis is the URL-omitted default.
func (b RelatedWorkBasis) IsDefault() bool {
	return b == DefaultRelatedWorkBasis
}

// RelatedWorkResult is the resolved related-work projection for one artwork and
// basis. Works holds at most relatedWorksLimit published, non-self records in a
// deterministic order.
type RelatedWorkResult struct {
	Basis RelatedWorkBasis
	Works []*core.Record
}

// RelatedWorkResolver is the bounded read-model for the four related-work bases.
// It only ever returns published records and always excludes the current
// artwork, so the projection is safe to render without further filtering.
type RelatedWorkResolver struct {
	app core.App
}

func NewRelatedWorkResolver(app core.App) *RelatedWorkResolver {
	return &RelatedWorkResolver{app: app}
}

// Resolve returns the related works for the active basis. The basis is
// re-normalised defensively so an invalid value resolves to the default rather
// than erroring.
func (r *RelatedWorkResolver) Resolve(artwork *core.Record, basis RelatedWorkBasis) (RelatedWorkResult, error) {
	basis = ParseRelatedWorkBasis(string(basis))

	var (
		works []*core.Record
		err   error
	)
	switch basis {
	case RelatedByCollection:
		works, err = r.relatedByCollection(artwork)
	case RelatedByPalette:
		works, err = r.relatedByPalette(artwork)
	case RelatedByPeriod:
		works, err = r.relatedByPeriod(artwork)
	default: // RelatedByArtist
		works, err = r.relatedByArtist(artwork)
	}
	if err != nil {
		return RelatedWorkResult{}, err
	}

	return RelatedWorkResult{Basis: basis, Works: works}, nil
}

// relatedByArtist returns published works whose author relation includes any of
// the current artwork's authors, excluding the current artwork, ordered by title
// then id for determinism.
func (r *RelatedWorkResolver) relatedByArtist(artwork *core.Record) ([]*core.Record, error) {
	authorIDs := artwork.GetStringSlice("author")
	if len(authorIDs) == 0 {
		return nil, nil
	}

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp("Artworks.id != {:current_id}", dbx.Params{"current_id": artwork.Id}))
	query.AndWhere(authorMembershipExp(authorIDs))
	query.AndWhere(publishedAuthorFilter())
	query.OrderBy("title ASC", "id ASC")
	query.Limit(relatedWorksLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// relatedByCollection returns published works sharing the artwork's canonical
// current-location relation. Only canonical public museums produce a path:
// private collections and public non-museum locations yield no results.
func (r *RelatedWorkResolver) relatedByCollection(artwork *core.Record) ([]*core.Record, error) {
	locationIDs := artwork.GetStringSlice("current_location_id")
	if len(locationIDs) == 0 {
		return nil, nil
	}
	locationID := locationIDs[0]

	location, err := r.app.FindRecordById(constants.CollectionLocations, locationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !location.GetBool("is_public") || !location.GetBool("museum") {
		return nil, nil
	}

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp("Artworks.id != {:current_id}", dbx.Params{"current_id": artwork.Id}))
	// current_location_id is a MaxSelect-1 relation, so PocketBase stores it as a
	// plain TEXT column rather than a JSON array; it is matched directly.
	query.AndWhere(dbx.NewExp(
		"Artworks.current_location_id = {:location_id}",
		dbx.Params{"location_id": locationID},
	))
	query.AndWhere(publishedAuthorFilter())
	query.OrderBy("title ASC", "id ASC")
	query.Limit(relatedWorksLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// relatedByPalette returns published works ranked by the squared Euclidean
// distance between their persisted colour signatures and the current artwork's,
// excluding the current artwork and every candidate that shares an author.
// A missing or invalid current signature yields no result; invalid candidate
// signatures are skipped rather than treated as a valid distance.
func (r *RelatedWorkResolver) relatedByPalette(artwork *core.Record) ([]*core.Record, error) {
	current, ok := parseColourSignature(artwork)
	if !ok {
		return nil, nil
	}
	authorIDs := artwork.GetStringSlice("author")

	records, err := r.paletteCandidates(artwork, authorIDs)
	if err != nil {
		return nil, err
	}

	type candidate struct {
		record   *core.Record
		distance int64
	}
	candidates := make([]candidate, 0, len(records))
	for _, record := range records {
		signature, ok := parseColourSignature(record)
		if !ok || len(signature.Bins) != len(current.Bins) {
			continue
		}
		candidates = append(candidates, candidate{
			record:   record,
			distance: signatureDistance(current, signature),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}
		if ti, tj := candidates[i].record.GetString("title"), candidates[j].record.GetString("title"); ti != tj {
			return ti < tj
		}
		return candidates[i].record.Id < candidates[j].record.Id
	})

	if len(candidates) > relatedWorksLimit {
		candidates = candidates[:relatedWorksLimit]
	}

	works := make([]*core.Record, len(candidates))
	for i, candidate := range candidates {
		works[i] = candidate.record
	}

	return works, nil
}

// paletteCandidates returns the bounded, deterministically ordered candidate
// set for palette ranking. The id-ordered LIMIT bounds the scan and makes any
// truncation deterministic; distance ranking happens in memory afterwards.
func (r *RelatedWorkResolver) paletteCandidates(artwork *core.Record, authorIDs []string) ([]*core.Record, error) {
	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp("Artworks.id != {:current_id}", dbx.Params{"current_id": artwork.Id}))
	query.AndWhere(dbx.NewExp("Artworks.colour_signature IS NOT NULL"))
	query.AndWhere(dbx.NewExp("Artworks.colour_signature != ''"))
	if len(authorIDs) > 0 {
		query.AndWhere(noAuthorOverlapExp(authorIDs))
	}
	query.AndWhere(publishedAuthorFilter())
	query.OrderBy("id ASC")
	query.Limit(relatedPaletteCandidateLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// relatedByPeriod returns published works by other artists whose known creation
// date falls within relatedPeriodWindow years of the current artwork, ordered by
// nearest date difference then title then id. No period is inferred from the
// date; only the stored date_start is used.
func (r *RelatedWorkResolver) relatedByPeriod(artwork *core.Record) ([]*core.Record, error) {
	currentDate := artwork.GetInt("date_start")
	if currentDate <= 0 {
		return nil, nil
	}
	authorIDs := artwork.GetStringSlice("author")

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp("Artworks.id != {:current_id}", dbx.Params{"current_id": artwork.Id}))
	query.AndWhere(dbx.NewExp("date_start > 0"))
	query.AndWhere(dbx.NewExp(
		"ABS(date_start - {:current_date}) <= {:window}",
		dbx.Params{"current_date": currentDate, "window": relatedPeriodWindow},
	))
	if len(authorIDs) > 0 {
		query.AndWhere(noAuthorOverlapExp(authorIDs))
	}
	query.AndWhere(publishedAuthorFilter())
	query.OrderBy(
		"ABS(date_start - "+strconv.Itoa(currentDate)+") ASC",
		"title ASC",
		"id ASC",
	)
	query.Limit(relatedWorksLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// authorMembershipExp builds an EXISTS condition that matches artworks whose
// author relation includes any of the supplied ids. The ids are bound through
// individually named placeholders because the relation is a JSON array.
func authorMembershipExp(authorIDs []string) dbx.Expression {
	params := dbx.Params{}
	placeholders := make([]string, len(authorIDs))
	for i, id := range authorIDs {
		key := "related_author_" + strconv.Itoa(i)
		placeholders[i] = "{:" + key + "}"
		params[key] = id
	}

	return dbx.NewExp(
		"EXISTS (SELECT 1 FROM json_each(Artworks.author) je WHERE je.value IN ("+strings.Join(placeholders, ", ")+"))",
		params,
	)
}

// noAuthorOverlapExp builds a NOT EXISTS condition that matches artworks whose
// author relation shares none of the supplied ids. It is the negation of
// authorMembershipExp, used by the palette and period bases to exclude same-author
// candidates.
func noAuthorOverlapExp(authorIDs []string) dbx.Expression {
	params := dbx.Params{}
	placeholders := make([]string, len(authorIDs))
	for i, id := range authorIDs {
		key := "excluded_author_" + strconv.Itoa(i)
		placeholders[i] = "{:" + key + "}"
		params[key] = id
	}

	return dbx.NewExp(
		"NOT EXISTS (SELECT 1 FROM json_each(Artworks.author) je WHERE je.value IN ("+strings.Join(placeholders, ", ")+"))",
		params,
	)
}

// publishedAuthorFilter matches artworks whose author relation includes at
// least one published artist. A candidate whose only author is unpublished is
// never surfaced, even when the artwork record itself is published. A candidate
// with one published and one unpublished author still passes.
func publishedAuthorFilter() dbx.Expression {
	return dbx.NewExp(
		"EXISTS (SELECT 1 FROM json_each(Artworks.author) je JOIN Artists a ON a.id = je.value WHERE a.published = true)",
	)
}

// parseColourSignature reads the stored JSON signature and reports whether it is
// a usable producer signature in the expected space. Stored JSON is treated as
// untrusted: malformed values, missing bins, and other spaces are rejected
// without error.
func parseColourSignature(record *core.Record) (colourSignature, bool) {
	data, err := json.Marshal(record.Get("colour_signature"))
	if err != nil {
		return colourSignature{}, false
	}

	var signature colourSignature
	if err := json.Unmarshal(data, &signature); err != nil {
		return colourSignature{}, false
	}
	if signature.Space != colourSignatureSpace || len(signature.Bins) == 0 {
		return colourSignature{}, false
	}

	return signature, true
}

// signatureDistance returns the squared Euclidean distance between two same-space
// signatures. Both signatures must have equal bin lengths; the caller enforces
// this before comparing.
func signatureDistance(a, b colourSignature) int64 {
	var distance int64
	for i := range a.Bins {
		diff := int64(a.Bins[i]) - int64(b.Bins[i])
		distance += diff * diff
	}

	return distance
}
