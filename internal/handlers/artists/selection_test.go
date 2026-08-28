package artists

import (
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

func TestBuildSelectionURLUsesDeterministicIdentity(t *testing.T) {
	got := buildSelectionURL("durer-artistone000001", "rselect00000001")
	want := "/artists/durer-artistone000001/selections/rselect00000001"
	if got != want {
		t.Fatalf("buildSelectionURL = %q, want %q", got, want)
	}
}

func TestSanitizeSelectionCommentaryStripsScripts(t *testing.T) {
	got := sanitizeSelectionCommentary("<p>Safe prose.</p><script>alert(1)</script><a href=\"javascript:alert(2)\">x</a>")

	if !strings.Contains(got, "<p>Safe prose.</p>") {
		t.Fatalf("sanitized commentary = %q, want safe prose retained", got)
	}
	if strings.Contains(got, "<script") {
		t.Fatalf("sanitized commentary = %q, want script removed", got)
	}
	if strings.Contains(got, "javascript:") {
		t.Fatalf("sanitized commentary = %q, want unsafe URL removed", got)
	}
}

func TestSelectionDescriptionPrefersCommentary(t *testing.T) {
	view := pages.SelectionView{Commentary: "<p>An editorial lede.</p>", ArtistShortName: "Dürer"}
	if got := selectionDescription(view); got != "An editorial lede." {
		t.Fatalf("description = %q, want stripped commentary", got)
	}

	empty := pages.SelectionView{ArtistShortName: "Dürer"}
	if got := selectionDescription(empty); got != "Curated selection of works by Dürer" {
		t.Fatalf("description = %q, want fallback", got)
	}
}

func TestBuildSelectionURLComposesWithArtistSlug(t *testing.T) {
	// The artist slug itself contains the deterministic artist identity; the
	// selection segment is the producer identity with no derived slug.
	if got := buildSelectionURL("van-gogh-artistone00001", "rselect00000009"); got != "/artists/van-gogh-artistone00001/selections/rselect00000009" {
		t.Fatalf("buildSelectionURL = %q", got)
	}
	if got := utils.ExtractIdFromString("van-gogh-artistone00001"); got != "artistone00001" {
		t.Fatalf("artist identity extraction = %q", got)
	}
}

func TestBuildSelectionCitationUsesPersistedProducerIdentity(t *testing.T) {
	artistCollection := core.NewBaseCollection("Artists")
	artistCollection.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "slug"})
	artist := core.NewRecord(artistCollection)
	artist.Id = "artistone000001"
	artist.Set("name", "Dürer")
	artist.Set("slug", "durer")

	selectionCollection := core.NewBaseCollection("Art_selections")
	selectionCollection.Fields.Add(&core.TextField{Name: "display_title"})
	selection := core.NewRecord(selectionCollection)
	selection.Id = "r2633f9d80c78f0"
	selection.Set("display_title", "Dürer: Paintings")

	citation := buildSelectionCitation(artist, selection)

	if want := "wga-r2633f9d80c78f0"; citation.Key != want {
		t.Errorf("citation key = %q, want %q", citation.Key, want)
	}
	if want := "Dürer: Paintings (selection)"; citation.Title != want {
		t.Errorf("citation title = %q, want %q", citation.Title, want)
	}
	if want := utils.AssetUrl("/artists/durer-artistone000001/selections/r2633f9d80c78f0"); citation.URL != want {
		t.Errorf("citation URL = %q, want %q", citation.URL, want)
	}
}

func TestBuildSelectionCitationDoesNotInferTitle(t *testing.T) {
	artistCollection := core.NewBaseCollection("Artists")
	artistCollection.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "slug"})
	artist := core.NewRecord(artistCollection)
	artist.Id = "artistone000001"
	artist.Set("name", "Dürer")
	artist.Set("slug", "durer")

	selectionCollection := core.NewBaseCollection("Art_selections")
	selectionCollection.Fields.Add(&core.TextField{Name: "display_title"})
	selection := core.NewRecord(selectionCollection)
	selection.Id = "rselect00000001"
	selection.Set("display_title", "Paintings")

	citation := buildSelectionCitation(artist, selection)

	// The citation title is the supplied display title plus the "(selection)"
	// qualifier; it never derives a form from the title, path, or artwork data.
	if want := "Paintings (selection)"; citation.Title != want {
		t.Errorf("citation title = %q, want %q", citation.Title, want)
	}
	if strings.Contains(citation.Title, "Dürer") {
		t.Error("citation title must not inject the artist name into the supplied display title")
	}
}
