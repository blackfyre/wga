package inspire

import (
	"bytes"
	"context"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
)

func inspirationHandler(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	artPieces, err := app.FindRecordsByFilter("artists", "random", "", 10, 0)

	if err != nil {
		app.Logger().Error("Error getting random artworks", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	content := dto.ImageGrid{}

	for _, artPiece := range artPieces {

		artworkId := artPiece.GetString("id")

		artist, err := app.FindRecordById("artists", artPiece.GetString("author"))

		if err != nil {
			app.Logger().Error("Error getting artist for artwork %s: %v", artPiece.GetString("id"), err)
			return utils.ServerFaultError(c)
		}

		content = append(content, dto.Image{
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId:     artist.GetString("id"),
				ArtistName:   artist.GetString("name"),
				ArtworkTitle: artPiece.GetString("title"),
				ArtworkId:    artPiece.GetString("id"),
			}),
			// Image:     url.GenerateFileUrl("artworks", artworkId, artPiece.Image, ""),
			// Thumb:     url.GenerateThumbUrl("artworks", artworkId, artPiece.Image, "320x240", ""),
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

	return nil
}

// RegisterHandlers registers the HTTP handlers for the PocketBase application.
// It binds a function to the OnServe event, which sets up a GET route for "/inspire".
// When the "/inspire" route is accessed, the inspirationHandler function is called.
//
// Parameters:
//   - app: A pointer to the PocketBase application instance.
func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/inspire", func(c *core.RequestEvent) error {
			return inspirationHandler(app, c)
		})
		return nil
	})
}
