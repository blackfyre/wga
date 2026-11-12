package jsonld

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

func TestArtistJsonLdIncludesPortrait(t *testing.T) {
	artist := testArtistRecord("portrait.jpg")
	if got, want := ArtistJsonLd(artist).Image, utils.AssetUrl(urlutils.GenerateFileUrl(constants.CollectionArtists, artist.Id, "portrait.jpg", "")); got != want {
		t.Fatalf("JSON-LD image = %q, want %q", got, want)
	}
}

func TestArtistJsonLdResolvesRecordProfilePortrait(t *testing.T) {
	artist := testArtistRecord("portrait.jpg")
	artist.Set("biography_image_width", 601)
	if got, want := ArtistJsonLd(artist).Image, utils.AssetUrl(urlutils.GenerateDeliveryURL(constants.CollectionArtists, artist.Id, "portrait.jpg", 601, urlutils.DeliveryProfilePortraitRecordAndWorkFallback, "")); got != want {
		t.Fatalf("JSON-LD image = %q, want %q", got, want)
	}
}

func TestArtistJsonLdOmitsMissingPortrait(t *testing.T) {
	artist := testArtistRecord("")
	if got := ArtistJsonLd(artist).Image; got != "" {
		t.Fatalf("JSON-LD image = %q, want empty", got)
	}
}

func TestArtistJsonLdOmitsUnknownDatesAndEmptyFields(t *testing.T) {
	artist := testArtistRecord("")
	artist.Set("year_of_birth", 0)
	artist.Set("year_of_death", 0)

	person := ArtistJsonLd(artist)
	if person.BirthDate != "" {
		t.Errorf("BirthDate = %q, want empty for unknown birth year", person.BirthDate)
	}
	if person.DeathDate != "" {
		t.Errorf("DeathDate = %q, want empty for unknown death year", person.DeathDate)
	}
	if person.BirthPlace != nil {
		t.Errorf("BirthPlace = %+v, want omitted", person.BirthPlace)
	}
	if person.HasOccupation != nil {
		t.Errorf("HasOccupation = %+v, want omitted", person.HasOccupation)
	}
	tag, err := ScriptTag(person)
	if err != nil {
		t.Fatalf("serialise: %v", err)
	}
	for _, unwanted := range []string{"{}", `"birthPlace"`, `"deathPlace"`, `"hasOccupation"`, `"image"`} {
		if strings.Contains(tag, unwanted) {
			t.Errorf("serialised Person contains %s: %s", unwanted, tag)
		}
	}
}

func TestArtistJsonLdKeepsTruthfulDates(t *testing.T) {
	artist := testArtistRecord("")
	artist.Set("year_of_birth", 1606)
	artist.Set("year_of_death", 1669)

	person := ArtistJsonLd(artist)
	if person.BirthDate != "1606" {
		t.Errorf("BirthDate = %q, want 1606", person.BirthDate)
	}
	if person.DeathDate != "1669" {
		t.Errorf("DeathDate = %q, want 1669", person.DeathDate)
	}
}

func TestArtistJsonLdUsesFilingName(t *testing.T) {
	artist := testArtistRecord("")
	if got, want := ArtistJsonLd(artist).Name, "ARTIST, Portrait"; got != want {
		t.Fatalf("JSON-LD name = %q, want filing form %q", got, want)
	}
}

func TestArtistJsonLdOmitsBlankFilingName(t *testing.T) {
	artist := testArtistRecord("")
	artist.Set("filing_name", "")
	if got := ArtistJsonLd(artist).Name; got != "" {
		t.Fatalf("JSON-LD name = %q, want empty (blank identity omitted)", got)
	}
}

func testArtistRecord(portrait string) *core.Record {
	artists := core.NewBaseCollection("Artists")
	artists.Id = constants.CollectionArtists
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "portrait"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "place_of_birth"},
		&core.TextField{Name: "place_of_death"},
		&core.TextField{Name: "profession"},
	)

	artist := core.NewRecord(artists)
	artist.Id = "artist"
	artist.Set("name", "Legacy Name")
	artist.Set("filing_name", "ARTIST, Portrait")
	artist.Set("short_name", "Portrait")
	artist.Set("slug", "portrait-artist")
	artist.Set("portrait", portrait)
	return artist
}

func TestArtistJsonLdIdentifiesCanonicalRecord(t *testing.T) {
	person := ArtistJsonLd(testArtistRecord(""))
	want := utils.AssetUrl("/artists/portrait-artist-artist")
	if person.Url != want || person.ID != want+"#primary" || person.Type != "Person" {
		t.Fatalf("Person identity = %q %q %q, want url %q with #primary @id", person.Type, person.ID, person.Url, want)
	}
}

