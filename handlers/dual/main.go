package dual

import (
	"bytes"
	"cmp"
	"context"
	"strings"

	"github.com/blackfyre/wga/assets/templ/dto"
	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/errs"
	"github.com/blackfyre/wga/handlers/artist"
	"github.com/blackfyre/wga/handlers/artworks"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type renderPaneDto struct {
	Side    string
	Content string
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
	ArtistNameList, err := artworks.GetArtistNameList(app)

	if err != nil {
		app.Logger().Error("Error getting artist name list", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	contentDto.ArtistNameList = formatArtistNameList(ArtistNameList)

	relPath := url.GenerateDualModeUrl()

	relPath.Query().Add("left", leftPane.RelPath)
	relPath.Query().Add("right", rightPane.RelPath)
	relPath.Query().Add("left_render_to", rightPane.Side)
	relPath.Query().Add("right_render_to", leftPane.Side)

	c.Response.Header().Set("HX-Push-Url", relPath.String())

	var buff bytes.Buffer

	err = pages.DualPage(contentDto).Render(ctx, &buff)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return c.HTML(200, buff.String())
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

	return artistNameListEntries
}

// reverseSide returns the opposite side of the given side.
func reverseSide(side string) string {
	if side == "left" {
		return "right"
	} else if side == "right" {
		return "left"
	}

	return ""
}

func renderPane(side string, app *pocketbase.PocketBase, c *core.RequestEvent) (renderPaneDto, error) {

	rawQParams := c.Request.URL.Query()

	queryParam := cmp.Or(rawQParams.Get(side), "default")
	renderTo := cmp.Or(rawQParams.Get(side+"_render_to"), reverseSide(side))

	pane := renderPaneDto{
		Side: side,
	}
	var paneContent string
	buf := new(bytes.Buffer)

	if queryParam == "default" {
		if side == "left" {
			paneContent = "Left pane"
		} else if side == "right" {
			pages.RightSideDefault().Render(context.Background(), buf)
		} else {
			return pane, errs.ErrUnsupportedPaneType
		}

		paneContent = buf.String()

		return renderPaneDto{
			Side:    side,
			Content: paneContent,
		}, nil
	}

	// remove any leading / or trailing /
	queryParam = strings.Trim(queryParam, "/")

	// split the query param into parts
	parts := strings.Split(queryParam, "/")

	if len(parts) > 3 {
		return pane, errs.ErrTooManyParts
	}

	paneType := parts[0]

	switch paneType {
	case "artists":
		if len(parts) == 3 {
			slug := parts[2]
			artworkId := utils.ExtractIdFromString(slug)

			artworkDto, err := renderArtworkPane(app, c, artworkId, renderTo, buf)

			if err != nil {
				return pane, err
			}

			pane.RelPath = artworkDto.Url

		} else {
			slug := parts[1]
			artistId := utils.ExtractIdFromString(slug)

			artistDto, err := renderArtistPane(app, c, artistId, renderTo, buf)

			if err != nil {
				return pane, err
			}

			pane.RelPath = artistDto.Url
		}

	case "artworks":
		paneContent = "Artworks"
	default:
		return pane, errs.ErrUnknownDualPane
	}

	pane.Content = buf.String()

	return pane, nil
}

func renderArtistPane(app *pocketbase.PocketBase, c *core.RequestEvent, artistId string, renderTo string, buf *bytes.Buffer) (dto.Artist, error) {
	artistModel, err := app.FindRecordById("artists", artistId)

	if err != nil {
		app.Logger().Error("Error finding artist", "error", err.Error())
		return dto.Artist{}, err
	}

	artistDto, err := artist.RenderArtistContent(app, c, artistModel, "#"+renderTo)

	if err != nil {
		app.Logger().Error("Error rendering artist content", "error", err.Error())
		return dto.Artist{}, err
	}

	err = pages.ArtistBlock(artistDto).Render(context.Background(), buf)

	if err != nil {
		app.Logger().Error("Error rendering artist page", "error", err.Error())
		return artistDto, err
	}

	return artistDto, nil
}

func renderArtworkPane(app *pocketbase.PocketBase, c *core.RequestEvent, artworkId string, renderTo string, buf *bytes.Buffer) (dto.Artwork, error) {
	artworkModel, err := app.FindRecordById("artworks", artworkId)

	if err != nil {
		app.Logger().Error("Error finding artwork", "error", err.Error())
		return dto.Artwork{}, err
	}

	artworkDto, err := artist.RenderArtworkContent(app, c, artworkModel, "#"+renderTo)

	if err != nil {
		app.Logger().Error("Error rendering artwork content", "error", err.Error())
		return dto.Artwork{}, err
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
