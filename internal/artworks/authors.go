package artworks

import (
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

// Artwork authorship fields. primary_author is the single primary artist and
// co_authors the additional artists. The legacy author relation is read only
// while primary_author is empty: the legacy-author migration leaves artworks
// with several legacy authors unassigned for curator review, and they must stay
// attributed until that review and the legacy-field removal are complete.
const (
	FieldPrimaryAuthor = "primary_author"
	FieldCoAuthors     = "co_authors"
	FieldLegacyAuthor  = "author"
)

// AuthorIDs returns an artwork's artist ids in attribution order: the primary
// author followed by co-authors, without duplicates. When no primary author is
// assigned it returns the legacy author relation instead.
func AuthorIDs(artwork *core.Record) []string {
	if artwork == nil {
		return nil
	}

	primary := strings.TrimSpace(artwork.GetString(FieldPrimaryAuthor))
	if primary == "" {
		return uniqueIDs(artwork.GetStringSlice(FieldLegacyAuthor))
	}

	return uniqueIDs(append([]string{primary}, artwork.GetStringSlice(FieldCoAuthors)...))
}

// HasAuthor reports whether the artwork carries any author under AuthorIDs.
func HasAuthor(artwork *core.Record) bool {
	return len(AuthorIDs(artwork)) > 0
}

func uniqueIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

// AuthorIDsSQL returns a SQLite expression evaluating to the JSON array of the
// artwork's artist ids under AuthorIDs for the given table name or alias. It
// is intended for json_each(...) membership tests; when a primary author is
// assigned the array holds the co-authors followed by the primary author, so
// callers that need attribution order must use FirstAuthorIDSQL instead.
func AuthorIDsSQL(table string) string {
	return "(CASE WHEN COALESCE(" + table + ".primary_author, '') != '' " +
		"THEN json_insert(CASE WHEN json_valid(" + table + ".co_authors) THEN " + table + ".co_authors ELSE '[]' END, '$[#]', " + table + ".primary_author) " +
		"ELSE (CASE WHEN json_valid(" + table + ".author) THEN " + table + ".author ELSE '[]' END) END)"
}

// FirstAuthorIDSQL returns a SQLite expression evaluating to the artwork's
// attributed artist id: the primary author, or the first legacy author while
// no primary author is assigned.
func FirstAuthorIDSQL(table string) string {
	return "COALESCE(NULLIF(" + table + ".primary_author, ''), " +
		"(CASE WHEN json_valid(" + table + ".author) THEN json_extract(" + table + ".author, '$[0]') END))"
}

// HasAuthorFilter is the PocketBase filter expression matching artworks with
// any author under AuthorIDs.
const HasAuthorFilter = "(primary_author != '' || author:length > 0)"

// AuthorFieldFilter returns a PocketBase filter expression matching artworks
// where any author under AuthorIDs satisfies "<field> <op> <placeholder>", for
// example AuthorFieldFilter("filing_name", "~", "{:query}"). The operator must
// be a plain PocketBase comparison operator without the "?" any-of prefix.
func AuthorFieldFilter(field string, op string, placeholder string) string {
	return "((primary_author != '' && (primary_author." + field + " " + op + " " + placeholder +
		" || co_authors." + field + " ?" + op + " " + placeholder + "))" +
		" || (primary_author = '' && author." + field + " ?" + op + " " + placeholder + "))"
}

// AttributedAuthorFilter returns a PocketBase filter expression requiring the
// artwork's attributed author to satisfy every condition: the primary author
// when assigned, otherwise every legacy author. Each condition is a field
// comparison such as "filing_name != ”".
func AttributedAuthorFilter(conditions ...string) string {
	primary := make([]string, 0, len(conditions)+1)
	legacy := make([]string, 0, len(conditions)+1)
	primary = append(primary, "primary_author != ''")
	legacy = append(legacy, "primary_author = ''", "author:length > 0")
	for _, condition := range conditions {
		primary = append(primary, "primary_author."+condition)
		legacy = append(legacy, "author."+condition)
	}

	return "((" + strings.Join(primary, " && ") + ") || (" + strings.Join(legacy, " && ") + "))"
}

// FindAttributedAuthor returns the artist record of the artwork's first author
// under AuthorIDs, or nil when the artwork has no author or the artist record
// cannot be found.
func FindAttributedAuthor(app core.App, artwork *core.Record) *core.Record {
	ids := AuthorIDs(artwork)
	if len(ids) == 0 {
		return nil
	}
	artist, err := app.FindRecordById(constants.CollectionArtists, ids[0])
	if err != nil {
		return nil
	}

	return artist
}
