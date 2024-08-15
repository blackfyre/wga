package dual

import (
	"bytes"
	"context"
	"strings"

	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/errs"
	"github.com/blackfyre/wga/handlers/artist"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func renderDualModePage(app *pocketbase.PocketBase, c echo.Context) error {
	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Dual View")
	ctx = tmplUtils.DecorateContext(ctx, tmplUtils.DescriptionKey, "On this page you can search for artworks by title, artist, art form, art type and art school!")
	// ctx = tmplUtils.DecorateContext(ctx, tmplUtils.OgUrlKey, pHtmxUrl)

	contentDto := pages.DualViewDto{}

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

	contentDto.Left = leftPane
	contentDto.Right = rightPane

	// c.Response().Header().Set("HX-Push-Url", pHtmxUrl)
	err = pages.DualPage(contentDto).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artwork search page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil
}

func renderPane(side string, app *pocketbase.PocketBase, c echo.Context) (string, error) {

	queryParam := c.QueryParamDefault(side, "default")

	var paneContent string
	buf := new(bytes.Buffer)

	if queryParam == "default" {
		if side == "left" {
			paneContent = "Left pane"
		} else if side == "right" {
			pages.RightSideDefault().Render(context.Background(), buf)
		} else {
			return "", errs.ErrUnknownDualPane
		}

		paneContent = buf.String()

		return paneContent, nil
	}

	// remove any leading / or trailing /
	queryParam = strings.Trim(queryParam, "/")

	// split the query param into parts
	parts := strings.Split(queryParam, "/")

	if len(parts) < 2 {
		return "", errs.ErrTooManyParts
	}

	paneType := parts[0]

	switch paneType {
	case "artists":
		slug := parts[1]
		artistId := utils.ExtractIdFromString(slug)

		artistModel, err := app.Dao().FindRecordById("artists", artistId)

		if err != nil {
			app.Logger().Error("Error finding artist", "error", err.Error())
			return "", err
		}

		artistDto, err := artist.RenderArtistContent(app, c, artistModel)

		if err != nil {
			app.Logger().Error("Error rendering artist content", "error", err.Error())
			return "", err
		}

		err = pages.ArtistBlock(artistDto).Render(context.Background(), buf)

		if err != nil {
			app.Logger().Error("Error rendering artist page", "error", err.Error())
			return "", err
		}
	case "artworks":
		paneContent = "Artworks"
	default:
		return "", errs.ErrUnsupportedPaneType

	}

	paneContent = buf.String()

	return paneContent, nil
}

func RegisterHandlers(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/dual-mode", func(c echo.Context) error {
			return renderDualModePage(app, c)
		})
		return nil
	})
}
