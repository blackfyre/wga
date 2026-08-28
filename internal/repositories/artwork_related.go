package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

// relatedWorksLimit caps the related works presented for any basis: a four-work
// closest-date sample selected from the resolved candidates.
const relatedWorksLimit = 4

// relatedCandidatesLimit bounds the number of candidates any basis resolves
// before the closest-date sample is selected. Palette candidates are ranked by
// colour-signature distance before this limit is applied, so the eight-candidate
// set is the eight nearest profiles rather than an id-ordered prefix.
const relatedCandidatesLimit = 8

// relatedPeriodWindow is the inclusive year window around the artwork's known
// creation date used by SAME PERIOD.
const relatedPeriodWindow = 40

// colourSignatureSpace is the only signature space the resolver treats as a
// comparable producer colour profile. Signatures in any other space are ignored
// rather than compared across incompatible histograms.
const colourSignatureSpace = "oklab-hcl-12x3x4"

// colourSignatureBinCount is the fixed number of histogram bins in the producer
// colour signature: 12 hue * 3 chroma * 4 lightness chromatic bins plus 4
// neutral lightness bins. The palette distance is expressed over exactly this
// many parameter-bound bins in SQL rather than loading candidate records.
const colourSignatureBinCount = 148

// Related-work holding link keys. artist and venue match the artwork-search
// query parameter names (venue is the canonical collection facet); period
// matches the art_period_id facet.
const (
	relatedHoldingArtist = "artist"
	relatedHoldingVenue  = "venue"
	relatedHoldingPeriod = "period"
)

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

// RelatedWorkHolding describes the complete artwork-search holding behind a
// filterable related-work basis. QueryKey and QueryValue are the artwork-search
// query parameter name and value that reproduce the holding; Count is the number
// of published works the search returns for that filter, including the current
// artwork. Palette similarity is ranking-only and exposes no holding.
type RelatedWorkHolding struct {
	QueryKey   string
	QueryValue string
	Count      int
}

// RelatedWorkResult is the resolved related-work projection for one artwork and
// basis. Works holds at most relatedWorksLimit published, non-self records in a
// deterministic closest-date order. Holding is non-nil only for the filterable
// artist, collection, and period bases with a usable filter value.
type RelatedWorkResult struct {
	Basis   RelatedWorkBasis
	Works   []*core.Record
	Holding *RelatedWorkHolding
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

	candidates, err := r.candidates(artwork, basis)
	if err != nil {
		return RelatedWorkResult{}, err
	}

	works := selectClosestDateWorks(candidates, artwork.GetInt("date_start"), relatedWorksLimit)

	holding, err := r.holding(artwork, basis)
	if err != nil {
		return RelatedWorkResult{}, err
	}

	return RelatedWorkResult{Basis: basis, Works: works, Holding: holding}, nil
}

// candidates resolves the bounded, deterministically ordered candidate set for
// the active basis. Each basis returns at most relatedCandidatesLimit published,
// non-self, public-author-valid records; the common closest-date selector then
// reduces them to the presented sample.
func (r *RelatedWorkResolver) candidates(artwork *core.Record, basis RelatedWorkBasis) ([]*core.Record, error) {
	switch basis {
	case RelatedByCollection:
		return r.relatedByCollection(artwork)
	case RelatedByPalette:
		return r.relatedByPalette(artwork)
	case RelatedByPeriod:
		return r.relatedByPeriod(artwork)
	default: // RelatedByArtist
		return r.relatedByArtist(artwork)
	}
}

