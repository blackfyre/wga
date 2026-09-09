package artists

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// processSelection is the public artist-and-selection route adapter. It
// resolves the artist and selection through their bounded read-models, enforces
// artist ownership and publication, and renders the selection page.
func processSelection(c *core.RequestEvent, app *pocketbase.PocketBase) error {
	return processSelectionWithCheckpoint(c, app, requestprotection.Checkpoint)
}

type selectionDetailCheckpoint func(context.Context, string) error

func processSelectionWithCheckpoint(c *core.RequestEvent, app *pocketbase.PocketBase, checkpoint selectionDetailCheckpoint) error {
	artistSlug := c.Request.PathValue("name")
	selectionID := c.Request.PathValue("selectionID")
	ctx := c.Request.Context()

	artistID := utils.ExtractIdFromString(artistSlug)
	if err := checkpoint(ctx, "selection.detail.artist_lookup"); err != nil {
		return err
	}
	artist, err := repositories.NewArtistRecordRepository(app).FindPublishedArtist(artistID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logging.RequestLogger(app, c).Error("Find published artist", "slug", artistSlug, "error", err)
		}
		return utils.NotFoundError(c)
	}

	expectedSlug := utils.GenerateArtistSlug(artist)
	if artistSlug != expectedSlug {
		return c.Redirect(http.StatusMovedPermanently, buildSelectionURL(expectedSlug, selectionID))
	}

	if err := checkpoint(ctx, "selection.detail.selection_lookup"); err != nil {
		return err
	}
	selection, err := repositories.NewArtistSelectionsRepository(app).FindPublishedSelection(artistID, selectionID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logging.RequestLogger(app, c).Error("Find published selection", "artist", artistID, "selection", selectionID, "error", err)
		}
		return utils.NotFoundError(c)
	}

	view, err := buildSelectionViewContext(ctx, app, artist, selection, checkpoint)
	if err != nil {
		if requestprotection.IsCancellation(err) {
			return err
		}
		logging.RequestLogger(app, c).Error("Build selection view", "error", err)
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	fullURL := buildSelectionURL(expectedSlug, selectionID)

	renderCtx := tmplUtils.DecorateContext(tmplUtils.ContextFromRequest(c.Request), tmplUtils.TitleKey, fmt.Sprintf("%s - %s", view.DisplayTitle, view.ArtistFilingName))
	renderCtx = tmplUtils.DecorateContext(renderCtx, tmplUtils.DescriptionKey, selectionDescription(view))
	renderCtx = tmplUtils.DecorateContext(renderCtx, tmplUtils.CanonicalUrlKey, utils.AssetUrl(fullURL))

	c.Response.Header().Set("HX-Push-Url", fullURL)

	var buff bytes.Buffer
	if err := checkpoint(ctx, "selection.detail.render"); err != nil {
		return err
	}
	if err := pages.SelectionPage(view).Render(renderCtx, &buff); err != nil {
		logging.RequestLogger(app, c).Error("Error rendering selection page", "error", err)
		return utils.ServerFaultError(c, utils.ServerFailure{Category: "server_fault", Cause: err})
	}

	return c.HTML(http.StatusOK, buff.String())
}

// buildSelectionURL is the stable artist-and-selection route identity. The
// artist segment follows the existing slug-id convention while the selection
// segment is the producer's deterministic selection identity; no form or slug
// is derived from the selection title, path, or artwork metadata.
func buildSelectionURL(artistSlug string, selectionID string) string {
	return "/artists/" + artistSlug + "/selections/" + selectionID
}

// buildSelectionView assembles the page-owned selection read model from the
// bounded selection repository. Commentary is sanitised through the record's
// trusted-HTML boundary before it reaches templ.Raw.
func buildSelectionView(app *pocketbase.PocketBase, artist *core.Record, selection *core.Record) (pages.SelectionView, error) {
	return buildSelectionViewContext(context.Background(), app, artist, selection, requestprotection.Checkpoint)
}

func buildSelectionViewContext(ctx context.Context, app *pocketbase.PocketBase, artist *core.Record, selection *core.Record, checkpoint selectionDetailCheckpoint) (pages.SelectionView, error) {
	repo := repositories.NewArtistSelectionsRepository(app)

	if err := checkpoint(ctx, "selection.detail.works"); err != nil {
		return pages.SelectionView{}, err
	}
	works, err := repo.ListSelectionArtworks(artist.Id, selection)
	if err != nil {
		return pages.SelectionView{}, err
	}

	if err := checkpoint(ctx, "selection.detail.related_content"); err != nil {
		return pages.SelectionView{}, err
	}
	all, err := repo.ListPublishedSelections(artist.Id, 0)
	if err != nil {
		return pages.SelectionView{}, err
	}
	if err := checkpoint(ctx, "selection.detail.projection"); err != nil {
		return pages.SelectionView{}, err
	}

	otherSelections := make([]pages.SelectionLink, 0, len(all))
	for _, other := range all {
		if other.Id == selection.Id {
			continue
		}
		otherSelections = append(otherSelections, pages.SelectionLink{
			URL:   buildSelectionURL(utils.GenerateArtistSlug(artist), other.Id),
			Title: other.GetString("display_title"),
		})
	}

	commentary := sanitizeSelectionCommentary(selection.GetString("commentary"))

	return pages.SelectionView{
		ArtistFilingName: artist.GetString("filing_name"),
		ArtistShortName:  artist.GetString("short_name"),
		ArtistURL:        "/artists/" + utils.GenerateArtistSlug(artist),
		DisplayTitle:     selection.GetString("display_title"),
		Context:          selection.GetString("context"),
		Commentary:       commentary,
		HasCommentary:    commentary != "",
		Works:            buildRecordWorkImages(artist, works),
		WorkCount:        len(works),
		OtherSelections:  otherSelections,
		HoldingURL:       buildArtistNameWorksURL(artist.GetString("filing_name")),
		Url:              buildSelectionURL(utils.GenerateArtistSlug(artist), selection.Id),
		Citation:         buildSelectionCitation(artist, selection),
		ShowBreadcrumbs:  true,
	}, nil
}

// buildSelectionCitation builds the canonical BibTeX citation for a selection.
// The entry key is derived from the producer's deterministic selection identity
// and the title from the persisted display title, so the citation never infers
// editorial prose or identity beyond the source-backed record.
func buildSelectionCitation(artist *core.Record, selection *core.Record) components.Citation {
	return components.Citation{
		Key:   "wga-" + selection.Id,
		Title: selection.GetString("display_title") + " (selection)",
		URL:   utils.AssetUrl(buildSelectionURL(utils.GenerateArtistSlug(artist), selection.Id)),
	}
}

// sanitizeSelectionCommentary applies the record-owned biography sanitizer to
// external selection commentary so unsafe markup never reaches templ.Raw.
func sanitizeSelectionCommentary(commentary string) string {
	return bioSanitizer.Sanitize(commentary)
}

// selectionDescription returns a plain-text description for the page head,
// preferring the sanitised commentary and never rendering raw HTML there.
func selectionDescription(view pages.SelectionView) string {
	if view.Commentary != "" {
		return tmplUtils.StripHtmlTags(view.Commentary)
	}

	return fmt.Sprintf("Curated selection of works by %s", view.ArtistShortName)
}
