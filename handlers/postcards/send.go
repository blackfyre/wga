package postcards

import (
	"context"
	"fmt"

	"github.com/blackfyre/wga/assets/templ/components"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

func sendPostcard(app *pocketbase.PocketBase, c echo.Context) error {

	artworkId, err := url.GetRequiredQueryParam(c, "awid")

	if err != nil {
		app.Logger().Error("Failed to get required query param", "error", err.Error())
		return utils.BadRequestError(c)
	}

	return renderForm(artworkId, app, c)
}

func renderForm(artworkId string, app *pocketbase.PocketBase, c echo.Context) error {
	ctx := context.Background()

	r, err := app.FindRecordById("artworks", artworkId)

	if err != nil {
		app.Logger().Error("Failed to find artwork "+artworkId, "error", err.Error())
		return utils.NotFoundError(c)
	}

	err = components.PostcardEditor(components.PostcardEditorDTO{
		Image:     url.GenerateFileUrl("artworks", artworkId, r.GetString("image"), ""),
		ImageId:   artworkId,
		Title:     r.GetString("title"),
		Comment:   r.GetString("comment"),
		Technique: r.GetString("technique"),
	}).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error(fmt.Sprintf("Failed to render the postcard editor with image_id %s", artworkId), "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil
}
