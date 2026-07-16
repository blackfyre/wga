package dual

import (
	"bytes"
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/errs"
	"github.com/blackfyre/wga/internal/handlers/artists"
	"github.com/blackfyre/wga/internal/handlers/artworks"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type renderPaneDto struct {
	Side    string
	Content string
	RelPath string
}

type panePathDto struct {
	Kind    string
	Id      string
	RelPath string
}

// renderDualModePage renders the dual mode page.
// It takes an instance of pocketbase.PocketBase and an echo.Context as parameters.
// It returns an error if there is any issue during the rendering process.
//
// The function first decorates the context with title and description keys.
// Then it initializes a contentDto variable of type dto.DualViewDto.
// It calls the renderPane function twice to render the left and right panes.
// If there is an error during the rendering process, it logs the error and returns a server fault error.
// It retrieves the artist name list using the GetArtistNameList function from the artworks package.
// If there is an error getting the artist name list, it logs the error and returns a server fault error.
// It formats the artist name list using the formatArtistNameList function.
// It generates the relative path for the dual mode URL using the GenerateDualModeUrl function from the url package.
// It adds query parameters to the relative path for left pane, right pane, left render to, and right render to.
// It sets the "HX-Push-Url" header of the response with the generated relative path.
// It renders the dual page using the contentDto and writes the response to the context's writer.
// If there is an error during the rendering process, it logs the error and returns a server fault error.
// Finally, it returns nil if the rendering process is successful.
func renderDualModePage(app *pocketbase.PocketBase, c *core.RequestEvent) error {
	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Dual View")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "On this page you can search for artworks by title, artist, art form, art type and art school!")
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
	artistNameList, err := artworks.GetArtistNameList(app)

	if err != nil {
		app.Logger().Error("Error getting artist name list", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	contentDto.ArtistNameList = formatArtistNameList(artistNameList)

	c.Response.Header().Set("HX-Push-Url", buildDualModePushURL(leftPane, rightPane))

	var buff bytes.Buffer

	err = pages.DualPage(contentDto).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(200, buff.String())
}

func buildDualModePushURL(leftPane renderPaneDto, rightPane renderPaneDto) string {
	relPath := url.GenerateDualModeUrl()
	queryValues := relPath.Query()

	queryValues.Set("left", leftPane.RelPath)
	queryValues.Set("right", rightPane.RelPath)
	queryValues.Set("left_render_to", rightPane.Side)
	queryValues.Set("right_render_to", leftPane.Side)

	relPath.RawQuery = queryValues.Encode()

	return relPath.String()
}

func buildDualModePaneURL(side string, currentRelPath string, destinationRelPath string, queryValues map[string][]string) string {
	return buildDualModeTargetPaneURL(side, currentRelPath, destinationRelPath, "right", queryValues)
}

func buildDualModeOppositePaneURL(side string, currentRelPath string, destinationRelPath string, queryValues map[string][]string) string {
	targetSide := cmp.Or(strings.TrimSpace(firstQueryValue(queryValues, side+"_render_to")), reverseSide(side))
	if targetSide == "" {
		targetSide = reverseSide(side)
	}

	return buildDualModeTargetPaneURL(side, currentRelPath, destinationRelPath, targetSide, queryValues)
}

func buildDualModeTargetPaneURL(side string, currentRelPath string, destinationRelPath string, targetSide string, queryValues map[string][]string) string {
	relPath := url.GenerateDualModeUrl()
	nextQueryValues := make(map[string][]string, len(queryValues))
	for key, values := range queryValues {
		nextQueryValues[key] = append([]string(nil), values...)
	}

	setQueryValue(nextQueryValues, side, currentRelPath)
	setQueryValue(nextQueryValues, targetSide, destinationRelPath)
	setQueryValue(nextQueryValues, "left_render_to", "right")
	setQueryValue(nextQueryValues, "right_render_to", "left")

	encodedQuery := relPath.Query()
	for key, values := range nextQueryValues {
		for _, value := range values {
			encodedQuery.Add(key, value)
		}
	}

	relPath.RawQuery = encodedQuery.Encode()

	return relPath.String()
}

func firstQueryValue(queryValues map[string][]string, key string) string {
	if len(queryValues[key]) == 0 {
		return ""
	}

	return queryValues[key][0]
}

func setQueryValue(queryValues map[string][]string, key string, value string) {
	queryValues[key] = []string{value}
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

// formatArtistNameList formats the artist name list.
// It takes a map of artist names as a parameter.
// It returns a slice of dto.ArtistNameListEntry.
func formatArtistNameList(artistNameList map[string]string) []dto.ArtistNameListEntry {
	artistNameListEntries := make([]dto.ArtistNameListEntry, 0)

	for url, label := range artistNameList {
		artistNameListEntries = append(artistNameListEntries, dto.ArtistNameListEntry{
			Url:   url,
			Label: label,
		})
	}

	slices.SortFunc(artistNameListEntries, func(a dto.ArtistNameListEntry, b dto.ArtistNameListEntry) int {
		if diff := strings.Compare(a.Label, b.Label); diff != 0 {
			return diff
		}

		return strings.Compare(a.Url, b.Url)
	})

	return artistNameListEntries
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
	renderTo := cmp.Or(rawQParams.Get(side+"_render_to"), reverseSide(side))

	pane := renderPaneDto{
		Side: side,
	}
	buf := new(bytes.Buffer)

	parsedPath, err := parsePanePath(queryParam)
	if err != nil {
		return pane, err
	}

	if parsedPath.Kind == "default" {
		defaultContent, defaultErr := defaultPaneContent(side)
		if defaultErr != nil {
			return pane, defaultErr
		}

		return renderPaneDto{
			Side:    side,
			Content: defaultContent,
			RelPath: parsedPath.RelPath,
		}, nil
	}

	switch parsedPath.Kind {
	case "artist":
		artistDto, renderErr := renderArtistPane(app, c, side, parsedPath.RelPath, parsedPath.Id, renderTo, buf)
		if renderErr != nil {
			return pane, renderErr
		}

		pane.RelPath = resolvePaneRelPath(parsedPath.RelPath, artistDto.Url)
	case "artwork":
		artworkDto, renderErr := renderArtworkPane(app, c, side, parsedPath.RelPath, parsedPath.Id, renderTo, buf)
		if renderErr != nil {
			return pane, renderErr
		}

		pane.RelPath = resolvePaneRelPath(parsedPath.RelPath, artworkDto.Url)
	default:
		return pane, errs.ErrUnsupportedPaneType
	}

	pane.Content = buf.String()

	return pane, nil
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

func renderArtistPane(app *pocketbase.PocketBase, c *core.RequestEvent, side string, currentRelPath string, artistId string, renderTo string, buf *bytes.Buffer) (dto.Artist, error) {
	artistModel, err := app.FindRecordById(constants.CollectionArtists, artistId)

	if err != nil {
		app.Logger().Error("Error finding artist", "error", err.Error())
		return dto.Artist{}, err
	}

	artistDto, err := artists.RenderArtistContent(app, c, artistModel, "#"+renderTo, false)

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

func renderArtworkPane(app *pocketbase.PocketBase, c *core.RequestEvent, side string, currentRelPath string, artworkId string, renderTo string, buf *bytes.Buffer) (dto.Artwork, error) {
	artworkModel, err := app.FindRecordById(constants.CollectionArtworks, artworkId)

	if err != nil {
		app.Logger().Error("Error finding artwork", "error", err.Error())
		return dto.Artwork{}, err
	}

	artworkDto, err := artists.RenderArtworkContent(app, c, artworkModel, "#"+renderTo, false)

	if err != nil {
		app.Logger().Error("Error rendering artwork content", "error", err.Error())
		return dto.Artwork{}, err
	}

	if c.Request != nil && c.Request.URL != nil && c.Request.URL.Path == "/dual-mode" {
		currentQueryValues := c.Request.URL.Query()
		artworkDto.Url = buildDualModeOppositePaneURL(side, currentRelPath, artworkDto.Url, currentQueryValues)
		if artworkDto.Artist.Url != "" {
			artworkDto.Artist.Url = buildDualModeOppositePaneURL(side, currentRelPath, artworkDto.Artist.Url, currentQueryValues)
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
		return se.Next()
	})
}
