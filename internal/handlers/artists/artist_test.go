package artists

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/glossary"
	"github.com/pocketbase/pocketbase/core"
)

func TestArtistRecordStepFailureLogIsDiagnosableAndRedacted(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))

	logArtistRecordStepFailure(logger, "list_published_works", time.Now(), errors.New("token=secret-value"))

	log := output.String()
	for _, expected := range []string{
		`"event":"artists.record_view.step_failed"`,
		`"step":"list_published_works"`,
		`"duration_ms":`,
		`"error_type":"*errors.errorString"`,
		`"error":"[REDACTED]"`,
	} {
		if !strings.Contains(log, expected) {
			t.Errorf("log missing %q: %s", expected, log)
		}
	}
	if strings.Contains(log, "secret-value") {
		t.Errorf("log must redact error content: %s", log)
	}
}

func TestArtistOpenGraphImagePrefersPortrait(t *testing.T) {
	view := pages.ArtistView{
		Portrait: "/api/files/artists/artist/portrait.jpg",
		Works: dto.ImageGrid{{
			Image: "/api/files/artworks/artwork/artwork.jpg",
		}},
	}

	if got, want := artistOpenGraphImage(view), utils.AssetUrl(view.Portrait); got != want {
		t.Fatalf("Open Graph image = %q, want %q", got, want)
	}
}

func TestArtistOpenGraphImageFallsBackToFirstArtwork(t *testing.T) {
	view := pages.ArtistView{Works: dto.ImageGrid{{Image: "/api/files/artworks/artwork/artwork.jpg"}}}
	if got, want := artistOpenGraphImage(view), utils.AssetUrl(view.Works[0].Image); got != want {
		t.Fatalf("Open Graph image = %q, want %q", got, want)
	}
}

func TestArtistOpenGraphImageEmptyWithoutPortraitOrWorks(t *testing.T) {
	if got := artistOpenGraphImage(pages.ArtistView{}); got != "" {
		t.Fatalf("Open Graph image = %q, want empty", got)
	}
}

func TestArtistLifeSummaryIncludesBirthAndDeath(t *testing.T) {
	artist := testArtistRecord()
	artist.Set("year_of_birth", 1606)
	artist.Set("year_of_death", 1669)
	artist.Set("place_of_birth", "Leiden")
	artist.Set("place_of_death", "Amsterdam")

	if got, want := artistLifeSummary(artist), "b. 1606 Leiden · d. 1669 Amsterdam"; got != want {
		t.Fatalf("life summary = %q, want %q", got, want)
	}
}

func TestArtistLifeSummaryOmitsMissingValues(t *testing.T) {
	artist := testArtistRecord()
	artist.Set("year_of_birth", 1606)

	if got, want := artistLifeSummary(artist), "b. 1606"; got != want {
		t.Fatalf("life summary = %q, want %q", got, want)
	}

	empty := testArtistRecord()
	if got := artistLifeSummary(empty); got != "" {
		t.Fatalf("life summary = %q, want empty", got)
	}
}

func TestBuildArtistWorksURLUsesExactID(t *testing.T) {
	if got, want := buildArtistWorksURL("artistone000001"), "/artworks?artist_id=artistone000001"; got != want {
		t.Fatalf("works URL = %q, want %q", got, want)
	}
}

func TestBuildArtistNameWorksURLEncodesName(t *testing.T) {
	if got, want := buildArtistNameWorksURL("Johannes Vermeer"), "/artworks?artist=Johannes+Vermeer"; got != want {
		t.Fatalf("works URL = %q, want %q", got, want)
	}
}

func TestBuildArtistCitationUsesCanonicalAbsoluteURL(t *testing.T) {
	artist := testArtistRecord()
	artist.Set("name", "portrait-artist")
	artist.Set("filing_name", "Artist, Portrait")
	artist.Set("short_name", "Portrait")
	artist.Set("slug", "portrait-artist")

	citation := buildArtistCitation(artist, "portrait-artist-artist")
	if citation.Key != "wga-portrait-artist" {
		t.Errorf("key = %q, want wga-portrait-artist", citation.Key)
	}
	if citation.Title != "Artist, Portrait" {
		t.Errorf("title = %q, want Artist, Portrait", citation.Title)
	}
	if want := utils.AssetUrl("/artists/portrait-artist-artist"); citation.URL != want {
		t.Errorf("URL = %q, want %q", citation.URL, want)
	}
}

func TestBuildRecordMusicOmitsWithoutSource(t *testing.T) {
	if got := buildRecordMusic(nil); got != (components.MusicPeriodCard{}) {
		t.Errorf("nil song = %#v, want empty", got)
	}

	song := &repositories.PeriodSong{Record: testMusicRecord(""), Composer: "Sweelinck"}
	if got := buildRecordMusic(song); got != (components.MusicPeriodCard{}) {
		t.Errorf("sourceless song = %#v, want empty", got)
	}
}

func TestBuildRecordMusicUsesPlayerRoute(t *testing.T) {
	song := &repositories.PeriodSong{Record: testMusicRecord("fantasia.mp3"), Composer: "Sweelinck"}

	got := buildRecordMusic(song)
	if got.Piece != "Fantasia chromatica" {
		t.Errorf("piece = %q, want Fantasia chromatica", got.Piece)
	}
	if got.SongID != song.Record.Id {
		t.Errorf("song id = %q, want %q", got.SongID, song.Record.Id)
	}
	if want := "/player?song=" + song.Record.Id; got.PlayerURL != want {
		t.Errorf("player URL = %q, want %q", got.PlayerURL, want)
	}
}

