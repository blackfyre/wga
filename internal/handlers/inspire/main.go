package inspire

import (
	"bytes"
	"context"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func inspirationHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	artPieces, err := app.FindRecordsByFilter(constants.CollectionArtworks, "published = true", "@random", 50, 0, dbx.Params{})

	if err != nil {
		app.Logger().Error("Error getting random artworks", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	content := dto.ImageGrid{}

	for _, artPiece := range artPieces {
		if len(content) == 10 {
			break
		}

		artworkId := artPiece.GetString("id")
		authorIds := artPiece.GetStringSlice("author")
		authorId := ""

		for _, id := range authorIds {
			if id != "" {
				authorId = id
				break
			}
		}

		if authorId == "" {
			app.Logger().Warn("Skipping artwork without author", "artworkId", artworkId)
			continue
		}

		artist, err := app.FindRecordById(constants.CollectionArtists, authorId)

		if err != nil {
			app.Logger().Error("Error getting artist for artwork", "artworkId", artworkId, "error", err.Error())
			continue
		}

		imageUrl := utils.AssetUrl("/assets/images/no-image.png")
		thumbUrl := imageUrl
		imageName := artPiece.GetString("image")

		if imageName != "" {
			imageUrl = url.GenerateFileUrl(constants.CollectionArtworks, artworkId, imageName, "")
			thumbUrl = url.GenerateThumbUrl(constants.CollectionArtworks, artworkId, imageName, "320x240", "")
		}

		content = append(content, dto.Image{
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId:     artist.GetString("id"),
				ArtistName:   artist.GetString("name"),
				ArtworkTitle: artPiece.GetString("title"),
				ArtworkId:    artPiece.GetString("id"),
			}),
			Image:     imageUrl,
			Thumb:     thumbUrl,
			Comment:   artPiece.GetString("comment"),
			Title:     artPiece.GetString("title"),
			Technique: artPiece.GetString("technique"),
			Id:        artworkId,
			Artist: dto.Artist{
				Id:   artist.Id,
				Name: artist.GetString("name"),
				Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
					ArtistId:   artist.Id,
					ArtistName: artist.GetString("name"),
				}),
				Profession: artist.GetString("profession"),
			},
		})
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Inspiration")

	var buff bytes.Buffer

	c.Response.Header().Set("HX-Push-Url", "/inspire")
	err = pages.InspirePage(content).Render(ctx, &buff)

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

// RegisterHandlers registers the HTTP handlers for the PocketBase application.
// It binds a function to the OnServe event, which sets up a GET route for "/inspire".
// When the "/inspire" route is accessed, the inspirationHandler function is called.
//
// Parameters:
//   - app: A pointer to the PocketBase application instance.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/inspire", func(c *core.RequestEvent) error {
			return inspirationHandler(app, c)
		})
		return se.Next()
	})
}
