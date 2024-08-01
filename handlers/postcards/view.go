package postcards

import (
	"context"
	"fmt"

	"github.com/blackfyre/wga/assets/templ/pages"
	tmplUtils "github.com/blackfyre/wga/assets/templ/utils"
	"github.com/blackfyre/wga/utils"
	"github.com/blackfyre/wga/utils/url"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

func viewPostcard(app *pocketbase.PocketBase, c echo.Context) error {
	postCardId := c.QueryParamDefault("p", "nope")

	if postCardId == "nope" {
		app.Logger().Error(fmt.Sprintf("Invalid postcard id: %s", postCardId))
		return utils.NotFoundError(c)
	}

	r, err := app.Dao().FindRecordById("Postcards", postCardId)

	if err != nil {
		app.Logger().Error("Failed to find postcard", "id", postCardId, "error", err.Error())
		return utils.NotFoundError(c)
	}

	if errs := app.Dao().ExpandRecord(r, []string{"image_id"}, nil); len(errs) > 0 {
		app.Logger().Error("Failed to expand record", "id", postCardId, "errors", errs)
		return utils.ServerFaultError(c)
	}

	aw := r.ExpandedOne("image_id")

	content := pages.PostcardView{
		SenderName: r.GetString("sender_name"),
		Message:    r.GetString("message"),
		Image:      url.GenerateFileUrl("artworks", aw.GetString("id"), aw.GetString("image"), ""),
		Title:      aw.GetString("title"),
		Comment:    aw.GetString("comment"),
		Technique:  aw.GetString("technique"),
	}

	ctx := tmplUtils.DecorateContext(context.Background(), tmplUtils.TitleKey, "Postcard")

	//c.Response().Header().Set("HX-Push-Url", fullUrl)
	err = pages.PostcardPage(content).Render(ctx, c.Response().Writer)

	if err != nil {
		app.Logger().Error("Error rendering artwork page", "error", err.Error())
		return utils.ServerFaultError(c)
	}

	return nil
}
