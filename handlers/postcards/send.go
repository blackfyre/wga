package postcards

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"

	"github.com/blackfyre/wga/assets/templ/components"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func sendPostcard(app *pocketbase.PocketBase, c *core.RequestEvent) error {

	artworkId := cmp.Or(c.Request.URL.Query().Get("awid"), "")

	if artworkId == "" {
		app.Logger().Error("No artwork id provided for postcard", "ip", c.RealIP())
		return utils.BadRequestError(c)
	}

	return renderForm(artworkId, app, c)
}

func renderForm(artworkId string, app *pocketbase.PocketBase, c *core.RequestEvent) error {
	ctx := context.Background()

	r, err := app.FindRecordById("artworks", artworkId)

	if err != nil {
		app.Logger().Error("Failed to find artwork "+artworkId, "error", err.Error())
		return utils.NotFoundError(c)
	}

	var buf bytes.Buffer

	err = components.PostcardEditor(components.PostcardEditorDTO{
		Image:     url.GenerateFileUrl("artworks", artworkId, r.GetString("image"), ""),
		ImageId:   artworkId,
		Title:     r.GetString("title"),
		Comment:   r.GetString("comment"),
		Technique: r.GetString("technique"),
	}).Render(ctx, &buf)

	if err != nil {
		app.Logger().Error(fmt.Sprintf("Failed to render the postcard editor with image_id %s", artworkId), "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buf.String())
}