func TestBuildRecordWorkImagesUsesCanonicalRecordLinks(t *testing.T) {
	artist := testArtistRecord()
	artist.Set("name", "portrait-artist")
	artist.Set("filing_name", "Artist, Portrait")
	artist.Set("short_name", "Portrait")

	work := testArtworkRecord("painting.jpg", 800)
	work.Id = "artwork12345678"
	work.Set("title", "A Painting")

	images := buildRecordWorkImages(artist, []*core.Record{work})
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}

	image := images[0]
	if want := "/artists/portrait-artist-artist/a-painting-artwork12345678"; image.Url != want {
		t.Errorf("record URL = %q, want %q", image.Url, want)
	}
	if image.Image != "/api/files/artworks/artwork12345678/painting.jpg?thumb=500x0" {
		t.Errorf("image = %q, want 500-profile thumb", image.Image)
	}
	if image.Artist.FilingName != "Artist, Portrait" {
		t.Errorf("artist filing name = %q, want Artist, Portrait", image.Artist.FilingName)
	}
	if image.Zoom != "" {
		t.Errorf("zoom = %q, want empty (no viewer hooks)", image.Zoom)
	}
}

func TestBuildRecordWorkImagesFallsBackWithoutImage(t *testing.T) {
	artist := testArtistRecord()
	artist.Set("name", "Portrait Artist")
	work := testArtworkRecord("", 0)
	work.Id = "artwork12345678"

	images := buildRecordWorkImages(artist, []*core.Record{work})
	if got, want := images[0].Image, utils.AssetUrl("/assets/images/no-image.png"); got != want {
		t.Errorf("image = %q, want no-image fallback %q", got, want)
	}
}

func testArtistRecord() *core.Record {
	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "slug"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "place_of_birth"},
		&core.TextField{Name: "place_of_death"},
	)

	record := core.NewRecord(artists)
	record.Id = "artist"
	return record
}

func testArtworkRecord(image string, width int) *core.Record {
	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.Fields.Add(
		&core.TextField{Name: "title"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
	)

	record := core.NewRecord(artworks)
	record.Id = "artwork12345678"
	record.Set("title", "A Painting")
	record.Set("image", image)
	record.Set("image_width", width)
	return record
}

func TestUnambiguousPeriodName(t *testing.T) {
	collection := core.NewBaseCollection("Art_periods")
	collection.Id = "art_periods"
	collection.Fields.Add(&core.TextField{Name: "name"})

	makeRecord := func(name string) *core.Record {
		record := core.NewRecord(collection)
		record.Id = "period" + name
		record.Set("name", name)
		return record
	}

	if got := unambiguousPeriodName(nil); got != "" {
		t.Errorf("nil periods = %q, want empty", got)
	}
	if got := unambiguousPeriodName([]*core.Record{makeRecord("Baroque")}); got != "Baroque" {
		t.Errorf("single period = %q, want Baroque", got)
	}
	if got := unambiguousPeriodName([]*core.Record{makeRecord("Baroque"), makeRecord("Rococo")}); got != "" {
		t.Errorf("two periods = %q, want empty (ambiguous)", got)
	}
}

func testMusicRecord(source string) *core.Record {
	songs := core.NewBaseCollection("Music_song")
	songs.Id = "music_song"
	songs.Fields.Add(
		&core.TextField{Name: "title"},
		&core.TextField{Name: "source"},
	)

	record := core.NewRecord(songs)
	record.Id = "song1234567890a"
	record.Set("title", "Fantasia chromatica")
	record.Set("source", source)
	return record
}

func TestAnnotateBiographyRendersTask44Contract(t *testing.T) {
	entries := []glossary.GlossaryEntry{
		{MatchTerm: "fresco", Definition: "A wall painting technique."},
	}
	got := annotateBiography("<p>A fresco on the wall.</p><script>alert(1)</script>", entries)

	if strings.Contains(got, "<script") {
		t.Error("script must be removed")
	}
	if !strings.Contains(got, `<dfn class="wga-term"`) {
		t.Errorf("expected task-4.4 dfn, got %q", got)
	}
	if !strings.Contains(got, `aria-label="fresco: A wall painting technique."`) {
		t.Errorf("expected complete accessible name, got %q", got)
	}
	if strings.Contains(got, "glossary-term") {
		t.Errorf("legacy annotation must not remain, got %q", got)
	}
}

func TestAnnotateBiographySkipsLinkText(t *testing.T) {
	entries := []glossary.GlossaryEntry{{MatchTerm: "fresco", Definition: "A wall painting technique."}}
	got := annotateBiography(`<p>A <a href="/x">fresco</a> and a fresco.</p>`, entries)

	if count := strings.Count(got, `class="wga-term"`); count != 1 {
		t.Errorf("expected exactly one annotated term, got %d in %q", count, got)
	}
}

func TestAnnotateBiographyDegradesWithoutEntries(t *testing.T) {
	got := annotateBiography("<p>He used fresco.</p><script>alert(1)</script>", nil)
	if strings.Contains(got, "<script") {
		t.Error("script must be removed")
	}
	if !strings.Contains(got, "<p>He used fresco.</p>") {
		t.Errorf("expected readable biography, got %q", got)
	}
}
