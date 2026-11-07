package artworks

import (
	"slices"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	authorTestX = "artistx00000001"
	authorTestY = "artisty00000001"
	authorTestZ = "artistz00000001"

	workPrimary    = "workprimary0001"
	workCoAuthor   = "workcoauthor001"
	workLegacy     = "worklegacy00001"
	workReassigned = "workreassigned1"
	workNoAuthor   = "worknoauthor001"
)

// newAuthorTestApp builds artists and artworks with the primary, co-author,
// and legacy author relations, and five works covering each attribution case.
func newAuthorTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	artists := core.NewBaseCollection("Artists")
	artists.Id = constants.CollectionArtists
	artists.Fields.Add(&core.TextField{Name: "filing_name"}, &core.TextField{Name: "short_name"})
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}
	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = constants.CollectionArtworks
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.RelationField{Name: FieldLegacyAuthor, CollectionId: constants.CollectionArtists, MaxSelect: 10},
		&core.RelationField{Name: FieldPrimaryAuthor, CollectionId: constants.CollectionArtists, MaxSelect: 1},
		&core.RelationField{Name: FieldCoAuthors, CollectionId: constants.CollectionArtists, MaxSelect: 10},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	for id, fields := range map[string]map[string]any{
		authorTestX: {"filing_name": "XAVER, Anton", "short_name": "Anton Xaver"},
		authorTestY: {"filing_name": "YSELT, Berta", "short_name": "Berta Yselt"},
		authorTestZ: {"filing_name": "ZEDEK, Carl", "short_name": ""},
	} {
		saveAuthorTestRecord(t, app, artists, id, fields)
	}
	for id, fields := range map[string]map[string]any{
		workPrimary:    {"title": "Primary", FieldPrimaryAuthor: authorTestX, FieldLegacyAuthor: []string{authorTestX}},
		workCoAuthor:   {"title": "Co-author", FieldPrimaryAuthor: authorTestY, FieldCoAuthors: []string{authorTestX, authorTestY}},
		workLegacy:     {"title": "Legacy", FieldLegacyAuthor: []string{authorTestY, authorTestX}},
		workReassigned: {"title": "Reassigned", FieldPrimaryAuthor: authorTestZ, FieldLegacyAuthor: []string{authorTestX}},
		workNoAuthor:   {"title": "No author"},
	} {
		saveAuthorTestRecord(t, app, artworks, id, fields)
	}

	return app
}

func saveAuthorTestRecord(t *testing.T, app *tests.TestApp, collection *core.Collection, id string, fields map[string]any) {
	t.Helper()
	record := core.NewRecord(collection)
	record.Id = id
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", id, err)
	}
}

func filteredWorkIDs(t *testing.T, app *tests.TestApp, filter string, params dbx.Params) []string {
	t.Helper()
	records, err := app.FindRecordsByFilter(constants.CollectionArtworks, filter, "id", 0, 0, params)
	if err != nil {
		t.Fatalf("filter %q: %v", filter, err)
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.Id)
	}
	return ids
}

func TestAuthorIDsPrefersPrimaryAndFallsBackToLegacy(t *testing.T) {
	app := newAuthorTestApp(t)

	for id, want := range map[string][]string{
		workPrimary:    {authorTestX},
		workCoAuthor:   {authorTestY, authorTestX},
		workLegacy:     {authorTestY, authorTestX},
		workReassigned: {authorTestZ},
		workNoAuthor:   {},
	} {
		record, err := app.FindRecordById(constants.CollectionArtworks, id)
		if err != nil {
			t.Fatal(err)
		}
		if got := AuthorIDs(record); !slices.Equal(got, want) {
			t.Errorf("AuthorIDs(%s) = %v, want %v", id, got, want)
		}
		if got := HasAuthor(record); got != (len(want) > 0) {
			t.Errorf("HasAuthor(%s) = %v", id, got)
		}
		author := FindAttributedAuthor(app, record)
		if len(want) == 0 {
			if author != nil {
				t.Errorf("FindAttributedAuthor(%s) = %s, want nil", id, author.Id)
			}
		} else if author == nil || author.Id != want[0] {
			t.Errorf("FindAttributedAuthor(%s) = %v, want %s", id, author, want[0])
		}
	}
	if AuthorIDs(nil) != nil {
		t.Error("AuthorIDs(nil) must be nil")
	}
}

func TestAuthorFiltersMatchPrimaryCoAuthorAndLegacyFallback(t *testing.T) {
	app := newAuthorTestApp(t)

	if got, want := filteredWorkIDs(t, app, HasAuthorFilter, nil), []string{workCoAuthor, workLegacy, workPrimary, workReassigned}; !slices.Equal(got, want) {
		t.Errorf("HasAuthorFilter = %v, want %v", got, want)
	}

	// Artist X attributes the primary work, the co-authored work, and the
	// unreviewed legacy work, but not the work whose curated primary author
	// replaced a stale legacy assignment.
	byID := filteredWorkIDs(t, app, AuthorFieldFilter("id", "=", "{:artist}"), dbx.Params{"artist": authorTestX})
	if want := []string{workCoAuthor, workLegacy, workPrimary}; !slices.Equal(byID, want) {
		t.Errorf("AuthorFieldFilter(id = X) = %v, want %v", byID, want)
	}
	byName := filteredWorkIDs(t, app, AuthorFieldFilter("filing_name", "~", "{:query}"), dbx.Params{"query": "xaver"})
	if !slices.Equal(byName, byID) {
		t.Errorf("AuthorFieldFilter(filing_name ~ xaver) = %v, want %v", byName, byID)
	}

	// The attributed author must carry the identity: Z has no short name, and
	// the legacy work requires every legacy author to qualify.
	identity := filteredWorkIDs(t, app, AttributedAuthorFilter("filing_name != ''", "short_name != ''"), nil)
	if want := []string{workCoAuthor, workLegacy, workPrimary}; !slices.Equal(identity, want) {
		t.Errorf("AttributedAuthorFilter = %v, want %v", identity, want)
	}
}

func TestAuthorSQLExpressionsMatchAuthorIDs(t *testing.T) {
	app := newAuthorTestApp(t)

	rows := []struct {
		ID     string `db:"id"`
		First  string `db:"first_author"`
		Member string `db:"member"`
	}{}
	if err := app.DB().NewQuery(`
		SELECT Artworks.id AS id,
			COALESCE(` + FirstAuthorIDSQL("Artworks") + `, '') AS first_author,
			COALESCE((SELECT group_concat(value, ',') FROM (SELECT DISTINCT je.value FROM json_each(` + AuthorIDsSQL("Artworks") + `) je ORDER BY je.value)), '') AS member
		FROM Artworks
		ORDER BY Artworks.id
	`).All(&rows); err != nil {
		t.Fatalf("query author SQL: %v", err)
	}

	for _, row := range rows {
		record, err := app.FindRecordById(constants.CollectionArtworks, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		ids := AuthorIDs(record)
		wantFirst := ""
		if len(ids) > 0 {
			wantFirst = ids[0]
		}
		if row.First != wantFirst {
			t.Errorf("%s FirstAuthorIDSQL = %q, want %q", row.ID, row.First, wantFirst)
		}
		sorted := slices.Clone(ids)
		slices.Sort(sorted)
		wantMembers := ""
		for i, id := range sorted {
			if i > 0 {
				wantMembers += ","
			}
			wantMembers += id
		}
		if row.Member != wantMembers {
			t.Errorf("%s AuthorIDsSQL members = %q, want %q", row.ID, row.Member, wantMembers)
		}
	}
}
