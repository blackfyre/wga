package postcards

import (
	"bytes"
	"cmp"
	"context"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

func viewPostcard(app core.App, c *core.RequestEvent) error {
	postCardId := cmp.Or(c.Request.URL.Query().Get("p"), "nope")
	logger := logging.RequestLogger(app, c)

	if postCardId == "nope" {
		logger.Warn("Postcard view rejected",
			"event", "postcard.view.rejected",
			"outcome", "missing_postcard_id",
		)
		return utils.NotFoundError(c)
	}

	r, err := app.FindRecordById(constants.CollectionPostcards, postCardId)

	if err != nil {
		logger.Error("Postcard view lookup failed",
			"event", "postcard.view.failed",
			"outcome", "postcard_not_found",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return utils.NotFoundError(c)
	}

	if errs := app.ExpandRecord(r, []string{"image_id"}, nil); len(errs) > 0 {
		logger.Error("Postcard view expansion failed",
			"event", "postcard.view.failed",
			"outcome", "expansion_error",
			"error", logging.Redact(errs),
		)
		return utils.ServerFaultError(c)
	}

	aw := r.ExpandedOne("image_id")

	content := pages.PostcardView{
		SenderName: r.GetString("sender_name"),
		Message:    r.GetString("message"),
		Image:      url.GenerateFileUrl(constants.CollectionArtworks, aw.GetString("id"), aw.GetString("image"), ""),
		Title:      aw.GetString("title"),
		Comment:    aw.GetString("comment"),
		Technique:  aw.GetString("technique"),
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Postcard")

	// c.Response.Header().Set("HX-Push-Url", fullUrl)

	var buf bytes.Buffer

	err = pages.PostcardPage(content).Render(ctx, &buf)

	if err != nil {
		logger.Error("Postcard view rendering failed",
			"event", "postcard.view.failed",
			"outcome", "render_error",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return utils.ServerFaultError(c)
	}
	if err := postcardworkflow.MarkReceived(app, r.Id); err != nil {
		logger.Error("Postcard receipt update failed",
			"event", "postcard.view.failed",
			"outcome", "receipt_update_error",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buf.String())
}