// relatedByArtist returns published works whose author relation includes any of
// the current artwork's authors, excluding the current artwork, ordered by title
// then id for determinism. It returns at most relatedCandidatesLimit candidates;
// the common closest-date selector narrows them to the presented sample.
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
	query.Limit(relatedCandidatesLimit)

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
	location, err := r.canonicalPublicMuseum(artwork)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, nil
	}

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp("Artworks.id != {:current_id}", dbx.Params{"current_id": artwork.Id}))
	// current_location_id is a MaxSelect-1 relation, so PocketBase stores it as a
	// plain TEXT column rather than a JSON array; it is matched directly.
	query.AndWhere(dbx.NewExp(
		"Artworks.current_location_id = {:location_id}",
		dbx.Params{"location_id": location.Id},
	))
	query.AndWhere(publishedAuthorFilter())
	query.OrderBy("title ASC", "id ASC")
	query.Limit(relatedCandidatesLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// canonicalPublicMuseum returns the artwork's current-location record when it is
// a canonical public museum, or nil when the location is absent, private, or a
// public non-museum. A missing location record yields nil rather than an error.
func (r *RelatedWorkResolver) canonicalPublicMuseum(artwork *core.Record) (*core.Record, error) {
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
	return location, nil
}

// relatedByPalette returns published works ranked by the squared Euclidean
// distance between their persisted colour signatures and the current artwork's,
// excluding the current artwork and every candidate that shares an author.
// A missing or invalid current signature yields no result. The distance is
// computed in SQL over the producer's fixed bin count and the nearest candidates
// are returned before LIMIT, so the query never loads the full profiled
// catalogue into memory.
func (r *RelatedWorkResolver) relatedByPalette(artwork *core.Record) ([]*core.Record, error) {
	current, ok := parseColourSignature(artwork)
	if !ok || len(current.Bins) != colourSignatureBinCount {
		return nil, nil
	}
	authorIDs := artwork.GetStringSlice("author")

	query := r.app.RecordQuery(constants.CollectionArtworks)
	query.AndWhere(dbx.NewExp("published = true"))
	query.AndWhere(dbx.NewExp("Artworks.id != {:current_id}", dbx.Params{"current_id": artwork.Id}))
	query.AndWhere(validColourSignatureFilter())
	if len(authorIDs) > 0 {
		query.AndWhere(noAuthorOverlapExp(authorIDs))
	}
	query.AndWhere(publishedAuthorFilter())
	query.AndBind(colourSignatureBinsParams(current.Bins))
	query.OrderBy(colourDistanceOrderExpr(), "title ASC", "id ASC")
	query.Limit(relatedCandidatesLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// validColourSignatureFilter restricts candidates to producer colour signatures
// in the expected space with the fixed bin count. Invalid or partial signatures
// are excluded in SQL so the distance expression never evaluates a missing bin
// as a NULL distance that would rank ahead of valid candidates.
func validColourSignatureFilter() dbx.Expression {
	return dbx.NewExp(
		"json_extract(Artworks.colour_signature, '$.space') = {:space} AND json_array_length(json_extract(Artworks.colour_signature, '$.bins')) = {:bins}",
		dbx.Params{"space": colourSignatureSpace, "bins": colourSignatureBinCount},
	)
}

// colourSignatureBinsParams binds the current signature's bin values under
// stable placeholder names so the distance expression can reference them.
func colourSignatureBinsParams(bins []int) dbx.Params {
	params := make(dbx.Params, len(bins))
	for i, bin := range bins {
		params["bin_"+strconv.Itoa(i)] = bin
	}
	return params
}

// colourDistanceOrderExpr builds the ORDER BY expression that ranks candidates
// by squared Euclidean distance to the current signature. Each bin is extracted
// via json_extract and compared against a bound parameter; the expression is
// parenthesised so dbx passes it through unquoted.
func colourDistanceOrderExpr() string {
	terms := make([]string, 0, colourSignatureBinCount)
	for i := 0; i < colourSignatureBinCount; i++ {
		idx := strconv.Itoa(i)
		terms = append(terms, fmt.Sprintf(
			"(json_extract(Artworks.colour_signature, '$.bins[%s]') - {:bin_%s}) * (json_extract(Artworks.colour_signature, '$.bins[%s]') - {:bin_%s})",
			idx, idx, idx, idx,
		))
	}
	return "(" + strings.Join(terms, " + ") + ") ASC"
}

// relatedByPeriod returns published works by other artists whose known creation
// date falls within relatedPeriodWindow years of the current artwork, ordered by
// nearest date difference then title then id. No period is inferred from the
// date; only the stored date_start is used. It returns at most
// relatedCandidatesLimit candidates; the common closest-date selector narrows
// them to the presented sample.
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
	query.Limit(relatedCandidatesLimit)

	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return nil, err
	}

	return records, nil
}

// selectClosestDateWorks reduces the candidate set to a deterministic at-most
// limit closest-date sample. Known dates (date_start > 0) always precede unknown
// dates. When the current date is known, known candidates are ordered by their
// distance to it, with earlier dates breaking equal distances. When the current
// date is unknown, known candidates are ordered by their own date instead. Title
// then id provide the final deterministic tie-breaks.
func selectClosestDateWorks(candidates []*core.Record, currentDate int, limit int) []*core.Record {
	if len(candidates) == 0 || limit <= 0 {
		return nil
	}

	sorted := make([]*core.Record, len(candidates))
	copy(sorted, candidates)

	sort.SliceStable(sorted, func(i, j int) bool {
		di := sorted[i].GetInt("date_start")
		dj := sorted[j].GetInt("date_start")
		iKnown := di > 0
		jKnown := dj > 0

		if iKnown != jKnown {
			return iKnown
		}

		if iKnown && currentDate > 0 {
			if adi, adj := absInt(di-currentDate), absInt(dj-currentDate); adi != adj {
				return adi < adj
			}
		}

		if di != dj {
			return di < dj
		}

		if ti, tj := sorted[i].GetString("title"), sorted[j].GetString("title"); ti != tj {
			return ti < tj
		}

		return sorted[i].Id < sorted[j].Id
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	return sorted
}

// absInt returns the absolute value of v.
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
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

// holding returns the counted artwork-search holding for the active basis, or
// nil for palette (ranking-only) and for bases whose current artwork lacks a
// usable filter value.
func (r *RelatedWorkResolver) holding(artwork *core.Record, basis RelatedWorkBasis) (*RelatedWorkHolding, error) {
	switch basis {
	case RelatedByCollection:
		return r.collectionHolding(artwork)
	case RelatedByPeriod:
		return r.periodHolding(artwork)
	case RelatedByArtist:
		return r.artistHolding(artwork)
	default: // RelatedByPalette
		return nil, nil
	}
}

// artistHolding counts published works by the current artwork's primary author.
// The count matches the artwork-search artist filter (author.filing_name ~
// filing name) and includes the current artwork. A missing author or an empty
// filing name yields a nil holding because the filter link would be unusable.
func (r *RelatedWorkResolver) artistHolding(artwork *core.Record) (*RelatedWorkHolding, error) {
	authorIDs := artwork.GetStringSlice("author")
	if len(authorIDs) == 0 {
		return nil, nil
	}

	filingName, err := r.firstAuthorFilingName(authorIDs[0])
	if err != nil {
		return nil, err
	}
	if filingName == "" {
		return nil, nil
	}

	count, err := r.countArtworkHolding(artistFilingNameExp(filingName))
	if err != nil {
		return nil, err
	}

	return &RelatedWorkHolding{QueryKey: relatedHoldingArtist, QueryValue: filingName, Count: count}, nil
}

// collectionHolding counts published works held at the artwork's canonical
// current museum. Private collections and public non-museum locations never
// participate in the public related-work path, so they yield a nil holding. The
// count matches the artwork-search venue filter and includes the current artwork.
func (r *RelatedWorkResolver) collectionHolding(artwork *core.Record) (*RelatedWorkHolding, error) {
	location, err := r.canonicalPublicMuseum(artwork)
	if err != nil || location == nil {
		return nil, err
	}

	count, err := r.countArtworkHolding(dbx.NewExp(
		"Artworks.current_location_id = {:location_id}",
		dbx.Params{"location_id": location.Id},
	))
	if err != nil {
		return nil, err
	}

	return &RelatedWorkHolding{QueryKey: relatedHoldingVenue, QueryValue: location.Id, Count: count}, nil
}

// periodHolding counts published works catalogued under the artwork's art
// period. The count matches the artwork-search period facet and includes the
// current artwork. A missing art period yields a nil holding.
func (r *RelatedWorkResolver) periodHolding(artwork *core.Record) (*RelatedWorkHolding, error) {
	periodID := artwork.GetString("art_period_id")
	if periodID == "" {
		return nil, nil
	}

	count, err := r.countArtworkHolding(dbx.NewExp(
		"Artworks.art_period_id = {:period_id}",
		dbx.Params{"period_id": periodID},
	))
	if err != nil {
		return nil, err
	}

	return &RelatedWorkHolding{QueryKey: relatedHoldingPeriod, QueryValue: periodID, Count: count}, nil
}

// firstAuthorFilingName resolves the filing name of the given author, or an
// empty string when the artist is missing or has no filing name.
func (r *RelatedWorkResolver) firstAuthorFilingName(authorID string) (string, error) {
	artist, err := r.app.FindRecordById(constants.CollectionArtists, authorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return artist.GetString("filing_name"), nil
}

// countArtworkHolding counts published artworks that carry at least one author
// and match the supplied facet expression. The base predicates mirror the
// artwork search so a holding count always equals the search result total for
// the corresponding filter, including the current artwork.
func (r *RelatedWorkResolver) countArtworkHolding(facetExp dbx.Expression) (int, error) {
	total, err := r.app.CountRecords(
		constants.CollectionArtworks,
		dbx.NewExp("published = true"),
		dbx.NewExp("json_array_length(Artworks.author) > 0"),
		facetExp,
	)
	if err != nil {
		return 0, err
	}

	return int(total), nil
}

// artistFilingNameExp matches artworks whose author relation includes an artist
// whose filing name contains the supplied value, mirroring the artwork-search
// artist filter (author.filing_name ~ value).
func artistFilingNameExp(filingName string) dbx.Expression {
	return dbx.NewExp(
		"EXISTS (SELECT 1 FROM json_each(Artworks.author) je JOIN Artists a ON a.id = je.value WHERE a.filing_name LIKE {:artist_like} ESCAPE '\\')",
		dbx.Params{"artist_like": likeContains(filingName)},
	)
}

// likeContains returns the value wrapped for a case-insensitive SQL LIKE
// contains match, escaping LIKE wildcards the same way PocketBase's ~ operator
// does so a filing name cannot inject a wildcard.
func likeContains(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return "%" + replacer.Replace(value) + "%"
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
