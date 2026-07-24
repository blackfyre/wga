package postcards

import (
	"bytes"
	"cmp"
	"context"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

func sendPostcard(app core.App, c *core.RequestEvent, captcha config.Captcha) error {
	artworkId := cmp.Or(c.Request.URL.Query().Get("awid"), "")

	if artworkId == "" {
		logging.RequestLogger(app, c).Warn("Postcard form request rejected",
			"event", "postcard.form.rejected",
			"outcome", "missing_artwork_id",
		)
		return utils.BadRequestError(c)
	}

	return renderForm(artworkId, app, c, captcha)
}

func renderForm(artworkId string, app core.App, c *core.RequestEvent, captcha config.Captcha) error {
	ctx := context.Background()
	logger := logging.RequestLogger(app, c)

	r, err := app.FindRecordById(constants.CollectionArtworks, artworkId)

	if err != nil {
		logger.Error("Postcard form artwork lookup failed",
			"event", "postcard.form.failed",
			"outcome", "artwork_not_found",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
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
		logger.Error("Postcard form rendering failed",
			"event", "postcard.form.failed",
			"artwork_id", r.Id,
			"outcome", "render_error",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buf.String())
}
