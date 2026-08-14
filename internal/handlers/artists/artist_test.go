package artists

import (
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/utils"
)

func TestArtistOpenGraphImagePrefersPortrait(t *testing.T) {
	content := dto.Artist{
		Portrait: "/api/files/artists/artist/portrait.jpg",
		Works: dto.ImageGrid{{
			Image: "/api/files/artworks/artwork/artwork.jpg",
		}},
	}

	if got, want := artistOpenGraphImage(content), utils.AssetUrl(content.Portrait); got != want {
		t.Fatalf("Open Graph image = %q, want %q", got, want)
	}
}

func TestArtistOpenGraphImageFallsBackToFirstArtwork(t *testing.T) {
	content := dto.Artist{Works: dto.ImageGrid{{Image: "/api/files/artworks/artwork/artwork.jpg"}}}
	if got, want := artistOpenGraphImage(content), utils.AssetUrl(content.Works[0].Image); got != want {
		t.Fatalf("Open Graph image = %q, want %q", got, want)
	}
}
