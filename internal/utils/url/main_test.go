package url

import "testing"

func TestGenerateThumbUrlUsesVisualRendition(t *testing.T) {
	for _, test := range []struct {
		name string
		size string
		want string
	}{
		{name: "row", size: ThumbnailArtworkRow, want: "200x0"},
		{name: "related card", size: ThumbnailArtworkCardSmall, want: "400x0"},
		{name: "card", size: ThumbnailArtworkCard, want: "500x0"},
		{name: "postcard", size: ThumbnailArtworkPostcard, want: "700x0"},
		{name: "feature", size: ThumbnailArtworkFeature, want: "900x0"},
		{name: "plate", size: ThumbnailArtworkPlate, want: "1400x0"},
		{name: "zoom", size: ThumbnailArtworkZoom, want: "2000x0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GenerateThumbUrl("artworks", "artwork-id", "work.jpg", test.size, "")
			want := "/api/files/artworks/artwork-id/work.jpg?thumb=" + test.want
			if got != want {
				t.Fatalf("GenerateThumbUrl() = %q, want %q", got, want)
			}
		})
	}
}

func TestGenerateThumbUrlPreservesToken(t *testing.T) {
	got := GenerateThumbUrl("artists", "artist-id", "portrait.jpg", ThumbnailPortraitRecord, "token")
	want := "/api/files/artists/artist-id/portrait.jpg?thumb=600x0&token=token"
	if got != want {
		t.Fatalf("GenerateThumbUrl() = %q, want %q", got, want)
	}
}
