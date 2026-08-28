package postcards

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	postcardworkflow "github.com/blackfyre/wga/internal/postcards"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/utils"
	asseturl "github.com/blackfyre/wga/internal/utils/url"
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
	artwork, err := app.FindFirstRecordByFilter(
		constants.CollectionArtworks,
		"id = {:id} && published = true",
		map[string]any{"id": postcard.GetString("image_id")},
	)
	if err != nil {
		return utils.NotFoundError(c)
	}
	if errs := app.ExpandRecord(artwork, []string{"author"}, nil); len(errs) > 0 {
		logger.Error("Postcard view expansion failed", "event", "postcard.view.failed", "outcome", "expansion_error", "error", logging.Redact(errs))
		return utils.ServerFaultError(c)
	}
	image := utils.AssetUrl("/assets/images/no-image.png")
	if imageName := artwork.GetString("image"); imageName != "" {
		image = asseturl.GenerateArtworkImageURL(artwork, asseturl.DeliveryProfilePostcardSmallDualPlate, "")
	}
	author := artwork.ExpandedOne("author")
	if !hasCompleteArtistIdentity(author) {
		logger.Warn("Postcard view rejected", "event", "postcard.view.rejected", "outcome", "artist_identity_unavailable")
		return utils.NotFoundError(c)
	}
	artistFilingName := author.GetString("filing_name")
	music := resolveRecipientMusic(app, postcard.GetBool("include_music"), artwork)
	content := pages.PostcardView{
		SenderName: postcard.GetString("sender_name"), Message: postcard.GetString("message"), Image: image,
		Title: artwork.GetString("title"), Comment: artwork.GetString("comment"), Technique: artwork.GetString("technique"), ArtistFilingName: artistFilingName,
		Music: music,
	}
	ctx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, "Postcard")
	var buf bytes.Buffer
	if err := pages.PostcardPage(content).Render(ctx, &buf); err != nil {
		return utils.ServerFaultError(c)
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
			return utils.ServerFaultError(c)
		}
		return c.HTML(http.StatusOK, buf.String())
	}

	if err := pages.PostcardLandingPage().Render(ctx, &buf); err != nil {
		return utils.ServerFaultError(c)
	}
	return c.HTML(http.StatusOK, buf.String())
}

// resolveRecipientMusic returns a player-route card only when the postcard
// opted into music and the artwork's author carries a deterministic period-song
// match whose song and composer are both published. It never exposes unmatched
// or unpublished media.
func resolveRecipientMusic(app core.App, includeMusic bool, artwork *core.Record) components.MusicPeriodCard {
	if !includeMusic {
		return components.MusicPeriodCard{}
	}
	author := artwork.ExpandedOne("author")
	if author == nil {
		return components.MusicPeriodCard{}
	}
	song, err := repositories.NewArtistRecordRepository(app).MatchPeriodSong(author.GetInt("year_of_birth"))
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

// hasCompleteArtistIdentity reports whether an artist record carries both
// authoritative identity fields. Prior-bootstrap artists have blank fields and
// must fail closed rather than render reconstructed or blank identity.
func hasCompleteArtistIdentity(artist *core.Record) bool {
	return artist != nil &&
		strings.TrimSpace(artist.GetString("filing_name")) != "" &&
		strings.TrimSpace(artist.GetString("short_name")) != ""
}
