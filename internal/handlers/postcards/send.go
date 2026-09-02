package postcards

import (
	"bytes"
	"cmp"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/utils"
	asseturl "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

func sendPostcard(app core.App, c *core.RequestEvent, captcha config.Captcha) error {
	artworkID := cmp.Or(c.Request.URL.Query().Get("awid"), "")
	if artworkID == "" {
		logging.RequestLogger(app, c).Warn("Postcard form request rejected", "event", "postcard.form.rejected", "outcome", "missing_artwork_id")
		return utils.BadRequestError(c)
	}
	submissionKey, err := postcardworkflow.NewSubmissionKey()
	if err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return renderForm(artworkID, pages.PostcardComposeView{SubmissionKey: submissionKey}, "", http.StatusOK, app, c, captcha)
}

func renderForm(artworkID string, values pages.PostcardComposeView, formError string, status int, app core.App, c *core.RequestEvent, captcha config.Captcha) error {
	logger := logging.RequestLogger(app, c)
	record, err := app.FindFirstRecordByFilter(constants.CollectionArtworks, "id = {:id} && published = true", map[string]any{"id": artworkID})
	if err != nil {
		logger.Warn("Postcard form artwork unavailable", "event", "postcard.form.rejected", "outcome", "artwork_unavailable")
		return utils.NotFoundError(c)
	}
	if errs := app.ExpandRecord(record, []string{"author"}, nil); len(errs) > 0 {
		logger.Error("Postcard form artwork expansion failed", "event", "postcard.form.failed", "outcome", "expansion_error", "error", logging.Redact(errs))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	values.ImageID = artworkID
	values.Title = record.GetString("title")
	values.Technique = record.GetString("technique")
	values.SiteKey = captcha.SiteKey()
	values.Error = formError
	author := record.ExpandedOne("author")
	if !hasCompleteArtistIdentity(author) {
		logger.Warn("Postcard form artwork author unavailable", "event", "postcard.form.rejected", "outcome", "artist_identity_unavailable")
		return utils.NotFoundError(c)
	}
	values.ArtistFilingName = author.GetString("filing_name")
	values.MusicAvailable = resolveRecipientMusic(app, true, record).SongID != ""
	if image := record.GetString("image"); image != "" {
		values.Image = asseturl.GenerateArtworkImageURL(record, asseturl.DeliveryProfilePostcardSmallDualPlate, "")
	} else {
		values.Image = utils.AssetUrl("/assets/images/no-image.png")
	}

	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Send a postcard")
	var buf bytes.Buffer
	c.Response.Header().Add("Vary", "HX-Request")
	if utils.IsHtmxRequest(c) {
		err = pages.PostcardComposeBlock(values).Render(ctx, &buf)
	} else {
		err = pages.PostcardComposePage(values).Render(ctx, &buf)
	}
	if err != nil {
		logger.Error("Postcard form rendering failed", "event", "postcard.form.failed", "outcome", "render_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return c.HTML(status, buf.String())
}
