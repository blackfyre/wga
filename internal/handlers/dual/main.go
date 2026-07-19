package dual

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/handlers/artists"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type renderPaneDto struct {
	Side     string
	Content  string
	RelPath  string
	RenderTo string
}

type panePathDto struct {
	Kind    string
	Id      string
	RelPath string
}

const (
	dualLookupPath              = "/dual-mode/lookup"
	dualLookupLimit             = 20
	dualLookupMinimumQueryRunes = 2
)

// renderDualModePage renders the dual-mode comparison view.
func renderDualModePage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Dual View")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "Compare artists and artworks side by side.")
	// ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, pHtmxUrl)

	contentDto := dto.DualViewDto{}

	leftPane, err := renderPane("left", app, c)
	if err != nil {
		app.Logger().Error("Error rendering left pane", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	rightPane, err := renderPane("right", app, c)
	if err != nil {
		app.Logger().Error("Error rendering right pane", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	contentDto.Left = leftPane.Content
	contentDto.Right = rightPane.Content
	contentDto.LeftLinksOpenInOtherPane = leftPane.RenderTo == reverseSide(leftPane.Side)
	contentDto.RightLinksOpenInOtherPane = rightPane.RenderTo == reverseSide(rightPane.Side)
	contentDto.ArtworkSearchLeftUrl = buildDualArtworkSearchURL(leftPane, rightPane, "left")
	contentDto.ArtworkSearchRightUrl = buildDualArtworkSearchURL(leftPane, rightPane, "right")
	contentDto.CopyLeftToRightUrl = buildDualModeActionURL(leftPane, rightPane, "copy-left-to-right")
	contentDto.CopyRightToLeftUrl = buildDualModeActionURL(leftPane, rightPane, "copy-right-to-left")
	contentDto.ReverseUrl = buildDualModeActionURL(leftPane, rightPane, "reverse")
	contentDto.ClearLeftUrl = buildDualModeActionURL(leftPane, rightPane, "clear-left")
	contentDto.ClearRightUrl = buildDualModeActionURL(leftPane, rightPane, "clear-right")
	c.Response.Header().Set("HX-Push-Url", buildDualModePushURL(leftPane, rightPane))

	var buff bytes.Buffer

	err = pages.DualPage(
		contentDto,
		buildDualPaneLoadForms(leftPane, rightPane),
		buildDualPaneTargetURLs(leftPane, rightPane),
	).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering dual mode page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(200, buff.String())
}

func renderDualLookupResults(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	queryValues := c.Request.URL.Query()
	content, err := getDualLookupResults(app, queryValues.Get("kind"), queryValues.Get("q"))

	if err != nil {
		app.Logger().Error("Error getting dual lookup results", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	var buff bytes.Buffer

	if err := pages.DualLookupResultContent(content).Render(context.Background(), &buff); err != nil {
		app.Logger().Error("Error rendering dual lookup results", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(http.StatusOK, buff.String())
}

func getDualLookupResults(app core.App, kind string, query string) (dto.DualLookupResultsDto, error) {
	content := dto.DualLookupResultsDto{
		Kind:    resolveDualLookupKind(kind),
		Query:   strings.TrimSpace(query),
		Results: []dto.DualLookupResultDto{},
	}

	if content.Query == "" {
		return content, nil
	}

	if utf8.RuneCountInString(content.Query) < dualLookupMinimumQueryRunes {
		content.QueryTooShort = true
		return content, nil
	}

	var err error

	switch content.Kind {
	case "artist":
		content.Results, err = getArtistLookupResults(app, content.Query)
	case "artwork":
		content.Results, err = getArtworkLookupResults(app, content.Query)
	}

	return content, err
}

func resolveDualLookupKind(kind string) string {
	if strings.TrimSpace(kind) == "artwork" {
		return "artwork"
	}

	return "artist"
}

func getArtistLookupResults(app core.App, query string) ([]dto.DualLookupResultDto, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		"published = true && name ~ {:query}",
		"+name,+id",
		dualLookupLimit,
		0,
		dbx.Params{"query": query},
	)

	if err != nil {
		return nil, err
	}

	results := make([]dto.DualLookupResultDto, 0, len(records))

	for _, record := range records {
		results = append(results, dto.DualLookupResultDto{
			Url: url.GenerateArtistUrl(url.ArtistUrlDTO{
				ArtistId:   record.Id,
				ArtistName: record.GetString("name"),
			}),
			Label: record.GetString("name"),
		})
	}

	return results, nil
}

func getArtworkLookupResults(app core.App, query string) ([]dto.DualLookupResultDto, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollectionArtworks,
		"published = true && author:length > 0 && title ~ {:query}",
		"+title,+id",
		dualLookupLimit,
		0,
		dbx.Params{"query": query},
	)

	if err != nil {
		return nil, err
	}

	artistsByID, err := getLookupArtistsByID(app, uniqueLookupArtistIDs(records))
	if err != nil {
		return nil, err
	}

	results := make([]dto.DualLookupResultDto, 0, len(records))

	for _, record := range records {
		authorIDs := record.GetStringSlice("author")
		if len(authorIDs) == 0 {
			continue
		}

		artist, ok := artistsByID[authorIDs[0]]
		if !ok {
			continue
		}

		results = append(results, dto.DualLookupResultDto{
			Url: url.GenerateFullArtworkUrl(url.ArtworkUrlDTO{
				ArtistId:     artist.Id,
				ArtistName:   artist.GetString("name"),
				ArtworkId:    record.Id,
				ArtworkTitle: record.GetString("title"),
			}),
			Label:   record.GetString("title"),
			Context: artist.GetString("name"),
		})
	}

	return results, nil
}

func uniqueLookupArtistIDs(records []*core.Record) []string {
	seen := map[string]struct{}{}
	artistIDs := make([]string, 0, len(records))

	for _, record := range records {
		for _, artistID := range record.GetStringSlice("author") {
			if artistID == "" {
				continue
			}

			if _, exists := seen[artistID]; exists {
				continue
			}

			seen[artistID] = struct{}{}
			artistIDs = append(artistIDs, artistID)
		}
	}

	return artistIDs
}

func getLookupArtistsByID(app core.App, artistIDs []string) (map[string]*core.Record, error) {
	if len(artistIDs) == 0 {
		return map[string]*core.Record{}, nil
	}

	params := dbx.Params{}
	conditions := make([]string, 0, len(artistIDs))

	for index, artistID := range artistIDs {
		key := fmt.Sprintf("artist_id_%d", index)
		conditions = append(conditions, fmt.Sprintf("id = {:%s}", key))
		params[key] = artistID
	}

	records, err := app.FindRecordsByFilter(
		constants.CollectionArtists,
		strings.Join(conditions, " || "),
		"+name,+id",
		len(artistIDs),
		0,
		params,
	)

	if err != nil {
		return nil, err
	}

	artistsByID := make(map[string]*core.Record, len(records))
	for _, record := range records {
		artistsByID[record.Id] = record
	}

	return artistsByID, nil
}

func buildDualModePushURL(leftPane renderPaneDto, rightPane renderPaneDto) string {
	return buildDualModeURL(
		leftPane.RelPath,
		rightPane.RelPath,
		leftPane.RenderTo,
		rightPane.RenderTo,
	)
}

func buildDualModeURL(leftRelPath string, rightRelPath string, leftRenderTo string, rightRenderTo string) string {
	relPath := url.GenerateDualModeUrl()
	queryValues := relPath.Query()

	queryValues.Set("left", cmp.Or(strings.TrimSpace(leftRelPath), "default"))
	queryValues.Set("right", cmp.Or(strings.TrimSpace(rightRelPath), "default"))
	queryValues.Set("left_render_to", resolvePaneTarget("left", leftRenderTo))
	queryValues.Set("right_render_to", resolvePaneTarget("right", rightRenderTo))

	relPath.RawQuery = queryValues.Encode()

	return relPath.String()
}

func buildDualModeActionURL(leftPane renderPaneDto, rightPane renderPaneDto, action string) string {
	leftPath := leftPane.RelPath
	rightPath := rightPane.RelPath

	switch action {
	case "copy-left-to-right":
		rightPath = leftPath
	case "copy-right-to-left":
		leftPath = rightPath
	case "reverse":
		leftPath, rightPath = rightPath, leftPath
	case "clear-left":
		leftPath = "default"
	case "clear-right":
		rightPath = "default"
	}

	return buildDualModeURL(leftPath, rightPath, leftPane.RenderTo, rightPane.RenderTo)
}

func buildDualArtworkSearchURL(leftPane renderPaneDto, rightPane renderPaneDto, target string) string {
	searchURL := url.GenerateDualModeUrl()
	searchURL.Path = "/artworks"
	queryValues := searchURL.Query()
	queryValues.Set("dual_left", leftPane.RelPath)
	queryValues.Set("dual_right", rightPane.RelPath)
	queryValues.Set("dual_left_render_to", leftPane.RenderTo)
	queryValues.Set("dual_right_render_to", rightPane.RenderTo)
	queryValues.Set("dual_target", target)
	searchURL.RawQuery = queryValues.Encode()

	return searchURL.String()
}

func buildDualPaneLoadForms(leftPane renderPaneDto, rightPane renderPaneDto) dto.DualPaneLoadFormsDto {
	return dto.DualPaneLoadFormsDto{
		Left: dto.DualPaneLoadFormDto{
			Path:          panePathInputValue(leftPane.RelPath),
			OtherPath:     rightPane.RelPath,
			LeftRenderTo:  leftPane.RenderTo,
			RightRenderTo: rightPane.RenderTo,
		},
		Right: dto.DualPaneLoadFormDto{
			Path:          panePathInputValue(rightPane.RelPath),
			OtherPath:     leftPane.RelPath,
			LeftRenderTo:  leftPane.RenderTo,
			RightRenderTo: rightPane.RenderTo,
		},
	}
}

func buildDualPaneTargetURLs(leftPane renderPaneDto, rightPane renderPaneDto) dto.DualPaneTargetUrlsDto {
	return dto.DualPaneTargetUrlsDto{
		LeftSamePaneUrl:   buildDualModeURL(leftPane.RelPath, rightPane.RelPath, "left", rightPane.RenderTo),
		LeftOtherPaneUrl:  buildDualModeURL(leftPane.RelPath, rightPane.RelPath, "right", rightPane.RenderTo),
		RightSamePaneUrl:  buildDualModeURL(leftPane.RelPath, rightPane.RelPath, leftPane.RenderTo, "right"),
		RightOtherPaneUrl: buildDualModeURL(leftPane.RelPath, rightPane.RelPath, leftPane.RenderTo, "left"),
	}
}

func panePathInputValue(relPath string) string {
	if relPath == "default" {
		return ""
	}

	return relPath
}

func buildDualModePaneURL(side string, currentRelPath string, destinationRelPath string, queryValues map[string][]string) string {
	leftRelPath := cmp.Or(strings.TrimSpace(firstQueryValue(queryValues, "left")), "default")
	rightRelPath := cmp.Or(strings.TrimSpace(firstQueryValue(queryValues, "right")), "default")
	leftRenderTo := resolvePaneTarget("left", firstQueryValue(queryValues, "left_render_to"))
	rightRenderTo := resolvePaneTarget("right", firstQueryValue(queryValues, "right_render_to"))
	targetSide := resolvePaneTarget(side, firstQueryValue(queryValues, side+"_render_to"))

	switch side {
	case "left":
		leftRelPath = currentRelPath
	case "right":
		rightRelPath = currentRelPath
	}

	switch targetSide {
	case "left":
		leftRelPath = destinationRelPath
	case "right":
		rightRelPath = destinationRelPath
	}

	return buildDualModeURL(leftRelPath, rightRelPath, leftRenderTo, rightRenderTo)
}

func firstQueryValue(queryValues map[string][]string, key string) string {
	if len(queryValues[key]) == 0 {
		return ""
	}

	return queryValues[key][0]
}

func resolvePaneTarget(side string, requestedTarget string) string {
	requestedTarget = strings.TrimSpace(requestedTarget)
	if requestedTarget == side || requestedTarget == reverseSide(side) {
		return requestedTarget
	}

	return reverseSide(side)
}

func parsePanePath(raw string) (panePathDto, error) {
	normalized := cmp.Or(strings.TrimSpace(raw), "default")

	if normalized == "default" {
		return panePathDto{Kind: "default", RelPath: "default"}, nil
	}

	normalized = "/" + strings.Trim(normalized, "/")
	parts := strings.Split(strings.Trim(normalized, "/"), "/")

	switch {
	case len(parts) == 2 && parts[0] == "artists":
		return panePathDto{
			Kind:    "artist",
			Id:      utils.ExtractIdFromString(parts[1]),
			RelPath: normalized,
		}, nil
	case len(parts) == 3 && parts[0] == "artists":
		return panePathDto{
			Kind:    "artwork",
			Id:      utils.ExtractIdFromString(parts[2]),
			RelPath: normalized,
		}, nil
	case len(parts) == 4 && parts[0] == "artists" && parts[2] == "artworks":
		return panePathDto{
			Kind:    "artwork",
			Id:      utils.ExtractIdFromString(parts[3]),
			RelPath: normalized,
		}, nil
	case len(parts) == 2 && parts[0] == "artworks":
		return panePathDto{
			Kind:    "artwork",
			Id:      utils.ExtractIdFromString(parts[1]),
			RelPath: normalized,
		}, nil
	default:
		return panePathDto{}, errs.ErrUnknownDualPane
	}
}

// reverseSide returns the opposite side of the given side.
func reverseSide(side string) string {
	switch side {
	case "left":
		return "right"
	case "right":
		return "left"
	default:
		return ""
	}
}

func renderPane(side string, app *pocketbase.PocketBase, c *core.RequestEvent) (renderPaneDto, error) {
	rawQParams := c.Request.URL.Query()

	queryParam := cmp.Or(rawQParams.Get(side), "default")
	renderTo := resolvePaneTarget(side, rawQParams.Get(side+"_render_to"))

	pane := renderPaneDto{
		Side:     side,
		RenderTo: renderTo,
	}
	buf := new(bytes.Buffer)

	parsedPath, err := parsePanePath(queryParam)
	if err != nil {
		return renderDefaultPane(side, renderTo)
	}

	if parsedPath.Kind == "default" {
		return renderDefaultPane(side, renderTo)
	}

	switch parsedPath.Kind {
	case "artist":
		artistDto, renderErr := renderArtistPane(app, c, side, parsedPath.RelPath, parsedPath.Id, buf)
		if renderErr != nil {
			if errors.Is(renderErr, sql.ErrNoRows) {
				return renderDefaultPane(side, renderTo)
			}

			return pane, renderErr
		}

		pane.RelPath = resolvePaneRelPath(parsedPath.RelPath, artistDto.Url)
	case "artwork":
		artworkDto, renderErr := renderArtworkPane(app, c, side, parsedPath.RelPath, parsedPath.Id, buf)
		if renderErr != nil {
			if errors.Is(renderErr, sql.ErrNoRows) {
				return renderDefaultPane(side, renderTo)
			}

			return pane, renderErr
		}

		pane.RelPath = resolvePaneRelPath(parsedPath.RelPath, artworkDto.Url)
	default:
		return pane, errs.ErrUnsupportedPaneType
	}

	pane.Content = buf.String()

	return pane, nil
}

func renderDefaultPane(side string, renderTo string) (renderPaneDto, error) {
	defaultContent, err := defaultPaneContent(side)
	if err != nil {
		return renderPaneDto{}, err
	}

	return renderPaneDto{
		Side:     side,
		Content:  defaultContent,
		RelPath:  "default",
		RenderTo: renderTo,
	}, nil
}

func resolvePaneRelPath(requestedRelPath string, renderedRelPath string) string {
	if strings.TrimSpace(renderedRelPath) == "" {
		return requestedRelPath
	}

	if strings.HasPrefix(renderedRelPath, "/dual-mode") {
		return requestedRelPath
	}

	return renderedRelPath
}

func defaultPaneContent(side string) (string, error) {
	if side != "left" && side != "right" {
		return "", errs.ErrUnsupportedPaneType
	}

	buf := new(bytes.Buffer)
	if err := pages.DualPaneDefault(side).Render(context.Background(), buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func renderArtistPane(app *pocketbase.PocketBase, c *core.RequestEvent, side string, currentRelPath string, artistId string, buf *bytes.Buffer) (dto.Artist, error) {
	artistModel, err := app.FindRecordById(constants.CollectionArtists, artistId)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.Logger().Error("Error finding artist", "error", err.Error())
		}

		return dto.Artist{}, err
	}

	artistDto, err := artists.RenderArtistContent(app, c, artistModel, "#dual-area", false)

	if err != nil {
		app.Logger().Error("Error rendering artist content", "error", err.Error())
		return dto.Artist{}, err
	}

	if c.Request != nil && c.Request.URL != nil && c.Request.URL.Path == "/dual-mode" {
		currentQueryValues := c.Request.URL.Query()
		for idx := range artistDto.Works {
			artistDto.Works[idx].Url = buildDualModePaneURL(side, currentRelPath, artistDto.Works[idx].Url, currentQueryValues)
		}
	}

	err = pages.ArtistBlock(artistDto).Render(context.Background(), buf)

	if err != nil {
		app.Logger().Error("Error rendering artist page", "error", err.Error())
		return artistDto, err
	}

	return artistDto, nil
}

func renderArtworkPane(app *pocketbase.PocketBase, c *core.RequestEvent, side string, currentRelPath string, artworkId string, buf *bytes.Buffer) (dto.Artwork, error) {
	artworkModel, err := app.FindRecordById(constants.CollectionArtworks, artworkId)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.Logger().Error("Error finding artwork", "error", err.Error())
		}

		return dto.Artwork{}, err
	}

	artworkDto, err := artists.RenderArtworkContent(app, c, artworkModel, "#dual-area", false)

	if err != nil {
		app.Logger().Error("Error rendering artwork content", "error", err.Error())
		return dto.Artwork{}, err
	}

	if c.Request != nil && c.Request.URL != nil && c.Request.URL.Path == "/dual-mode" {
		currentQueryValues := c.Request.URL.Query()
		artworkDto.Url = buildDualModePaneURL(side, currentRelPath, artworkDto.Url, currentQueryValues)
		if artworkDto.Artist.Url != "" {
			artworkDto.Artist.Url = buildDualModePaneURL(side, currentRelPath, artworkDto.Artist.Url, currentQueryValues)
		}
	}

	err = pages.ArtworkBlock(artworkDto).Render(context.Background(), buf)

	if err != nil {
		app.Logger().Error("Error rendering artwork page", "error", err.Error())
		return artworkDto, err
	}

	return artworkDto, nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/dual-mode", func(c *core.RequestEvent) error {
			return renderDualModePage(app, c)
		})
		se.Router.GET(dualLookupPath, func(c *core.RequestEvent) error {
			return renderDualLookupResults(app, c)
		})
		return se.Next()
	})
}
