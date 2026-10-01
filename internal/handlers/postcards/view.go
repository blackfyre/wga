package postcards

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func viewPostcard(app core.App, c *core.RequestEvent) error {
	logger := logging.RequestLogger(app, c)
	query := c.Request.URL.Query()
	if _, explicit := query["token"]; !explicit {
		return viewPostcardLanding(c)
	}
	// A present token means this is a recipient request regardless of whether the
	// lookup later succeeds, so confidentiality headers must apply before any
	// lookup to cover 404 outcomes too.
	c.Response.Header().Set("Cache-Control", "no-store")
	c.Response.Header().Set("Referrer-Policy", "no-referrer")
	token := query.Get("token")
	view, err := postcardworkflow.FindRecipientView(app, token, types.NowDateTime())
	if err != nil {
		logger.Warn("Postcard view rejected", "event", "postcard.view.rejected", "outcome", "invalid_expired_or_unknown_token")
		return utils.NotFoundError(c)
	}
	postcard := view.Postcard
	card, err := postcardworkflow.ResolveReceivedCard(app, postcard)
	switch {
	case errors.Is(err, postcardworkflow.ErrArtworkUnavailable):
		logger.Warn("Postcard view rejected", "event", "postcard.view.rejected", "outcome", "artwork_unavailable")
		return utils.NotFoundError(c)
	case errors.Is(err, postcardworkflow.ErrArtistIdentityUnavailable):
		logger.Warn("Postcard view rejected", "event", "postcard.view.rejected", "outcome", "artist_identity_unavailable")
		return utils.NotFoundError(c)
	case err != nil:
		logger.Error("Postcard view lookup failed", "event", "postcard.view.failed", "outcome", "lookup_error", "error", logging.Redact(err))
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	message := postcardworkflow.SanitiseMessage(postcard.GetString("message"))
	content := pages.PostcardView{
		SenderName: postcard.GetString("sender_name"), Message: message.HTML(), Image: card.Image,
		Title: card.Title, Comment: card.Comment, Technique: card.Technique, ArtistFilingName: card.ArtistFilingName,
		Dimensions: card.Dimensions, Location: card.Location, RecordURL: card.RecordURL, ComposeURL: card.ComposeURL,
		Music: resolveRecipientMusic(app, card.Artwork),
	}
	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Postcard")
	var buf bytes.Buffer
	if err := pages.PostcardPage(content).Render(ctx, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	if err := postcardworkflow.MarkReceived(app, postcard.Id); err != nil {
		logger.Error("Postcard receipt update failed", "event", "postcard.view.failed", "outcome", "receipt_update_error", "error_type", logging.ErrorType(err), "error", logging.Redact(err))
	}
	return c.HTML(http.StatusOK, buf.String())
}

// viewPostcardLanding renders the tokenless public postcard landing. It does not
// touch recipient state or bearer material, so it applies no no-store or
// no-referrer headers.
func viewPostcardLanding(c *core.RequestEvent) error {
	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Postcards")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Postcards are composed from published artwork pages. Choose a work to send it as a private link.")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.CanonicalUrlKey, tmplUtils.AssetUrl("/postcard"))
	c.Response.Header().Set("HX-Push-Url", "/postcard")

	var buf bytes.Buffer
	if utils.IsHtmxRequest(c) && !utils.RequestsMainContentArea(c) {
		err := pages.PostcardLandingContent().Render(ctx, &buf)
		if err != nil {
			return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
		}
		return c.HTML(http.StatusOK, buf.String())
	}

	if err := pages.PostcardLandingPage().Render(ctx, &buf); err != nil {
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}
	return c.HTML(http.StatusOK, buf.String())
}

// resolveRecipientMusic derives the ordinary player-route card from the
// published artwork. It never consults sender input or exposes unpublished media.
func resolveRecipientMusic(app core.App, artwork *core.Record) components.MusicPeriodCard {
	song, err := repositories.NewArtistRecordRepository(app).MatchPeriodSong(artwork.GetInt("date_start"))
	if err != nil || song == nil {
		return components.MusicPeriodCard{}
	}
	return buildPostcardMusic(song)
}

// buildPostcardMusic maps a deterministic period-song match onto the validated
// player-route card, or returns an empty card when no complete match exists.
func buildPostcardMusic(song *repositories.PeriodSong) components.MusicPeriodCard {
	if song == nil || song.Record == nil {
		return components.MusicPeriodCard{}
	}

	piece := strings.TrimSpace(song.Record.GetString("title"))
	if piece == "" || song.Record.GetString("source") == "" {
		return components.MusicPeriodCard{}
	}

	return components.MusicPeriodCard{
		SongID:    song.Record.Id,
		Piece:     piece,
		PlayerURL: "/player?song=" + song.Record.Id,
	}
}
