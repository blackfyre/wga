package inspire

import (
	"fmt"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const inspirationLimit int64 = 10

// inspirationWorks returns a bounded shuffled slice whose first author is published.
func inspirationWorks(app *pocketbase.PocketBase) (dto.ImageGrid, error) {
	collection, err := app.FindCollectionByNameOrId(constants.CollectionArtworks)
	if err != nil {
		return nil, fmt.Errorf("find artworks collection: %w", err)
	}

	rows, err := app.DB().NewQuery(`
		SELECT artworks.id, artworks.title, artworks.image, artworks.image_width,
			artworks.comment, artworks.technique, artists.id, artists.name
		FROM artworks
		INNER JOIN artists ON json_extract(artworks.author, '$[0]') = artists.id
		WHERE artworks.published = TRUE AND artists.published = TRUE
		ORDER BY RANDOM()
		LIMIT 10`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query inspiration works: %w", err)
	}
	defer rows.Close()

	content := make(dto.ImageGrid, 0, inspirationLimit)
	for rows.Next() {
		var artworkID, title, imageName, comment, technique, authorID, authorName string
		var imageWidth int
		if err := rows.Scan(&artworkID, &title, &imageName, &imageWidth, &comment, &technique, &authorID, &authorName); err != nil {
			return nil, fmt.Errorf("scan inspiration work: %w", err)
		}

		artwork := core.NewRecord(collection)
		artwork.Id = artworkID
		artwork.Set("image", imageName)
		artwork.Set("image_width", imageWidth)

		image := utils.AssetUrl("/assets/images/no-image.png")
		if imageName != "" {
			image = url.GenerateArtworkImageURL(artwork, url.DeliveryProfileCardAndArtistIndex, "")
		}

		content = append(content, dto.Image{
			Id:        artworkID,
			Image:     image,
			Thumb:     image,
			Title:     title,
			Technique: technique,
			Comment:   comment,
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId:     authorID,
				ArtistName:   authorName,
				ArtworkId:    artworkID,
				ArtworkTitle: title,
			}),
			Artist: dto.Artist{
				Id:   authorID,
				Name: authorName,
				Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
					ArtistId:   authorID,
					ArtistName: authorName,
				}),
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read inspiration works: %w", err)
	}

	return content, nil
}
