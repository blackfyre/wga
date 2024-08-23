package dual

import (
	"bytes"
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
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type renderPaneDto struct {
	Side    string
	Content string
	RelPath string
}

func renderDualModePage(app *pocketbase.PocketBase, c echo.Context) error {
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
	contentDto.ArtistNameList, err = artworks.GetArtistNameList(app)

	if err != nil {
		app.Logger().Error("Error getting artist name list", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	relPath := url.GenerateDualModeUrl()

	relPath.Query().Add("left", leftPane.RelPath)
	relPath.Query().Add("right", rightPane.RelPath)
	relPath.Query().Add("left_render_to", rightPane.Side)
	relPath.Query().Add("right_render_to", leftPane.Side)

	c.Response().Header().Set("HX-Push-Url", relPath.String())
	err = pages.DualPage(contentDto).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil
}

func reverseSide(side string) string {
	if side == "left" {
		return "right"
	} else if side == "right" {
		return "left"
	}

	return ""
}

func renderPane(side string, app *pocketbase.PocketBase, c echo.Context) (renderPaneDto, error) {

	queryParam := c.QueryParamDefault(side, "default")
	renderTo := c.QueryParamDefault(side+"_render_to", reverseSide(side))

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

	if len(parts) < 2 {
		return pane, errs.ErrTooManyParts
	}

	paneType := parts[0]

	switch paneType {
	case "artists":
		slug := parts[1]
		artistId := utils.ExtractIdFromString(slug)

		artistModel, err := app.Dao().FindRecordById("artists", artistId)

		if err != nil {
			app.Logger().Error("Error finding artist", "error", err.Error())
			return pane, err
		}

		artistDto, err := artist.RenderArtistContent(app, c, artistModel, "#"+renderTo)

		if err != nil {
			app.Logger().Error("Error rendering artist content", "error", err.Error())
			return pane, err
		}

		err = pages.ArtistBlock(artistDto).Render(context.Background(), buf)

		if err != nil {
			app.Logger().Error("Error rendering artist page", "error", err.Error())
			return pane, err
		}

		pane.RelPath = artistDto.Url
	case "artworks":
		paneContent = "Artworks"
	default:
		return pane, errs.ErrUnknownDualPane

	}

	pane.Content = buf.String()

	return pane, nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/dual-mode", func(c echo.Context) error {
			return renderDualModePage(app, c)
		})
		return nil
	})
}
