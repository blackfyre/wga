package inspire

import (
	"context"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	"github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
)

func inspirationHandler(app *pocketbase.PocketBase, c echo.Context) error {

	items, err := models.GetRandomArtworks(app.Dao(), 20)

	if err != nil {
		app.Logger().Error("Error getting random artworks: %v", err)
		return utils.ServerFaultError(c)
	}

	content := dto.ImageGrid{}

	for _, item := range items {

		artworkId := item.GetId()

		artist, err := models.GetArtistById(app.Dao(), item.Author)

		if err != nil {
			app.Logger().Error("Error getting artist for artwork %s: %v", item.GetId(), err)
			return utils.ServerFaultError(c)
		}

		content = append(content, dto.Image{
			Url:       url.GenerateArtworkUrl(url.ArtworkUrlDTO{
				ArtistId: artist.Id,
				ArtistName: artist.Name,
				ArtworkTitle: item.Author,
				ArtworkId: item.Id,
			}),
			Image:     url.GenerateFileUrl("artworks", artworkId, item.Image, ""),
			Thumb:     url.GenerateThumbUrl("artworks", artworkId, item.Image, "320x240", ""),
			Comment:   item.Comment,
			Title:     item.Title,
			Technique: item.Technique,
			Id:        artworkId,
			Artist: dto.Artist{
				Id:         artist.Id,
				Name:       artist.Name,
				Url:        url.GenerateArtistUrl(url.ArtistUrlDTO{
					ArtistId: artist.Id,
					ArtistName: artist.Name,
				}),
				Profession: artist.Profession,
			},
		})
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Inspiration")

	c.Response().Header().Set("HX-Push-Url", "/inspire")
	err = pages.InspirePage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		return utils.ServerFaultError(c)
	}

	return nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/inspire", func(c echo.Context) error {
			return inspirationHandler(app, c)
		})
		return nil
	})
}
