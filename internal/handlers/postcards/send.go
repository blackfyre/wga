package postcards

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func sendPostcard(app *pocketbase.PocketBase, c *core.RequestEvent, captcha config.Captcha) error {

	artworkId := cmp.Or(c.Request.URL.Query().Get("awid"), "")

	if artworkId == "" {
		app.Logger().Error("No artwork id provided for postcard", "ip", c.RealIP())
		return utils.BadRequestError(c)
	}

	return renderForm(artworkId, app, c, captcha)
}

func renderForm(artworkId string, app *pocketbase.PocketBase, c *core.RequestEvent, captcha config.Captcha) error {
	ctx := context.Background()

	r, err := app.FindRecordById(constants.CollectionArtworks, artworkId)

	if err != nil {
		app.Logger().Error("Failed to find artwork "+artworkId, "error", err.Error())
		return utils.NotFoundError(c)
	}

	var buf bytes.Buffer
	var editor components.PostcardEditorDTO

	editor.ImageId = artworkId
	if r.GetString("image") == "" {
		editor.Image = utils.AssetUrl("/assets/images/no-image.png")
	} else {
		editor.Image = url.GenerateFileUrl(constants.CollectionArtworks, artworkId, r.GetString("image"), "")
	}
	editor.Title = r.GetString("title")
	editor.Comment = r.GetString("comment")
	editor.Technique = r.GetString("technique")
	editor.SiteKey = captcha.SiteKey()

	err = components.PostcardEditor(editor).Render(ctx, &buf)

	if err != nil {
		app.Logger().Error(fmt.Sprintf("Failed to render the postcard editor with image_id %s", artworkId), "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buf.String())
}