func TestArtworkJsonLdDescribesVisibleRecord(t *testing.T) {
	artist := testArtistRecord("")
	artwork := ArtworkJsonLd(ArtworkInput{
		CanonicalURL: "https://gallery.example/artists/portrait-artist-artist/annunciation-a1",
		Title:        " Annunciation ",
		Commentary:   "<p>First.</p><p>Second &amp; last.</p>",
		ImageURL:     "https://gallery.example/images/a1.jpg",
		Technique:    "Tempera on wood",
		ArtType:      "painting",
		Year:         "1426",
		Artist:       artist,
	})

	for name, check := range map[string][2]string{
		"@type":       {artwork.Type, "VisualArtwork"},
		"@id":         {artwork.ID, "https://gallery.example/artists/portrait-artist-artist/annunciation-a1#primary"},
		"name":        {artwork.Name, "Annunciation"},
		"url":         {artwork.Url, "https://gallery.example/artists/portrait-artist-artist/annunciation-a1"},
		"description": {artwork.Description, "First. Second & last."},
		"image":       {artwork.Image, "https://gallery.example/images/a1.jpg"},
		"artform":     {artwork.Artform, "painting"},
		"artMedium":   {artwork.ArtMedium, "Tempera on wood"},
		"dateCreated": {artwork.DateCreated, "1426"},
	} {
		if check[0] != check[1] {
			t.Errorf("%s = %q, want %q", name, check[0], check[1])
		}
	}
	if artwork.Creator == nil || artwork.Creator.ID != ArtistURL(artist)+"#primary" || artwork.Creator.Name != "ARTIST, Portrait" {
		t.Fatalf("creator = %+v, want reference to the artist record", artwork.Creator)
	}
	if artwork.Creator.Description != "" || artwork.Creator.BirthPlace != nil {
		t.Errorf("creator must be a reference, not the full artist profile: %+v", artwork.Creator)
	}
}

func TestArtworkJsonLdOmitsUnparseableYearAndMissingImage(t *testing.T) {
	artwork := ArtworkJsonLd(ArtworkInput{CanonicalURL: "https://gallery.example/a", Title: "Untitled", Year: "c. 1500"})
	if artwork.DateCreated != "" {
		t.Errorf("dateCreated = %q, want omitted for a non-ISO year", artwork.DateCreated)
	}
	tag, err := ScriptTag(artwork)
	if err != nil {
		t.Fatalf("serialise: %v", err)
	}
	for _, unwanted := range []string{`"image"`, `"creator"`, "{}"} {
		if strings.Contains(tag, unwanted) {
			t.Errorf("serialised artwork contains %s: %s", unwanted, tag)
		}
	}
}

func TestBreadcrumbsRequireTwoSteps(t *testing.T) {
	if Breadcrumbs("https://gallery.example/x", []Crumb{{Name: "Only", URL: "https://gallery.example/x"}}) != nil {
		t.Fatal("a single step must not produce a BreadcrumbList")
	}
	list := Breadcrumbs("https://gallery.example/x", []Crumb{
		{Name: "Artists", URL: "https://gallery.example/artists"},
		{Name: "Giotto", URL: "https://gallery.example/x"},
	})
	if list == nil || list.ID != "https://gallery.example/x#breadcrumb" || len(list.ItemListElement) != 2 {
		t.Fatalf("breadcrumbs = %+v", list)
	}
	if second := list.ItemListElement[1]; second.Position != 2 || second.Name != "Giotto" || second.Item != "https://gallery.example/x" {
		t.Errorf("second item = %+v", second)
	}
}

func TestScriptTagEscapesMarkupAndSkipsNilNodes(t *testing.T) {
	var missing *BreadcrumbList
	tag, err := ScriptTag(Person{Type: "Person", Name: "</script><script>alert(1)</script>"}, missing)
	if err != nil {
		t.Fatalf("serialise: %v", err)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(tag, `<script type="application/ld+json">`), `</script>`)
	if strings.Contains(body, "<") {
		t.Fatalf("JSON-LD body contains a literal '<': %s", body)
	}
	var document struct {
		Context string           `json:"@context"`
		Graph   []map[string]any `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(body), &document); err != nil {
		t.Fatalf("JSON-LD body is not JSON: %v", err)
	}
	if document.Context != "https://schema.org" || len(document.Graph) != 1 || document.Graph[0]["name"] != "</script><script>alert(1)</script>" {
		t.Fatalf("document = %+v", document)
	}
	if empty, err := ScriptTag(missing); err != nil || empty != "" {
		t.Fatalf("empty graph = %q, %v; want no output", empty, err)
	}
}

func TestPlainTextSeparatesBlocks(t *testing.T) {
	if got := PlainText("<p>One.</p><p>Two<br>three</p>"); got != "One. Two three" {
		t.Fatalf("PlainText = %q", got)
	}
}

func TestSelectionJsonLdListsShownWorksInOrder(t *testing.T) {
	list := SelectionJsonLd("https://gallery.example/s", "Dürer: Paintings", []SelectionItem{
		{Name: "First", URL: "https://gallery.example/a"},
		{Name: " ", URL: "https://gallery.example/blank"},
		{Name: "Second", URL: "https://gallery.example/b"},
	})
	if list.Type != "ItemList" || list.ID != "https://gallery.example/s#primary" || list.Name != "Dürer: Paintings" || list.NumberOfItems != 2 {
		t.Fatalf("list = %+v", list)
	}
	if second := list.ItemListElement[1]; second.Position != 2 || second.Name != "Second" || second.Item != "https://gallery.example/b" {
		t.Errorf("second item = %+v", second)
	}
}
